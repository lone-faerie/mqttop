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

// Flags for [RunCommand]
var (
	ConfigPath []string      // Path(s) to config file/directory (default is first of $MQTTOP_CONFIG_FILE, $XDG_CONFIG_HOME/mqttop.yaml, $HOME/.config/mqttop.yaml)
	Broker     string        // MQTT broker address
	Port       int           // MQTT broker port
	Username   string        // MQTT broker username
	Password   string        // MQTT broker password
	Interval   time.Duration // Update interval
	Discovery  string        // Discovery prefix, or 'disabled' to disable
	LogLevel   string        // Log level
)

var cfg *config.Config

// RunCommand is the main [cobra.Command] used for running the bridge.
var RunCommand = &cobra.Command{
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
		if err = PrintBanner(cmd); err != nil {
			cmd.Println(err)
			return
		}

		initConfig()
		cfg, err = config.Load(ConfigPath...)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}
		if err = flagsToConfig(cfg, cmd, args); err != nil {
			return
		}
		log.Info("Config loaded")
		setLogHandler(cfg, log.LevelDebug)
		log.Debug("MQTT broker", "addr", cfg.MQTT.Broker)
		return
	},
	RunE: runBridge,

	DisableFlagsInUseLine: true,
}

func init() {
	RunCommand.Flags().SortFlags = false
	RunCommand.Flags().StringSliceVarP(&ConfigPath, "config", "c", nil, "Path(s) to config file/directory (default is first of $MQTTOP_CONFIG_FILE, $XDG_CONFIG_HOME/mqttop.yaml, $HOME/.config/mqttop.yaml)")
	RunCommand.Flags().StringVarP(&Broker, "broker", "b", "", "MQTT broker address")
	RunCommand.Flags().IntVarP(&Port, "port", "p", 1883, "MQTT broker port")
	RunCommand.Flags().StringVar(&Username, "username", "", "MQTT client username")
	RunCommand.Flags().StringVar(&Password, "password", "", "MQTT client password")
	RunCommand.Flags().DurationVarP(&Interval, "interval", "i", 0, "Update interval")
	RunCommand.Flags().StringVarP(&Discovery, "discovery", "d", "", "Discovery prefix, or 'disabled' to disable")
	RunCommand.Flags().StringVarP(&LogLevel, "log", "l", "", "Log level")

	RootCommand.AddCommand(RunCommand)
}

func initConfig() {
	const defaultConfigFile = "mqttop.yaml"

	if len(ConfigPath) > 0 {
		return
	}
	cfgFile, ok := os.LookupEnv("MQTTOP_CONFIG_PATH")
	if ok {
		ConfigPath = strings.Split(cfgFile, ",")
		return
	}
	if xdg, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		ConfigPath = []string{filepath.Join(xdg, defaultConfigFile)}
		return
	}
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	ConfigPath = []string{filepath.Join(home, ".config", defaultConfigFile)}
}

func flagsToConfig(cfg *config.Config, cmd *cobra.Command, args []string) error {
	if LogLevel != "" {
		var level log.Level
		if err := level.UnmarshalText([]byte(LogLevel)); err != nil {
			return err
		}
		cfg.Log.Level = level
	}
	if Broker != "" {
		var hasPort bool
		if last := Broker[len(Broker)-1]; '0' <= last && last <= '9' {
			for _, c := range Broker {
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
		if !hasPort && Port >= 0 {
			Broker += ":" + strconv.Itoa(Port)
		}
		cfg.MQTT.Broker = Broker
	}
	if Username != "" {
		cfg.MQTT.Username = Username
	}
	if Password != "" {
		cfg.MQTT.Password = Password
	}
	if Interval > 0 {
		cfg.SetInterval(Interval)
	}
	if Discovery == "disabled" {
		cfg.Discovery.Enabled = false
	} else if Discovery != "" {
		cfg.Discovery.Prefix = Discovery
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
		AddCleanup(func() { f.Close() })
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

// PrintBanner prints the banner to the given commands output.
func PrintBanner(cmd *cobra.Command) error {
	t := template.New("banner")
	template.Must(t.Parse(BannerTemplate()))
	w := cmd.OutOrStdout()
	err := t.Execute(w, cmd.Root())
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
		return &ExitErr{err, 1}
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
			discoveryPath := filepath.Join(filepath.Dir(ConfigPath[0]), "discovery.json")
			bridge.Discover(ctx, discoveryPath)
		}
	case <-c:
		return nil
	}
	cfg = nil

	<-c
	log.Debug("Received signal")
	return nil
}
