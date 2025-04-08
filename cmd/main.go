package main

import (
	"fmt"
	"os"

	"github.com/lone-faerie/mqttop/internal/build"
)

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

const fullDocsFooter = `Full documentation is available at:
https://pkg.go.dev/github.com/lone-faerie/mqttop`

// ExitError is an error that should cause the program to exit with the given code.
type ExitError struct {
	Err  error
	Code int
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func main() {
	if c, err := RootCommand.ExecuteC(); err != nil {
		if exit, ok := err.(*ExitError); ok {
			os.Exit(exit.Code)
		}
		c.PrintErrln("Error:", err)
		c.Usage()
	}
}
