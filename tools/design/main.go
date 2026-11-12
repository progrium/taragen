// Command design runs taragen's proposal-driven design workflow. See
// docs/design/README.md.
//
//	go run ./tools/design <command> [flags]
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/progrium/taragen/internal/design"
)

type command struct {
	name, args, short string
	run               func(root string, args []string) error
}

var commands = []command{
	{"list", "[--status S] [--area A]", "table of slug, issue, status, computed state, and title", cmdList},
	{"ready", "", "accepted proposals with no branch and every dependency implemented", cmdReady},
	{"graph", "", "the dependency graph, as Mermaid", cmdGraph},
	{"new", "<slug> [--title T] [--author LOGIN]", "start a draft from the template", cmdNew},
	{"log", "<slug>", "its status and branch changes, with their commits", cmdLog},
	{"sign", "<slug|spec>", "sign a doc you've reviewed (people only, never agents)", cmdSign},
	{"check", "", "validate proposals, signatures, and links", cmdCheck},
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go run ./tools/design [-C dir] <command> [flags]")
	fmt.Fprintln(os.Stderr)
	for _, c := range commands {
		fmt.Fprintf(os.Stderr, "  %-14s %s\n  %-14s   %s\n", c.name, c.args, "", c.short)
	}
}

func main() {
	args := os.Args[1:]
	root := ""
	if len(args) >= 2 && args[0] == "-C" {
		root, args = args[1], args[2:]
	}
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		usage()
		os.Exit(2)
	}
	if root == "" {
		root = findRoot()
	}
	for _, c := range commands {
		if c.name == args[0] {
			if err := c.run(root, args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, "design:", err)
				os.Exit(1)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "design: unknown command %q\n\n", args[0])
	usage()
	os.Exit(2)
}

// findRoot returns the top of the git checkout, or the current directory.
func findRoot() string {
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	wd, _ := os.Getwd()
	return wd
}

func load(root string) (*design.Repo, error) {
	r, err := design.Load(root)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", filepath.Join(root, design.DesignDir), err)
	}
	return r, nil
}

// parseFlags parses flags that may come before or after positional args.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

func oneSlug(name string, pos []string) (string, error) {
	if len(pos) != 1 {
		return "", fmt.Errorf("%s takes exactly one slug", name)
	}
	return pos[0], nil
}
