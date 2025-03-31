package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
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
	listCmd.Flags().StringSliceVarP(&cfgPath, "config", "c", nil, "Path(s) to config file/directory")

	rootCmd.AddCommand(listCmd)
}

func listMetrics(cmd *cobra.Command, args []string) {
	log.SetLogLevel(log.LevelWarn)
	if len(cfgPath) > 0 {
		cfg, _ = config.Load(cfgPath...)
		setLogHandler(cfg, log.LevelWarn)
	} else {
		cfg = config.Default()
	}
	mm := metrics.New(cfg)
	slices.SortFunc(mm, func(a, b metrics.Metric) int {
		return strings.Compare(a.Type(), b.Type())
	})
	cleanup = append(cleanup, func() { metrics.Stop(mm...) })

	summary, _ := cmd.Flags().GetBool("summary")

	w := cmd.OutOrStdout()
	var r *strings.Replacer
	if !summary {
		r = strings.NewReplacer("\n", "\n  ")
	}

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
			if d, ok := m.(*metrics.Dir); ok {
				fmt.Fprintf(w, " (%s)", d)
			}
		} else {
			if !first {
				w.Write([]byte{'\n'})
			}
			fmt.Fprintf(w, "[%s]\n  ", m.Type())
			r.WriteString(w, m.String())
		}
		first = false
	}
	w.Write([]byte{'\n'})
}
