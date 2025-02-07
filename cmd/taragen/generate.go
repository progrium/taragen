package main

import (
	"os"

	"github.com/progrium/taragen"
	"tractor.dev/toolkit-go/engine/cli"
)

func generateCmd() *cli.Command {
	cmd := &cli.Command{
		Usage:   "generate",
		Aliases: []string{"gen"},
		Short:   "generate the site",
		Run: func(ctx *cli.Context, args []string) {
			wd, err := os.Getwd()
			if err != nil {
				fatal(err)
			}

			if err := taragen.NewSite(wd).GenerateAll("public", false); err != nil {
				fatal(err)
			}
		},
	}
	return cmd
}
