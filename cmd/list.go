package main

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/metrics"
)

// Flags for [ListCommand]
var (
	ListSummary bool // Display a summary of available metrics
)

// ListCommand is the [cobra.Command] used for listing available metrics.
var ListCommand = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List available metrics",
	RunE:    listMetrics,
}

func init() {
	ListCommand.Flags().SortFlags = false
	ListCommand.Flags().StringSliceVarP(&ConfigPath, "config", "c", nil, "Path(s) to config file/directory")
	ListCommand.Flags().BoolVarP(&ListSummary, "summary", "s", false, "Display a summary of available metrics")

	RootCommand.AddCommand(ListCommand)
}

func printMetrics(w io.Writer, mm []metrics.Metric, args []string) {
	r := strings.NewReplacer("\n", "\n  ")
	for _, m := range mm {
		if len(args) > 0 && !slices.Contains(args, m.Type()) {
			continue
		}
		fmt.Fprintf(w, "[%s]\n  ", m.Type())
		r.WriteString(w, m.String())
		w.Write([]byte{'\n'})
	}
}

func printSummary(w io.Writer, mm []metrics.Metric, args []string) {
	for i, m := range mm {
		if len(args) > 0 && !slices.Contains(args, m.Type()) {
			continue
		}
		if i > 0 {
			w.Write([]byte{',', ' '})
		}
		w.Write([]byte(m.Type()))
		if d, ok := m.(*metrics.Dir); ok {
			fmt.Fprintf(w, " (%s)", d)
		}
	}
	w.Write([]byte{'\n'})
}

func listMetrics(cmd *cobra.Command, args []string) (err error) {
	log.SetLogLevel(log.LevelWarn)
	if len(ConfigPath) > 0 {
		cfg, err = config.Load(ConfigPath...)
		if err != nil {
			return
		}
		setLogHandler(cfg, log.LevelWarn)
	} else {
		cfg = config.Default()
	}
	mm := metrics.New(cfg)
	slices.SortFunc(mm, func(a, b metrics.Metric) int {
		return strings.Compare(a.Type(), b.Type())
	})
	// Nvidia GPU needs to be stopped, so we just stop all metrics when done
	AddCleanup(func() { metrics.Stop(mm...) })

	if ListSummary {
		printSummary(cmd.OutOrStdout(), mm, args)
	} else {
		printMetrics(cmd.OutOrStdout(), mm, args)
	}
	return nil
}
