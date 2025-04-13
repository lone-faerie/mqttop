package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop/bridge"
	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/discovery"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/metrics"
)

// Flags for mqttop run
var (
	ConfigPath []string      // Path(s) to config file/directory (default is first of $MQTTOP_CONFIG_PATH, $XDG_CONFIG_HOME/mqttop.yaml, $HOME/.config/mqttop.yaml)
	DataPath   string        // Path to data directory (default is first of $MQTTOP_DATA_PATH, $XDG_DATA_HOME/mqttop, $HOME/.local/share/mqttop)
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

//go:embed help/run.md
var runHelp string

// NewCmdRun returns the main [cobra.Command] used for running the bridge.
//
// A connection to the MQTT broker will be established and the bridge will run in the foreground until a signal is received.
//
//   - SIGINT or SIGTERM will gracefully shutdown the bridge.
//
// MQTTop can load configuration from multiple YAML files, including from directories. If no config file is specified, the default path(s) will be determined by the first defined value of $MQTTOP_CONFIG_PATH, $XDG_CONFIG_HOME/mqttop.yaml, or $HOME/.config/mqttop.yaml. In the case of $MQTTOP_CONFIG_PATH, the value may be a comma-separated list of paths. If none of these files exist, the default configuration will be used, which looks for the following environment variables:
//
//   - broker:   $MQTTOP_BROKER_ADDRESS
//   - username: $MQTTOP_BROKER_USERNAME
//   - password: $MQTTOP_BROKER_PASSWORD
//
// Enabled metrics may be supplied as arguments, which will ignore the enabled metrics of the config. The special argument 'all' may be supplied to enable all metrics. The valid arguments include:
//
//   - all, cpu, memory, disks, net, battery, dirs, gpu
//
// All of the flags, if specified, will override the equivalent values in the config. The format of --broker should be scheme://host:port Where "scheme" is one of "tcp", "ssl", or "ws", "host" is the ip-address (or hostname) and "port" is the port on which the broker is accepting connections. If "scheme" is not defined, it defaults to "tcp" and if "port" is not defined, it will use the value of --port (default 1883).
//
// Usage:
//
//	mqttop run [--config <path>]... [flags] [metric]...
//
// Aliases:
//
//	run, start
//
// Examples:
//
//	mqttop run --config config.yaml
//	mqttop run --config config.yaml cpu memory
//	mqttop run --broker 127.0.0.1:1883 --username mqttop --password p@55w0rd
//
// Flags:
//
//	-c, --config strings      Path(s) to config file/directory
//	-b, --broker string       MQTT broker address
//	-p, --port int            MQTT broker port (default 1883)
//	    --username string     MQTT client username
//	    --password string     MQTT client password
//	    --cert string         MQTT TLS certificate file (PEM encoded)
//	    --key string          MQTT TLS private key file (PEM encoded)
//	-i, --interval duration   Update interval
//	-D, --discovery string    Discovery prefix, or 'disabled' to disable
//	    --data string         Path to data directory
//	-l, --log string          Log level
//	-d, --detach              Run detached (in background)
//	-h, --help                help for run
func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run [--config <path>]... [flags] [metric]...",
		Aliases: []string{"start"},
		Short:   "Run the metrics bridge",
		Long:    runHelp,
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

			findConfig()
			findData()
			if DataPath != "" {
				err = os.MkdirAll(DataPath, 0660)
				if err != nil {
					return
				}
			}

			cfg, err = config.Load(ConfigPath...)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return
			}

			if err = flagsToConfig(cfg, args); err != nil {
				return
			}

			log.Info("Config loaded")
			setLogHandler(cfg, cfg.Log.Level)
			log.Debug("MQTT broker", "addr", cfg.MQTT.Broker)

			return
		},
		RunE: runBridge,

		DisableFlagsInUseLine: true,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringSliceVarP(&ConfigPath, "config", "c", nil, "Path(s) to config file/directory")
	cmd.Flags().StringVarP(&Broker, "broker", "b", "", "MQTT broker address")
	cmd.Flags().IntVarP(&Port, "port", "p", 1883, "MQTT broker port")
	cmd.Flags().StringVar(&Username, "username", "", "MQTT client username")
	cmd.Flags().StringVar(&Password, "password", "", "MQTT client password")
	cmd.Flags().StringVar(&CertFile, "cert", "", "MQTT TLS certificate file (PEM encoded)")
	cmd.Flags().StringVar(&KeyFile, "key", "", "MQTT TLS private key file (PEM encoded)")
	cmd.Flags().DurationVarP(&Interval, "interval", "i", 0, "Update interval")
	cmd.Flags().StringVarP(&Discovery, "discovery", "D", "", "Discovery prefix, or 'disabled' to disable")
	cmd.Flags().StringVar(&DataPath, "data", "", "Path to data directory")
	cmd.Flags().StringVarP(&LogLevel, "log", "l", "", "Log level")
	cmd.Flags().BoolVarP(&Detach, "detach", "d", false, "Run detached (in background)")
	cmd.Flags().String("pingback", "", "Pingback (hidden)")

	cmd.Flags().Lookup("pingback").Hidden = true

	cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.MarkFlagDirname("config")

	cmd.SetHelpTemplate(cmd.HelpTemplate() + "\n" + fullDocsFooter + "\n")

	return cmd
}

