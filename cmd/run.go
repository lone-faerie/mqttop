package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop"
	"github.com/lone-faerie/mqttop/config"
	//	"github.com/lone-faerie/mqttop/internal/build"
	"github.com/lone-faerie/mqttop/log"
)

var (
	cfgPath []string
	cfg     *config.Config
)

var (
	broker    string
	port      int
	username  string
	password  string
	interval  time.Duration
	discovery string
	logLevel  string
)

var (
	runCmd = &cobra.Command{
		Use:     "run [-c config]... [flags] [metric]...",
		Aliases: []string{"start"},
		Short:   "Run bridge to provide system metrics to the MQTT broker",
		GroupID: "commands",
		ValidArgs: []cobra.Completion{
			cobra.CompletionWithDesc("all", "all metrics"),
			"cpu", "memory", "disks", "net", "battery", "dirs", "gpu",
		},
		Args: cobra.OnlyValidArgs,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if err = printBanner(cmd); err != nil {
				cmd.Println(err)
				return
			}

			initConfig()
			cfg, err = config.Load(cfgPath...)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return
			}
			if err = flagsToConfig(cfg, cmd, args); err != nil {
				return
			}
			log.Info("Config loaded")
			setLogHandler(cfg, log.LevelDebug)
			return
		},
		RunE: runBridge,

		DisableFlagsInUseLine: true,
	}
)

func init() {
	runCmd.Flags().SortFlags = false
	runCmd.Flags().StringSliceVarP(&cfgPath, "config", "c", nil, "Path(s) to config file/directory (default is first of $MQTTOP_CONFIG_FILE, $XDG_CONFIG_HOME/mqttop.yaml, $HOME/.config/mqttop.yaml)")
	runCmd.Flags().StringVarP(&broker, "broker", "b", "", "MQTT broker address")
	runCmd.Flags().IntVarP(&port, "port", "p", 1883, "MQTT broker port")
	runCmd.Flags().StringVar(&username, "username", "", "MQTT client username")
	runCmd.Flags().StringVar(&password, "password", "", "MQTT client password")
	runCmd.Flags().DurationVarP(&interval, "interval", "i", 0, "Update interval")
	runCmd.Flags().StringVarP(&discovery, "discovery", "d", "", "Discovery prefix, or 'disabled' to disable")
	runCmd.Flags().StringVarP(&logLevel, "log", "l", "", "Log level")

	rootCmd.AddCommand(runCmd)
}

func initConfig() {
	const defaultConfigFile = "mqttop.yaml"

	if len(cfgPath) > 0 {
		return
	}
	cfgFile, ok := os.LookupEnv("MQTTOP_CONFIG_PATH")
	if ok {
		cfgPath = strings.Split(cfgFile, ",")
		return
	}
	if xdg, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		cfgPath = []string{filepath.Join(xdg, defaultConfigFile)}
		return
	}
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	cfgPath = []string{filepath.Join(home, ".config", defaultConfigFile)}
}

func flagsToConfig(cfg *config.Config, cmd *cobra.Command, args []string) error {
	if logLevel != "" {
		var level log.Level
		if err := level.UnmarshalText([]byte(logLevel)); err != nil {
			return err
		}
		cfg.Log.Level = level
	}
	if broker != "" {
		var hasPort bool
		if last := broker[len(broker)-1]; '0' <= last && last <= '9' {
			for _, c := range broker {
				switch {
				case c == ':':
					hasPort = true
				case '0' <= c && c <= '9':
					hasPort = hasPort && true
				default:
					hasPort = false
				}
			}
		}
		if !hasPort && port >= 0 {
			broker += ":" + strconv.Itoa(port)
		}
		cfg.MQTT.Broker = broker
	}
	if username != "" {
		cfg.MQTT.Username = username
	}
	if password != "" {
		cfg.MQTT.Password = password
	}
	if interval > 0 {
		cfg.SetInterval(interval)
	}
	if discovery == "disabled" {
		cfg.Discovery.Enabled = false
	} else if discovery != "" {
		cfg.Discovery.Prefix = discovery
	}
	if len(args) > 0 {
		cfg.SetMetrics(args...)
	}
	return nil
}

func setLogHandler(cfg *config.Config, minLevel log.Level) {
	var w io.Writer
	switch strings.ToLower(cfg.Log.Output) {
	case "", "stderr":
	case "stdout":
		w = os.Stdout
	case "discard":
		log.SetHandler(log.DiscardHandler)
		return
	default:
		f, err := os.Open(cfg.Log.Output)
		if err != nil {
			log.Error(
				"Unable to open log file, deferring to stderr",
				err,
			)
			return
		}
		w = f
		cleanup = append(cleanup, func() { f.Close() })
	}
	if cfg.Log.Level < minLevel {
		cfg.Log.Level = minLevel
	}
	log.SetLogLevel(cfg.Log.Level)
	switch strings.ToLower(cfg.Log.Format) {
	case "json":
		if w == nil {
			w = os.Stderr
		}
		log.SetJSONHandler(w)
	default:
		if w != nil {
			log.SetOutput(w)
		}
	}
	return
}

func printBanner(cmd *cobra.Command) error {
	t := template.New("banner")
	template.Must(t.Parse(bannerTemplate()))
	w := cmd.OutOrStdout()
	err := t.Execute(w, cmd.Root())
	/*	if s, ok := w.(interface{ Sync() error }); ok {
		s.Sync()
	}*/
	return err
}

func runBridge(cmd *cobra.Command, _ []string) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)

	bridge := mqttop.New(cfg)
	if err := bridge.Connect(ctx); err != nil {
		log.Error("Not connected.", err)
		return &exitErr{err, 1}
	}
	defer func() {
		cancel()
		bridge.Disconnect()
		log.Info("Done")
	}()

	bridge.Start(ctx)
	select {
	case <-bridge.Ready():
		if cfg.Discovery.Enabled {
			bridge.Discover(ctx)
		}
	case <-c:
		return nil
	}
	cfg = nil

	<-c
	log.Debug("Received signal")
	return nil
}
