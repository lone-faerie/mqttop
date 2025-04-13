package cmd

import (
	"github.com/lone-faerie/mqttop/internal/build"
	"github.com/spf13/cobra"
)

var cleanup []func()

func init() {
	cobra.EnableCommandSorting = false
}

// NewCmdRoot returns the root [cobra.Command] of the program.
//
// Usage:
//
//	mqttop [command]
//
// Commands:
//
//	run         Run the metrics bridge
//
// Additional Commands:
//
//	stop        Stop running bridge
//	list        List available metrics
//	help        Help about any command
//
// Flags:
//
//	-h, --help      help for mqttop
//	-v, --version   version for mqttop
func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
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

	cmd.SetVersionTemplate(BannerTemplate())
	cmd.SetHelpTemplate(cmd.HelpTemplate() + "\n" + fullDocsFooter + "\n")
	cmd.AddGroup(
		&cobra.Group{ID: "commands", Title: "Commands:"},
	)

	cmd.AddCommand(NewCmdRun())
	cmd.AddCommand(NewCmdStop())
	cmd.AddCommand(NewCmdList())

	return cmd
}

// AddCleanup adds function(s) to be run as part of the PersistentPostRun of
// [RootCommand]
func AddCleanup(f ...func()) {
	cleanup = append(cleanup, f...)
}

var cmd *cobra.Command

func Execute() (err error) {
	cmd, err = NewCmdRoot().ExecuteC()
	return err
}

// Error calls [cobra.Command.PrintErrln] of the executed command.
func Error(err error) {
	if cmd == nil {
		return
	}
	cmd.PrintErrln("Error:", err)
}

// Usage calls [cobra.Command.Usage] of the executed command.
func Usage() error {
	if cmd == nil {
		return nil
	}
	return cmd.Usage()
}
