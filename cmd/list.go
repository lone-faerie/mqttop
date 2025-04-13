package cmd

import (
	_ "embed"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/metrics"
)

// Flags for mqttop list
var (
	ListSummary bool // Display a summary of available metrics
)

//go:embed help/list.md
var listHelp string

// NewCmdList returns the [cobra.Command] used for listing available metrics.
//
// If --config is specified, the config will be used to determine which metrics to include.
//
// If --summary is specified, the list will be a comma-separated list of metric types. Otherwise, the metrics will be printed with some basic information, i.e. CPU name, total memory, etc.
//
// Usage:
//
//	mqttop list [flags]
//
// Aliases:
//
//	list, l
//
// Flags:
//
//	-c, --config strings   Path(s) to config file/directory
//	-s, --summary          Display a summary of available metrics
//	-h, --help             help for list
func NewCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "List available metrics",
		Long:    listHelp,
		ValidArgs: []cobra.Completion{
			cobra.CompletionWithDesc("all", "all metrics"),
			"cpu", "memory", "disks", "net", "battery", "dirs", "gpu",
		},
		Args: cobra.OnlyValidArgs,
		RunE: listMetrics,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringSliceVarP(&ConfigPath, "config", "c", nil, "Path(s) to config file/directory")
	cmd.Flags().BoolVarP(&ListSummary, "summary", "s", false, "Display a summary of available metrics")

	cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.MarkFlagDirname("config")

	cmd.SetHelpTemplate(cmd.HelpTemplate() + "\n" + fullDocsFooter + "\n")

	return cmd
}

type byteWriter struct {
	w io.Writer
}

func (w *byteWriter) WriteByte(c byte) error {
	_, err := w.w.Write([]byte{c})
	return err
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

	if len(args) > 0 {
		cfg.SetMetrics(args...)
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
