package main

import (
	"fmt"
	"os"

	"github.com/lone-faerie/mqttop/internal/build"
	"github.com/spf13/cobra"
)

var cleanup []func()

// RootCommand is the root [cobra.Command] of the program.
var RootCommand = &cobra.Command{
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

func init() {
	cobra.EnableCommandSorting = false
	RootCommand.SetVersionTemplate(BannerTemplate())
	RootCommand.AddGroup(
		&cobra.Group{"commands", "Commands:"},
	)
}

// AddCleanup adds function(s) to be run as part of the PersistentPostRun of
// [RootCommand]
func AddCleanup(f ...func()) {
	cleanup = append(cleanup, f...)
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

// ExitErr is an error that should cause the program to exit with the given code.
type ExitErr struct {
	Err  error
	Code int
}

func (e *ExitErr) Error() string {
	return e.Err.Error()
}

func main() {
	if c, err := RootCommand.ExecuteC(); err != nil {
		if exit, ok := err.(*ExitErr); ok {
			os.Exit(exit.Code)
		}
		c.PrintErrln("Error:", err)
		c.Usage()
	}
}
