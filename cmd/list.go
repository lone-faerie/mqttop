package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/metrics"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List available metrics",
	Run:     listMetrics,
}

func init() {
	listCmd.Flags().BoolP("summary", "s", false, "display a summary of available metrics")

	rootCmd.AddCommand(listCmd)
}

func listMetrics(cmd *cobra.Command, args []string) {
	cfg := config.Default()
	mm := metrics.New(cfg)
	slices.SortFunc(mm, func(a, b metrics.Metric) int {
		return strings.Compare(a.Type(), b.Type())
	})

	summary, _ := cmd.Flags().GetBool("summary")

	w := cmd.OutOrStdout()

	first := true
	for _, m := range mm {
		if len(args) > 0 && !slices.Contains(args, m.Type()) {
			continue
		}
		if summary {
			if !first {
				w.Write([]byte{',', ' '})
			}
			w.Write([]byte(m.Type()))
		} else {
			fmt.Fprintf(w, "[%s]\n%v\n", m.Type(), m)
		}
		first = false
	}
	if summary {
		w.Write([]byte{'\n'})
	}
}
