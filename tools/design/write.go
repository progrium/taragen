package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/progrium/taragen/internal/design"
)

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// cmdNew copies the template into a new draft with its frontmatter filled in.
func cmdNew(root string, args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	title := fs.String("title", "", "the proposal's title (default: from the slug)")
	author := fs.String("author", "", "your account login (default: from gh, then git config github.user)")
	pos, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	slug, err := oneSlug("new", pos)
	if err != nil {
		return err
	}
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("slug %q must be lowercase letters and digits joined by dashes, starting with a letter", slug)
	}
	dest := filepath.Join(root, design.DesignDir, slug+".md")
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("%s already exists", dest)
	}
	if *title == "" {
		words := strings.Split(slug, "-")
		words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
		*title = strings.Join(words, " ")
	}
	if *author == "" {
		*author = currentLogin()
	}
	if *author == "" {
		return fmt.Errorf("can't tell your login; pass --author")
	}
	tmpl, err := os.ReadFile(filepath.Join(root, design.DesignDir, design.TemplateName))
	if err != nil {
		return err
	}
	_, body, _, errs := design.ParseFrontmatter(string(tmpl))
	if len(errs) > 0 {
		return fmt.Errorf("template frontmatter: %v", errs[0])
	}
	body = strings.ReplaceAll(body, "<title>", *title)
	fm := strings.Join([]string{
		"---",
		"slug: " + slug,
		"title: " + *title,
		"status: draft",
		"created: " + time.Now().Format("2006-01-02"),
		"authors: " + design.FormatList([]string{*author}),
		"depends: []",
		"branch: []",
		"---",
	}, "\n")
	src := fm + "\n" + body
	// make sure what we wrote passes our own parser
	if _, _, _, errs := design.ParseFrontmatter(src); len(errs) > 0 {
		return fmt.Errorf("title %q can't be written as frontmatter: %s", *title, errs[0].Msg)
	}
	if err := os.WriteFile(dest, []byte(src), 0644); err != nil {
		return err
	}
	fmt.Println("created", filepath.Join(design.DesignDir, slug+".md"))
	return nil
}

// cmdPropose files the proposal's issue and marks it proposed, in one commit.
func cmdPropose(root string, args []string) error {
	fs := flag.NewFlagSet("propose", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "don't ask before filing")
	dryRun := fs.Bool("dry-run", false, "show what would be filed, and change nothing")
	adopt := fs.Int("issue", 0, "adopt this existing issue instead of filing one")
	pos, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	slug, err := oneSlug("propose", pos)
	if err != nil {
		return err
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	p := r.BySlug[slug]
	if p == nil {
		return fmt.Errorf("no proposal %q", slug)
	}
	if p.Status != "draft" {
		return fmt.Errorf("%s is %s; only a draft can be proposed", slug, p.Status)
	}
	if p.Issue != 0 {
		return fmt.Errorf("%s already has issue #%d", slug, p.Issue)
	}

	issueTitle := "proposal: " + p.Title
	number := *adopt
	if number == 0 {
		repo, err := repoName(root)
		if err != nil {
			return err
		}
		body := sectionText(p.Doc.Section("Abstract")) + "\n\n" +
			fmt.Sprintf("Proposal: https://github.com/%s/blob/main/%s", repo, p.Path)
		if *dryRun {
			fmt.Printf("would file an issue in %s with label Proposal:\n\ntitle: %s\n\n%s\n", repo, issueTitle, body)
			return nil
		}
		dupes, err := ghOut("issue", "list", "--state", "open", "--search", issueTitle+" in:title", "--json", "number,title", "--jq",
			fmt.Sprintf(`.[] | select(.title == %s) | .number`, strconv.Quote(issueTitle)))
		if err != nil {
			return err
		}
		if dupes != "" {
			return fmt.Errorf("an open issue already has the title %q (#%s); adopt it with --issue", issueTitle, strings.Fields(dupes)[0])
		}
		if !*yes && !confirm(fmt.Sprintf("File %q in %s as the logged-in gh account?", issueTitle, repo)) {
			return fmt.Errorf("not filed")
		}
		url, err := ghOut("issue", "create", "--title", issueTitle, "--label", "Proposal", "--body", body)
		if err != nil {
			return err
		}
		n, err := strconv.Atoi(url[strings.LastIndex(url, "/")+1:])
		if err != nil {
			return fmt.Errorf("can't read the issue number from %q", url)
		}
		number = n
		fmt.Println("filed", url)
	} else if *dryRun {
		fmt.Printf("would adopt issue #%d for %s\n", number, slug)
		return nil
	}

	src, err := design.SetField(p.Src, "issue", strconv.Itoa(number))
	if err == nil {
		src, err = design.SetField(src, "status", "proposed")
	}
	if err != nil {
		return err
	}
	// keep issue first, as new writes it
	src = moveIssueFirst(src)
	if err := os.WriteFile(filepath.Join(root, p.Path), []byte(src), 0644); err != nil {
		return err
	}
	return git(root, "commit", "-m", fmt.Sprintf("design: propose %s (#%d)", slug, number), "--", p.Path)
}

func moveIssueFirst(src string) string {
	lines := strings.Split(src, "\n")
	for i := 1; i < len(lines) && lines[i] != "---"; i++ {
		if strings.HasPrefix(lines[i], "issue:") && i != 1 {
			l := lines[i]
			lines = append(lines[:i], lines[i+1:]...)
			lines = append(lines[:1], append([]string{l}, lines[1:]...)...)
			break
		}
	}
	return strings.Join(lines, "\n")
}

func sectionText(s *design.Section) string {
	if s == nil {
		return ""
	}
	var lines []string
	for _, l := range s.Lines {
		lines = append(lines, l.Text)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// cmdSign adds the signer's signature under a doc's title. It is interactive
// on purpose: a person runs it at a terminal after reviewing the doc. CI
// runs it with explicit --login and --date after a /sign comment.
func cmdSign(root string, args []string) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	login := fs.String("login", "", "signer's account login (required without a terminal)")
	date := fs.String("date", "", "signature date, YYYY-MM-DD (required without a terminal)")
	pos, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	target, err := oneSlug("sign", pos)
	if err != nil {
		return err
	}
	r, err := load(root)
	if err != nil {
		return err
	}

	var rel string
	var doc *design.Doc
	var src string
	if target == "spec" {
		rel = design.SpecPath
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return err
		}
		src = string(b)
		doc = design.ParseDoc(src, 1)
	} else {
		p := r.BySlug[target]
		if p == nil {
			return fmt.Errorf("no proposal %q", target)
		}
		rel, src, doc = p.Path, p.Src, p.Doc
	}

	interactive := isTerminal(os.Stdin)
	if !interactive && (*login == "" || *date == "") {
		return fmt.Errorf("sign is interactive; without a terminal, pass both --login and --date")
	}
	when := time.Now()
	if *date != "" {
		if when, err = time.Parse("2006-01-02", *date); err != nil {
			return fmt.Errorf("--date: %w", err)
		}
	}

	var signer design.Signer
	var ok bool
	if *login != "" {
		signer, ok = r.SignerByLogin(*login)
	} else if l := currentLogin(); l != "" {
		signer, ok = r.SignerByLogin(l)
	}
	if !ok {
		if name, err := gitOut(root, "config", "user.name"); err == nil && *login == "" {
			signer, ok = r.SignerByName(name)
		}
	}
	if !ok {
		return fmt.Errorf("you're not in %s; only the person a line names may add it", design.SignersPath)
	}

	if doc.Signature != "" {
		if old, err := design.ParseSignature(doc.Signature); err == nil && old.Name == signer.Name &&
			old.Date.Format("2006-01-02") == when.Format("2006-01-02") {
			return fmt.Errorf("%s already signed %s on that date", signer.Name, rel)
		}
	}
	sig := design.FormatSignature(when, signer.Name, true)

	if interactive {
		fmt.Printf("Signing %s (%s)\n", rel, doc.Title)
		if doc.Signature != "" {
			fmt.Printf("  replacing: %s\n", doc.Signature)
		}
		fmt.Printf("  with:      %s\n\n", sig)
		fmt.Println("Your signature says you directed this work, reviewed it closely enough to defend it, and own it.")
		fmt.Printf("Type %q to sign: ", target)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(line) != target {
			return fmt.Errorf("not signed")
		}
	}

	out, err := design.ApplySignature(src, doc, sig)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte(out), 0644); err != nil {
		return err
	}
	fmt.Printf("signed %s: %s\n", rel, sig)
	return nil
}

