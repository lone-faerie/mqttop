package main

import (
	"fmt"
	"os"

	"github.com/lone-faerie/mqttop/internal/build"
	"github.com/spf13/cobra"
)

type CleanupFunc func() error

var cleanup []CleanupFunc

var (
	rootCmd = &cobra.Command{
		Use:     "mqttop [-c config]...",
		Short:   "Provide system metrics over MQTT",
		Version: build.Version(),
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			for _, f := range cleanup {
				f()
			}
		},
		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
		SilenceErrors:     true,
		SilenceUsage:      true,
	}
)

func init() {
	rootCmd.SetVersionTemplate(bannerTemplate())
	rootCmd.AddGroup(
		&cobra.Group{"commands", "Commands:"},
	)
}

const banner = `+────────────────────────────────────────────────────────────+
|                                                            |
|   ███╗   ███╗ ██████╗ ████████╗████████╗ ██████╗ ██████╗   |
|   ████╗ ████║██╔═══██╗╚══██╔══╝╚══██╔══╝██╔═══██╗██╔══██╗  |
|   ██╔████╔██║██║   ██║   ██║      ██║   ██║   ██║██████╔╝  |
|   ██║╚██╔╝██║██║▄▄ ██║   ██║      ██║   ██║   ██║██╔═══╝   |
|   ██║ ╚═╝ ██║╚██████╔╝   ██║      ██║   ╚██████╔╝██║       |
|   ╚═╝     ╚═╝ ╚══▀▀═╝    ╚═╝      ╚═╝    ╚═════╝ ╚═╝       |
|                                                            |
|       Version: %-18.18s                          |
|                                                            |
+────────────────────────────────────────────────────────────+
`

const bannerTmpl = `┌────────────────────────────────────────────────────────────┐
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
│     Version: {{printf "%%-12.12s" .Version}}                                  │
│     Build Time: %-26.26s                 │
│                                                            │
└────────────────────────────────────────────────────────────┘
`

func bannerTemplate() string {
	return fmt.Sprintf(bannerTmpl, build.BuildTime())
}

type exitErr struct {
	Err  error
	Code int
}

func (e *exitErr) Error() string {
	return e.Err.Error()
}

func main() {
	if c, err := rootCmd.ExecuteC(); err != nil {
		if exit, ok := err.(*exitErr); ok {
			os.Exit(exit.Code)
		}
		c.PrintErrln("Error:", err)
		c.Usage()
	}
}
