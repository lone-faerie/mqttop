package main

import (
	"github.com/spf13/cobra"
	"github.com/lone-faerie/mqttop/internal/build"
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
	}
)

func init() {
	rootCmd.AddGroup(
		&cobra.Group{"commands", "Commands:"},
	)
}

func main() {
	rootCmd.Execute()
}
