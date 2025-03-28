package main

import (
	"os"

	"github.com/progrium/taragen"
	"tractor.dev/toolkit-go/engine/cli"
)

func buildCmd() *cli.Command {
	cmd := &cli.Command{
		Usage:   "build",
		Aliases: []string{"gen", "generate"},
		Short:   "build the site",
		Run: func(ctx *cli.Context, args []string) {
			wd, err := os.Getwd()
			if err != nil {
				fatal(err)
			}

			if err := taragen.NewSite(wd, false).GenerateAll("_public", false); err != nil {
				fatal(err)
			}
		},
	}
	return cmd
}
