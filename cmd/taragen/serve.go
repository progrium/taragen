package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/progrium/taragen"
	"tractor.dev/toolkit-go/engine/cli"
)

func serveCmd() *cli.Command {
	cmd := &cli.Command{
		Usage: "serve",
		Short: "serve the site",
		Run: func(ctx *cli.Context, args []string) {
			wd, err := os.Getwd()
			if err != nil {
				fatal(err)
			}

			fmt.Println("serving on http://localhost:8088 ...")

			site := taragen.NewSite(wd)
			site.WatchForReloads()

			if err := site.ParseAll(); err != nil {
				fatal(err)
			}

			if err := http.ListenAndServe(":8088", site); err != nil {
				fatal(err)
			}

		},
	}
	return cmd
}
