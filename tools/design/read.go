package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/progrium/taragen/internal/design"
)

func cmdList(root string, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	status := fs.String("status", "", "only proposals with this status")
	area := fs.String("area", "", "only proposals tagged with this area")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	var ps []*design.Proposal
	for _, p := range r.Proposals {
		if *status != "" && p.Status != *status {
			continue
		}
		if *area != "" && !contains(p.Areas, *area) {
			continue
		}
		ps = append(ps, p)
	}
	printTable(r, ps)
	return nil
}

func cmdReady(root string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("ready takes no arguments")
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	printTable(r, r.Ready())
	return nil
}

func printTable(r *design.Repo, ps []*design.Proposal) {
	now := time.Now()
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tISSUE\tSTATUS\tSTATE\tTITLE")
	for _, p := range ps {
		issue := ""
		if p.Issue != 0 {
			issue = fmt.Sprintf("#%d", p.Issue)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", p.Slug, issue, p.Status, r.State(p, now), p.Title)
	}
	w.Flush()
}

func cmdGraph(root string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("graph takes no arguments")
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	fmt.Print(r.Graph(time.Now()))
	return nil
}

func cmdStatus(root string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("status takes no arguments")
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	out := filepath.Join(root, design.IndexPath)
	if err := os.WriteFile(out, []byte(r.Index(time.Now())), 0644); err != nil {
		return err
	}
	fmt.Println("wrote", design.IndexPath)
	return nil
}

func cmdCheck(root string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("check takes no arguments")
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	diags := design.Check(r)
	errs, warns := 0, 0
	for _, d := range diags {
		fmt.Println(d)
		if d.Warn {
			warns++
		} else {
			errs++
		}
	}
	fmt.Printf("checked %d proposals: %d errors, %d warnings\n", len(r.Proposals), errs, warns)
	if errs > 0 {
		return fmt.Errorf("check failed")
	}
	return nil
}

// cmdLog walks the history of a proposal's file and prints each commit that
// changed its status or branch.
func cmdLog(root string, args []string) error {
	slug, err := oneSlug("log", args)
	if err != nil {
		return err
	}
	rel := design.DesignDir + "/" + slug + ".md"
	out, err := exec.Command("git", "-C", root, "log", "--reverse", "--format=%H %h %ad", "--date=short", "--", rel).Output()
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	prevStatus, prevBranch := "", ""
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		src, err := exec.Command("git", "-C", root, "show", parts[0]+":"+rel).Output()
		if err != nil {
			continue // deleted in this commit
		}
		fm, _, _, _ := design.ParseFrontmatter(string(src))
		status, branch := "", "[]"
		if f := fm.Get("status"); f != nil {
			status = f.Scalar
		}
		if f := fm.Get("branch"); f != nil && f.IsList {
			branch = design.FormatList(f.List)
		}
		var changes []string
		if status != prevStatus {
			changes = append(changes, "status: "+status)
		}
		if branch != prevBranch {
			changes = append(changes, "branch: "+branch)
		}
		if len(changes) > 0 {
			fmt.Printf("%s %s  %s\n", parts[2], parts[1], strings.Join(changes, "  "))
		}
		prevStatus, prevBranch = status, branch
	}
	return nil
}

// cmdTest runs the test cases tagged with one proposal, whatever its status.
func cmdTest(root string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("test takes a slug")
	}
	slug, rest := args[0], args[1:]
	r, err := load(root)
	if err != nil {
		return err
	}
	if r.BySlug[slug] == nil {
		return fmt.Errorf("no proposal %q", slug)
	}
	// match subtests whose name ends in a tag list that includes slug
	run := fmt.Sprintf(`/\[(.*, )?%s(, .*)?\]$`, slug)
	cmdArgs := append([]string{"test", "-run", run}, rest...)
	cmdArgs = append(cmdArgs, "./...")
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "DESIGN_PENDING=1")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	fmt.Println("go", strings.Join(cmdArgs, " "))
	return cmd.Run()
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
