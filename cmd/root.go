/*
A bridge to provide system metrics over MQTT.

Full documentation is available at:
https://pkg.go.dev/github.com/lone-faerie/mqttop

Usage:

	mqttop [command]

Commands:

	run
		Run the metrics bridge

Additional Commands:

	list
		List available metrics
	stop
		Stop running bridge
	help
		Help about any command

Flags:

	-h, --help
		help for mqttop
	-v, --version
		version for mqttop

Use "mqttop [command] --help" for more information about a command.

Full documentation is available at:
https://pkg.go.dev/github.com/lone-faerie/mqttop
*/
package main

import (
	"github.com/lone-faerie/mqttop/internal/build"
	"github.com/spf13/cobra"
)

var cleanup []func()

// RootCommand is the root [cobra.Command] of the program.
var RootCommand = &cobra.Command{
	Use:     "mqttop",
	Short:   "A bridge to provide system metrics over MQTT.",
	Long:    `A bridge to provide system metrics over MQTT.`,
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
	RootCommand.SetHelpTemplate(RootCommand.HelpTemplate() + "\n" + fullDocsFooter + "\n")
	RootCommand.AddGroup(
		&cobra.Group{ID: "commands", Title: "Commands:"},
	)
}

// AddCleanup adds function(s) to be run as part of the PersistentPostRun of
// [RootCommand]
func AddCleanup(f ...func()) {
	cleanup = append(cleanup, f...)
}
