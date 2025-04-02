package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop"
	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
)

// Flags for [RunCommand]
var (
	ConfigPath []string      // Path(s) to config file/directory (default is first of $MQTTOP_CONFIG_FILE, $XDG_CONFIG_HOME/mqttop.yaml, $HOME/.config/mqttop.yaml)
	Broker     string        // MQTT broker address
	Port       int           // MQTT broker port
	Username   string        // MQTT broker username
	Password   string        // MQTT broker password
	CertFile   string        // MQTT TLS certificate file (PEM encoded)
	KeyFile    string        // MQTT TLS private key file (PEM encoded)
	Interval   time.Duration // Update interval
	Discovery  string        // Discovery prefix, or 'disabled' to disable
	LogLevel   string        // Log level
	Detach     bool          // Run detached (in background)
)

var cfg *config.Config

// RunCommand is the main [cobra.Command] used for running the bridge.
var RunCommand = &cobra.Command{
	Use:     "run [--config <path>]... [flags] [metric]...",
	Aliases: []string{"start"},
	Short:   "Run the metrics bridge",
	Long: `Run a bridge to provide system metrics to the MQTT broker.

A connection to the MQTT broker will be established and the bridge will run in the foreground until a signal is received.

	- SIGINT or SIGTERM will gracefully shutdown the bridge.

MQTTop can load configuration from multiple YAML files, including from directories. If no config file is specified, the default path(s) will be determined by the first defined value of $MQTTOP_CONFIG_PATH, $XDG_CONFIG_HOME/mqttop.yaml, or $HOME/.config/mqttop.yaml. In the case of $MQTTOP_CONFIG_PATH, the value may be a comma-separated list of paths. If none of these files exist, the default configuration will be used, which looks for the following environment variables:

	- broker:   $MQTTOP_BROKER_ADDRESS
	- username: $MQTTOP_BROKER_USERNAME
	- password: $MQTTOP_BROKER_PASSWORD

Enabled metrics may be supplied as arguments, which will ignore the enabled metrics of the config. The special argument 'all' may be supplied to enable all metrics. The valid arguments include:

	- all, cpu, memory, disks, net, battery, dirs, gpu

All of the flags, if specified, will override the equivalent values in the config. The format of --broker should be scheme://host:port Where "scheme" is one of "tcp", "ssl", or "ws", "host" is the ip-address (or hostname) and "port" is the port on which the broker is accepting connections. If "scheme" is not defined, it defaults to "tcp" and if "port" is not defined, it will use the value of --port (default 1883).`,
	Example: `  mqttop run --config config.yaml
  mqttop run --config config.yaml cpu memory
  mqttop run --broker 127.0.0.1:1883 --username mqttop --password p@55w0rd`,
	GroupID: "commands",
	ValidArgs: []cobra.Completion{
		cobra.CompletionWithDesc("all", "all metrics"),
		"cpu", "memory", "disks", "net", "battery", "dirs", "gpu",
	},
	Args: cobra.OnlyValidArgs,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if p, _ := cmd.Flags().GetString("pingback"); p != "" {
			log.Info("Pingback", "val", p)
		}
		if Detach {
			var code int
			if err = runDetached(cmd, args); err != nil {
				code = 1
			}
			return &ExitError{err, code}
		}

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
	RunCommand.Flags().StringSliceVarP(&ConfigPath, "config", "c", nil, "Path(s) to config file/directory")
	RunCommand.Flags().StringVarP(&Broker, "broker", "b", "", "MQTT broker address")
	RunCommand.Flags().IntVarP(&Port, "port", "p", 1883, "MQTT broker port")
	RunCommand.Flags().StringVar(&Username, "username", "", "MQTT client username")
	RunCommand.Flags().StringVar(&Password, "password", "", "MQTT client password")
	RunCommand.Flags().StringVar(&CertFile, "cert", "", "MQTT TLS certificate file (PEM encoded)")
	RunCommand.Flags().StringVar(&KeyFile, "key", "", "MQTT TLS private key file (PEM encoded)")
	RunCommand.Flags().DurationVarP(&Interval, "interval", "i", 0, "Update interval")
	RunCommand.Flags().StringVarP(&Discovery, "discovery", "D", "", "Discovery prefix, or 'disabled' to disable")
	RunCommand.Flags().StringVarP(&LogLevel, "log", "l", "", "Log level")
	RunCommand.Flags().BoolVarP(&Detach, "detach", "d", false, "Run detached (in background)")
	RunCommand.Flags().String("pingback", "", "Pingback (hidden)")
	RunCommand.Flags().Lookup("pingback").Hidden = true

	RunCommand.MarkFlagFilename("config", "yaml", "yml")
	RunCommand.MarkFlagDirname("config")

	RunCommand.SetHelpTemplate(RunCommand.HelpTemplate() + "\n" + fullDocsFooter + "\n")

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
	if CertFile != "" {
		cfg.MQTT.CertFile = CertFile
	}
	if KeyFile != "" {
		cfg.MQTT.KeyFile = KeyFile
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

func runDetached(cmd *cobra.Command, args []string) error {
	c := exec.Command(os.Args[0], os.Args[1:]...)
	if errors.Is(c.Err, exec.ErrDot) {
		c.Err = nil
	}
	c.Args = slices.DeleteFunc(c.Args, func(s string) bool { return s == "-d" || s == "--detach" })
	return c.Start()
}

func runBridge(cmd *cobra.Command, args []string) error {
	if Detach {
		return runDetached(cmd, args)
	}

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
		return &ExitError{err, 1}
	}
	defer func() {
		cancel()
		bridge.Disconnect()
		log.Info("Done")
	}()

	bridge.Start(ctx)
	select {
	case err := <-bridge.Ready():
		if err != nil {
			return &ExitError{err, 1}
		}
		if cfg.Discovery.Enabled {
			discoveryPath := filepath.Join(filepath.Dir(ConfigPath[0]), "discovery.json")
			bridge.Discover(ctx, discoveryPath)
		}
	case <-c:
		return nil
	}
	cfg = nil

	select {
	case <-bridge.Done():
	case <-c:
		log.Debug("Received signal")
	}
	return nil
}
