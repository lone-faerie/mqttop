//go:build docgen

package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var DocGenCommand = &cobra.Command{
	Use:    "docgen",
	Short:  "Generate documentation",
	Hidden: true,
}

var ManDocGenCommand = &cobra.Command{
	Use:   "man",
	Short: "Generate man pages",
	RunE: func(_ *cobra.Command, _ []string) error {
		hdr := &doc.GenManHeader{
			Title:   "MQTTOP",
			Section: "3",
		}
		if err := os.MkdirAll("docs/man", 0750); err != nil {
			return err
		}
		return doc.GenManTree(RootCommand, hdr, "docs/man")
	},
}

func init() {
	DocGenCommand.AddCommand(ManDocGenCommand)
	RootCommand.AddCommand(DocGenCommand)
}