// Adapted from https://github.com/caddyserver/caddy/blob/master/cmd/commandfuncs.go#L44
func runDetached(cmd *cobra.Command, args []string) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	defer ln.Close()

	c := exec.Command(os.Args[0], "--pingback", ln.Addr().String())
	if errors.Is(c.Err, exec.ErrDot) {
		c.Err = nil
	}

	c.Args = append(c.Args, os.Args[1:]...)
	c.Args = slices.DeleteFunc(c.Args, func(s string) bool { return s == "-d" || s == "--detach" })

	stdin, err := c.StdinPipe()
	if err != nil {
		return err
	}

	expect := make([]byte, 32)
	if _, err = rand.Read(expect); err != nil {
		return err
	}

	go func() {
		_, _ = stdin.Write(expect)
		stdin.Close()
	}()

	if err = c.Start(); err != nil {
		return err
	}

	success, exit := make(chan struct{}), make(chan error)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}

			if err = handlePingbackConn(conn, expect); err == nil {
				close(success)
				break
			}
		}
	}()

	go func() {
		err := c.Wait()
		exit <- err
	}()

	select {
	case <-success:
		log.Info("started in background", "pid", c.Process.Pid)
	case err := <-exit:
		return err
	}

	return nil
}

func handlePingbackConn(conn net.Conn, expect []byte) error {
	defer conn.Close()

	confirmationBytes, err := io.ReadAll(io.LimitReader(conn, 32))
	if err != nil {
		return err
	}

	if !bytes.Equal(confirmationBytes, expect) {
		return fmt.Errorf("wrong confirmation: %x", confirmationBytes)
	}

	return nil
}

func getDiscovery(mm []metrics.Metric) (d *discovery.Discovery, migrate bool, err error) {
	if d, err = discovery.New(&cfg.Discovery); err != nil {
		return
	}

	for _, m := range mm {
		if dd, ok := m.(discovery.Discoverer); ok {
			dd.Discover(d)
		}
	}

	var old *discovery.Discovery

	path := filepath.Join(filepath.Dir(ConfigPath[0]), "discovery.json")
	old, err = discovery.Load(path)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}

		return
	}

	migrate = d.Diff(old)

	return
}

func runBridge(cmd *cobra.Command, args []string) error {
	defer log.Info("Done")

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	m := metrics.New(cfg)
	defer metrics.Stop(m...)

	opts := []bridge.Option{
		bridge.WithMetrics(m...),
		bridge.WithLogLevel(cfg.MQTT.LogLevel),
	}

	if cfg.Discovery.Enabled {
		d, migrate, err := getDiscovery(m)
		if err == nil {
			opts = append(opts, bridge.WithDiscovery(d, migrate))
			AddCleanup(func() {
				log.Debug("Writing discovery")
				err := d.Write(filepath.Join(DataPath, "discovery.json"))
				log.Debug("Done writing discovery", "err", err)
			})
		}
	}

	b := bridge.New(cfg, opts...)

	if err := b.Start(ctx); err != nil {
		log.Error("Not connected.", err)
		return &ExitError{err, 1}
	}

	log.Debug("Connected")

	select {
	case <-b.Ready():
		if err := b.Error(); err != nil {
			return &ExitError{err, 1}
		}
	case <-ctx.Done():
		return nil
	}

	cfg = nil

	defer b.Stop()

	if pingback, _ := cmd.Flags().GetString("pingback"); pingback != "" {
		confirmationBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return &ExitError{err, 1}
		}

		conn, err := net.Dial("tcp", pingback)
		if err != nil {
			return &ExitError{err, 1}
		}

		_, err = conn.Write(confirmationBytes)
		conn.Close()

		if err != nil {
			return &ExitError{err, 1}
		}
	}

	select {
	case <-ctx.Done():
		log.Debug("Received signal")
	case <-b.Done():
	}

	return nil
}
