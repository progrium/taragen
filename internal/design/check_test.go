package design

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fm builds a proposal file: frontmatter lines, then a body.
func fm(lines, body string) string {
	return "---\n" + lines + "\n---\n\n" + body
}

// prop builds a valid proposal with extra frontmatter lines and a body.
func prop(slug, status, extra, body string) string {
	lines := "slug: " + slug + "\ntitle: T\nstatus: " + status + "\ncreated: 2026-11-01\nauthors: [alice]\ndepends: []"
	if strings.Contains(extra, "depends:") {
		lines = strings.Replace(lines, "\ndepends: []", "", 1)
	}
	if extra != "" {
		lines += "\n" + extra
	}
	return fm(lines, "# Proposal: T\n\n"+body)
}

func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const d = "design/"

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		branches []string // branches that exist
		want     []string
	}{
		{
			name: "valid",
			files: map[string]string{
				d + "base.md": prop("base", "implemented", "branch: [main]",
					"Nov 3, 2026 · Alice Example, h.d.a.\n\nSee [next](next.md).\n\n## Done criteria\n\n- [x] Works.\n  A note.\n- [~] On Plan 9. Waived: no runner.\n"),
				d + "next.md":   prop("next", "draft", "depends: [base]\nbranch: [next]", ""),
				d + "SIGNERS":   "# comment\nalice Alice Example\n",
				d + "SPEC.md":   "# Spec\n\nNov 3, 2026 · Alice Example\n\nSee [base](base.md) and design/next.md.\n",
				"testdata/x.md": "design/ignored.md\n",
			},
			branches: []string{"next"},
		},
		{
			name: "frontmatter",
			files: map[string]string{
				d + "a.md": fm("slug: a\ntitle: yes\nstatus: in-progress\nauthors: alice\ndepends: [a,b]\nmilestone: v1\nno separator", ""),
				d + "b.md": "# no frontmatter\n",
			},
			want: []string{
				`error: design/a.md:1: missing required key "created"`,
				`error: design/a.md:3: title: value "yes" would be read as a YAML boolean or null`,
				`error: design/a.md:4: unknown status "in-progress" (want one of draft, proposed, accepted, implemented, declined, hold, superseded)`,
				`error: design/a.md:5: authors must be a list, like [a, b]`,
				`error: design/a.md:6: depends: invalid list item "a,b" (lists are written [a, b])`,
				`error: design/a.md:7: unknown key "milestone"`,
				`error: design/a.md:8: expected "key: value", got "no separator"`,
				`error: design/b.md:1: missing frontmatter: file must start with a "---" line`,
			},
		},
		{
			name: "naming and references",
			files: map[string]string{
				d + "a.md":      prop("a", "draft", "issue: 3\ndepends: [12, nope]", ""),
				d + "a-copy.md": prop("a", "draft", "issue: 3", ""),
				d + "9lives.md": prop("9lives", "superseded", "superseded-by: gone", ""),
			},
			want: []string{
				`error: design/9lives.md:2: slug "9lives" must start with a letter`,
				`error: design/9lives.md:8: superseded-by refers to unknown proposal "gone"`,
				`error: design/a-copy.md:2: file must be named a.md for slug "a"`,
				`error: design/a-copy.md:2: duplicate slug "a" (also in design/a.md)`,
				`error: design/a-copy.md:8: duplicate issue 3 (also in design/a.md)`,
				`error: design/a.md:8: depends refers to "12" by number; refer to proposals by slug`,
				`error: design/a.md:8: depends refers to unknown proposal "nope"`,
			},
		},
		{
			name: "graph",
			files: map[string]string{
				d + "a.md": prop("a", "draft", "depends: [b]", ""),
				d + "b.md": prop("b", "draft", "depends: [a]", ""),
				d + "c.md": prop("c", "implemented", "depends: [d]\nbranch: [main]", ""),
				d + "d.md": prop("d", "accepted", "depends: [e]", ""),
				d + "e.md": prop("e", "declined", "", "## Declined\n\nNo.\n"),
			},
			want: []string{
				`error: design/a.md:7: dependency cycle: a -> b -> a`,
				`error: design/c.md:7: implemented proposal depends on "d", which is accepted`,
				`error: design/d.md:7: accepted proposal depends on "e", which was declined`,
			},
		},
		{
			name: "status",
			files: map[string]string{
				d + "a.md": prop("a", "proposed", "", ""),
				d + "b.md": prop("b", "declined", "", ""),
				d + "c.md": prop("c", "superseded", "", ""),
				d + "e.md": prop("e", "implemented", "branch: [main, e]", ""),
				d + "f.md": prop("f", "accepted", "branch: [main]", ""),
			},
			branches: []string{"main"},
			want: []string{
				`error: design/a.md:4: proposed proposal has no issue`,
				`error: design/b.md:4: declined proposal has no "## Declined" section`,
				`error: design/c.md:4: superseded proposal has no superseded-by`,
				`error: design/e.md:8: implemented proposal must have branch: [main]`,
				`error: design/f.md:8: branch lists main, but the proposal isn't implemented`,
			},
		},
		{
			name: "done criteria",
			files: map[string]string{
				d + "a.md": prop("a", "implemented", "branch: [main]", "## Done criteria\n\n- [x] Done.\n- [ ] Open.\n- Not a checkbox.\n\n```\n- in code is fine\n```\n"),
				d + "b.md": prop("b", "accepted", "", "## Done criteria\n\n- [x] Done.\n"),
			},
			want: []string{
				`error: design/a.md:16: implemented proposal has an open Done criteria item`,
				`error: design/a.md:17: Done criteria item is not a checkbox (- [ ], - [x], or - [~])`,
				`warning: design/b.md:4: every Done criteria item is checked or waived; is it implemented?`,
			},
		},
		{
			name: "signatures, links, and branches",
			files: map[string]string{
				d + "a.md":    prop("a", "draft", "branch: [a, a-v2]", "November 3, 2026 · Alice Example\n\nSee [b](b.md).\n"),
				d + "SIGNERS": "alice Alice Example\nbroken\n",
				d + "SPEC.md": "# Spec\n\nNov 3, 2026 · Mallory Example\n",
				"README.md":   "See design/missing.md.\n",
				"web/x.md":    "Not ours: web/design/missing.md, docs/design/missing.md.\n",
			},
			branches: []string{"a"},
			want: []string{
				`error: README.md:1: link to design/missing.md, which doesn't exist`,
				`error: design/SIGNERS:2: expected "<login> <Full Name>", got "broken"`,
				`error: design/SPEC.md:3: signed by "Mallory Example", who is not in design/SIGNERS`,
				`warning: design/a.md:8: branch "a-v2" not found (fetch every branch tip to check)`,
				`error: design/a.md:13: signature "November 3, 2026 · Alice Example" is not in the form "Nov 11, 2026 · Full Name, h.d.a."`,
				`error: design/a.md:15: link to design/b.md, which doesn't exist`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Load(writeRepo(t, tt.files))
			if err != nil {
				t.Fatal(err)
			}
			g := StaticGit{}
			for _, b := range tt.branches {
				g[b] = time.Now()
			}
			r.Git = g
			var got []string
			for _, d := range Check(r) {
				got = append(got, d.String())
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(tt.want, "\n  "))
			}
		})
	}
}

func TestState(t *testing.T) {
	r, err := Load(writeRepo(t, map[string]string{
		d + "base.md":    prop("base", "implemented", "branch: [main]", ""),
		d + "ready.md":   prop("ready", "accepted", "depends: [base]", ""),
		d + "blocked.md": prop("blocked", "accepted", "depends: [ready]", ""),
		d + "working.md": prop("working", "draft", "branch: [old, fresh]", ""),
		d + "stale.md":   prop("stale", "accepted", "branch: [old]", ""),
		d + "gone.md":    prop("gone", "hold", "branch: [gone]", ""),
		d + "dropped.md": prop("dropped", "declined", "branch: [old]", "## Declined\n\nNo.\n"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	r.Git = StaticGit{"fresh": now.Add(-2 * 24 * time.Hour), "old": now.Add(-30 * 24 * time.Hour)}
	for slug, want := range map[string]string{
		"base": "", "ready": StateReady, "blocked": "", "working": StateInProgress,
		"stale": StateStalled, "gone": StateStalled, "dropped": "",
	} {
		if got := r.State(r.BySlug[slug], now); got != want {
			t.Errorf("%s: state %q, want %q", slug, got, want)
		}
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
