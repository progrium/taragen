package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/progrium/taragen/internal/design"
)

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// cmdNew copies the template into a new draft with its frontmatter filled in.
func cmdNew(root string, args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	title := fs.String("title", "", "the proposal's title (default: from the slug)")
	author := fs.String("author", "", "your GitHub login (default: git config github.user)")
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
		*author, _ = gitOut(root, "config", "github.user")
	}
	if *author == "" {
		return fmt.Errorf("can't tell your login; pass --author or set git config github.user")
	}
	tmpl, err := os.ReadFile(filepath.Join(root, design.DesignDir, design.TemplateName))
	if err != nil {
		return err
	}
	_, body, _, errs := design.ParseFrontmatter(string(tmpl))
	if len(errs) > 0 {
		return fmt.Errorf("template frontmatter: %v", errs[0])
	}
	src := strings.Join([]string{
		"---",
		"slug: " + slug,
		"title: " + *title,
		"status: draft",
		"created: " + time.Now().Format("2006-01-02"),
		"authors: " + design.FormatList([]string{*author}),
		"depends: []",
		"branch: []",
		"---",
	}, "\n") + "\n" + strings.ReplaceAll(body, "<title>", *title)
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

// cmdSign adds your signature under a doc's title. It is interactive on
// purpose, so only a person at a terminal can sign: it finds you in SIGNERS
// by git config github.user or user.name, shows what you're signing, and
// asks you to type the slug.
func cmdSign(root string, args []string) error {
	target, err := oneSlug("sign", args)
	if err != nil {
		return err
	}
	if !isTerminal(os.Stdin) {
		return fmt.Errorf("sign is interactive; run it yourself at a terminal")
	}
	r, err := load(root)
	if err != nil {
		return err
	}

	var rel, src string
	var doc *design.Doc
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

	var signer design.Signer
	ok := false
	if login, err := gitOut(root, "config", "github.user"); err == nil && login != "" {
		signer, ok = r.SignerByLogin(login)
	}
	if !ok {
		if name, err := gitOut(root, "config", "user.name"); err == nil {
			signer, ok = r.SignerByName(name)
		}
	}
	if !ok {
		return fmt.Errorf("you're not in %s (matched by git config github.user or user.name)", design.SignersPath)
	}

	today := time.Now()
	if doc.Signature != "" {
		if old, err := design.ParseSignature(doc.Signature); err == nil && old.Name == signer.Name &&
			old.Date.Format("2006-01-02") == today.Format("2006-01-02") {
			return fmt.Errorf("you already signed %s today", rel)
		}
	}
	sig := design.FormatSignature(today, signer.Name, true)

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

func gitOut(root string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
