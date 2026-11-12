package design

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Each directory under testdata/check is one fixture: repo/ is a tiny
// repository root, and want.txt lists the exact diagnostics check reports
// for it, one per line. An optional branches.txt lists the branches that
// exist, as "<name> <days since last commit>".
func TestCheckFixtures(t *testing.T) {
	dirs, err := os.ReadDir("testdata/check")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		dir := filepath.Join("testdata/check", d.Name())
		t.Run(d.Name(), func(t *testing.T) {
			r, err := Load(filepath.Join(dir, "repo"))
			if err != nil {
				t.Fatal(err)
			}
			if g := readBranches(t, filepath.Join(dir, "branches.txt")); g != nil {
				r.Git = g
			}
			var got []string
			for _, d := range Check(r) {
				got = append(got, d.String())
			}
			want := readLines(t, filepath.Join(dir, "want.txt"))
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("check output differs\n got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
			}
		})
	}
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(string(b), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func readBranches(t *testing.T, path string) Git {
	t.Helper()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g := StaticGit{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name, days, ok := strings.Cut(sc.Text(), " ")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(days)
		if err != nil {
			t.Fatal(err)
		}
		g[name] = time.Now().Add(-time.Duration(n) * 24 * time.Hour)
	}
	return g
}

func TestState(t *testing.T) {
	r, err := Load("testdata/state/repo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	r.Git = StaticGit{"fresh": now.Add(-2 * 24 * time.Hour), "old": now.Add(-30 * 24 * time.Hour)}
	want := map[string]string{
		"base":        "",
		"ready-one":   StateReady,
		"blocked":     "",
		"working":     StateInProgress,
		"old-work":    StateStalled,
		"gone-branch": StateStalled,
		"dropped":     "",
	}
	for slug, w := range want {
		p := r.BySlug[slug]
		if p == nil {
			t.Fatalf("no fixture proposal %q", slug)
		}
		if got := r.State(p, now); got != w {
			t.Errorf("%s: state %q, want %q", slug, got, w)
		}
	}
	var ready []string
	for _, p := range r.Ready() {
		ready = append(ready, p.Slug)
	}
	if strings.Join(ready, ",") != "ready-one" {
		t.Errorf("Ready() = %v, want [ready-one]", ready)
	}
}

func TestFrontmatter(t *testing.T) {
	ok := []string{
		"slug: a-b",
		"title: Some title, with commas",
		"depends: []",
		"depends: [a, b-c]",
		"superseded-by:",
		"branch: [feature/x, v2.0]",
	}
	for _, line := range ok {
		if _, _, _, errs := ParseFrontmatter("---\n" + line + "\n---\nbody"); len(errs) != 0 {
			t.Errorf("%q: unexpected error %v", line, errs)
		}
	}
	bad := []string{
		"title: yes",
		"title: Null",
		"title: 'quoted'",
		"title: \"quoted\"",
		"title: - item",
		"title: a: b",
		"title: a #comment",
		"title:  two spaces",
		"depends: [a,b]",
		"depends: [a, 'b']",
		"depends: [a",
		"  nested: x",
		"no separator",
		"Key: x",
		"",
	}
	for _, line := range bad {
		if _, _, _, errs := ParseFrontmatter("---\n" + line + "\n---\nbody"); len(errs) == 0 {
			t.Errorf("%q: expected an error", line)
		}
	}
	if _, _, _, errs := ParseFrontmatter("---\nslug: a\nslug: b\n---\n"); len(errs) != 1 {
		t.Errorf("duplicate key: got %v", errs)
	}
	if _, _, _, errs := ParseFrontmatter("no frontmatter"); len(errs) != 1 {
		t.Errorf("missing frontmatter: got %v", errs)
	}
}

func TestSetField(t *testing.T) {
	src := "---\nslug: a\nstatus: draft\n---\n# body\n"
	out, err := SetField(src, "status", "proposed")
	if err != nil || out != "---\nslug: a\nstatus: proposed\n---\n# body\n" {
		t.Errorf("replace: %q, %v", out, err)
	}
	out, err = SetField(src, "issue", "7")
	if err != nil || out != "---\nslug: a\nstatus: draft\nissue: 7\n---\n# body\n" {
		t.Errorf("insert: %q, %v", out, err)
	}
}

func TestSignature(t *testing.T) {
	for _, line := range []string{
		"Nov 11, 2026 · Jane Q. Public, h.d.a.",
		"Nov 1, 2026 · Jane Q. Public",
	} {
		if _, err := ParseSignature(line); err != nil {
			t.Errorf("%q: %v", line, err)
		}
	}
	for _, line := range []string{
		"November 11, 2026 · Jane",
		"Nov 31, 2026 · Jane",
		"Nov 11 2026 · Jane",
		"Nov 11, 2026 - Jane",
		"Nov 11, 2026 · , h.d.a.",
		"Nov 01, 2026 · Jane",
	} {
		if _, err := ParseSignature(line); err == nil {
			t.Errorf("%q: expected an error", line)
		}
	}

	date := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	sig := FormatSignature(date, "Jane Q. Public", true)
	if sig != "Nov 12, 2026 · Jane Q. Public, h.d.a." {
		t.Fatalf("FormatSignature = %q", sig)
	}

	unsigned := "# Title\n\n## Abstract\n"
	out, err := ApplySignature(unsigned, ParseDoc(unsigned, 1), sig)
	if want := "# Title\n\n" + sig + "\n\n## Abstract\n"; err != nil || out != want {
		t.Errorf("add signature:\n got %q\nwant %q", out, want)
	}
	signed := "# Title\n\nOct 1, 2026 · Someone Else\n\n## Abstract\n"
	out, err = ApplySignature(signed, ParseDoc(signed, 1), sig)
	if want := "# Title\n\n" + sig + "\n\n## Abstract\n"; err != nil || out != want {
		t.Errorf("replace signature:\n got %q\nwant %q", out, want)
	}
	// line numbers in a proposal body are absolute, so the frontmatter stays put
	prop := "---\nslug: a\n---\n\n# Proposal: A\n\n## Abstract\n"
	_, body, bodyLine, _ := ParseFrontmatter(prop)
	out, err = ApplySignature(prop, ParseDoc(body, bodyLine), sig)
	if want := "---\nslug: a\n---\n\n# Proposal: A\n\n" + sig + "\n\n## Abstract\n"; err != nil || out != want {
		t.Errorf("sign proposal:\n got %q\nwant %q", out, want)
	}
}
