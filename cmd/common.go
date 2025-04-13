package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/build"
	"github.com/lone-faerie/mqttop/log"
	"github.com/spf13/cobra"
)

func findConfig() {
	const defaultConfigFile = "mqttop.yaml"

	if len(ConfigPath) > 0 {
		return
	}

	if env, ok := os.LookupEnv("MQTTOP_CONFIG_PATH"); ok {
		ConfigPath = strings.Split(env, ",")
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

func findData() {
	const defaultDataDir = "mqttop"

	if DataPath != "" {
		return
	}

	if env, ok := os.LookupEnv("MQTTOP_DATA_PATH"); ok {
		DataPath = env
		return
	}

	if xdg, ok := os.LookupEnv("XDG_DATA_HOME"); ok {
		DataPath = filepath.Join(xdg, defaultDataDir)
		return
	}

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	DataPath = filepath.Join(home, ".local", "share", defaultDataDir)
}

const banner = `┌────────────────────────────────────────────────────────────┐
│                                                            │
│   ███╗   ███╗ ██████╗ ████████╗████████╗ ██████╗ ██████╗   │
│   ████╗ ████║██╔═══██╗╚══██╔══╝╚══██╔══╝██╔═══██╗██╔══██╗  │
│   ██╔████╔██║██║   ██║   ██║      ██║   ██║   ██║██████╔╝  │
│   ██║╚██╔╝██║██║▄▄ ██║   ██║      ██║   ██║   ██║██╔═══╝   │
│   ██║ ╚═╝ ██║╚██████╔╝   ██║      ██║   ╚██████╔╝██║       │
│   ╚═╝     ╚═╝ ╚══▀▀═╝    ╚═╝      ╚═╝    ╚═════╝ ╚═╝       │
│                                                            │
│     Author: lone-faerie                                    │
│                                                            │
│     Version: {{printf "%%-18.18s" .Version}}                            │
│     Build Time: %-26.26s                 │
│                                                            │
└────────────────────────────────────────────────────────────┘
`

// BannerTemplate returns the string used for templating the banner.
func BannerTemplate() string {
	return fmt.Sprintf(banner, build.BuildTime())
}

// PrintBanner prints the banner to the given commands output.
func PrintBanner(cmd *cobra.Command) error {
	t := template.New("banner")

	template.Must(t.Parse(BannerTemplate()))

	return t.Execute(cmd.OutOrStdout(), cmd.Root())
}

const fullDocsFooter = `Full documentation is available at:
https://pkg.go.dev/github.com/lone-faerie/mqttop`

// ExitError is an error that should cause the program to exit with the given code.
type ExitError struct {
	Err  error
	Code int
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func maybeWithPort(addr string, port int) string {
	var hasPort bool

	if last := addr[len(addr)-1]; '0' <= last && last <= '9' {
		for _, c := range addr {
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

	if hasPort || port < 0 {
		return addr
	}

	return addr + ":" + strconv.Itoa(port)
}

func flagsToConfig(cfg *config.Config, args []string) error {
	if LogLevel != "" {
		var level log.Level

		if err := level.UnmarshalText([]byte(LogLevel)); err != nil {
			return err
		}

		cfg.Log.Level = level
	}

	if Broker != "" {
		cfg.MQTT.Broker = maybeWithPort(Broker, Port)
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
	case "text":
		if w == nil {
			w = os.Stderr
		}

		log.SetTextHandler(w)
	default:
		if w != nil {
			log.SetOutput(w)
		}
	}
}
