// Command intentions is the reference implementation of the Intentions
// Format: a CLI for workspaces of intentions, availability and commitments.
package main

import (
	"os"
	_ "time/tzdata" // the static binary carries the IANA database

	"golang.org/x/term"

	"github.com/nodelogicau/intentions-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, func() bool {
		return term.IsTerminal(int(os.Stdin.Fd()))
	}))
}
