package main

import (
	//	"net/http"
	//	_ "net/http/pprof"
	//	"log"

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
	}
)

func init() {
	rootCmd.AddGroup(
		&cobra.Group{"commands", "Commands:"},
	)
}

func main() {
	//	go func() {
	//		log.Println(http.ListenAndServe("localhost:6060", nil))
	//	}()
	rootCmd.Execute()
}
