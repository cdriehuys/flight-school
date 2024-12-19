package main

import (
	"fmt"
	"os"

	"github.com/cdriehuys/flight-school/acs"
	"github.com/cdriehuys/flight-school/internal/cli"
	"github.com/cdriehuys/flight-school/migrations"
)

func main() {
	cmd := cli.NewRootCmd(os.Stderr, acs.Files, migrations.Files)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
