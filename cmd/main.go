package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lone-faerie/mqttop/internal/build"
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

func main() {
	if c, err := RootCommand.ExecuteC(); err != nil {
		if exit, ok := err.(*ExitError); ok {
			os.Exit(exit.Code)
		}

		c.PrintErrln("Error:", err)
		c.Usage()
	}
}