// cmdCloseIssues closes the open issues of decided proposals.
func cmdCloseIssues(root string, args []string) error {
	fs := flag.NewFlagSet("close-issues", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "show what would be closed")
	if _, err := parseFlags(fs, args); err != nil {
		return err
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	repo := ""
	for _, p := range r.Proposals {
		reason := map[string]string{"implemented": "completed", "declined": "not planned", "superseded": "not planned"}[p.Status]
		if p.Issue == 0 || reason == "" {
			continue
		}
		state, err := ghOut("issue", "view", strconv.Itoa(p.Issue), "--json", "state", "--jq", ".state")
		if err != nil {
			return err
		}
		if state != "OPEN" {
			continue
		}
		if repo == "" {
			if repo, err = repoName(root); err != nil {
				return err
			}
		}
		comment := fmt.Sprintf("Closing: the proposal is %s. See https://github.com/%s/blob/main/%s", p.Status, repo, p.Path)
		if *dryRun {
			fmt.Printf("would close #%d (%s) as %s\n", p.Issue, p.Slug, reason)
			continue
		}
		if _, err := ghOut("issue", "close", strconv.Itoa(p.Issue), "--reason", reason, "--comment", comment); err != nil {
			return err
		}
		fmt.Printf("closed #%d (%s) as %s\n", p.Issue, p.Slug, reason)
	}
	return nil
}

var remoteRe = regexp.MustCompile(`github\.com[:/]([^/]+/[^/]+?)(\.git)?/?$`)

// repoName returns the GitHub owner/name of the origin remote, asking gh
// if the remote isn't a GitHub URL.
func repoName(root string) (string, error) {
	if url, err := gitOut(root, "remote", "get-url", "origin"); err == nil {
		if m := remoteRe.FindStringSubmatch(url); m != nil {
			return m[1], nil
		}
	}
	return ghOut("repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
}

// currentLogin asks gh for the logged-in account, then falls back to git
// config github.user.
func currentLogin() string {
	if l, err := ghOut("api", "user", "--jq", ".login"); err == nil && l != "" {
		return l
	}
	if l, err := gitOut(".", "config", "github.user"); err == nil {
		return l
	}
	return ""
}

func ghOut(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitOut(root string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

func git(root string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func confirm(q string) bool {
	if !isTerminal(os.Stdin) {
		return false
	}
	fmt.Printf("%s [y/N] ", q)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(line), "y")
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
