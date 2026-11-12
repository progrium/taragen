package taragen

import (
	"path"
	"slices"
	"strings"
	"testing"
)

// listPaths renders a page that evaluates the JS expression expr, which
// returns a pages() listing, and returns the listing's paths.
func listPaths(t *testing.T, files map[string]string, expr string, drafts bool) []string {
	t.Helper()
	out, err := renderExpr(t, files, expr, drafts)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func renderExpr(t *testing.T, files map[string]string, expr string, drafts bool) ([]string, error) {
	t.Helper()
	all := map[string]string{"q.jsx": `<p>[{(` + expr + `).map(p => p.path).join(",")}]</p>`}
	for k, v := range files {
		all[k] = v
	}
	out, err := render(t, writeSite(t, all), "q", drafts)
	if err != nil {
		return nil, err
	}
	start, end := strings.Index(out, "<p>["), strings.Index(out, "]</p>")
	if start < 0 || end < 0 {
		t.Fatalf("unexpected output: %s", out)
	}
	list := out[start+4 : end]
	if list == "" {
		return []string{}, nil
	}
	return strings.Split(list, ","), nil
}

func assertPaths(t *testing.T, got []string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if !slices.Equal(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

func assertSamePaths(t *testing.T, got []string, want ...string) {
	t.Helper()
	got, want = slices.Clone(got), slices.Clone(want)
	slices.Sort(got)
	slices.Sort(want)
	assertPaths(t, got, want...)
}

var nestedSite = map[string]string{
	"index.jsx":                  `<p>home</p>`,
	"about.md":                   "---\ntitle: About\n---\n",
	"articles/index.jsx":         `<p>articles</p>`,
	"articles/flat.md":           "---\ntitle: Flat\n---\n",
	"articles/2021/ngrok.md":     "---\ntitle: Ngrok\n---\n",
	"articles/2021/deep/post.md": "---\ntitle: Post\n---\n",
	"docs/api/a.md":              "---\ntitle: A\n---\n",
	"docs/v1/api/b.md":           "---\ntitle: B\n---\n",
	"docs/v1/x/api/c.md":         "---\ntitle: C\n---\n",
	"docs/other/d.md":            "---\ntitle: D\n---\n",
}

func TestPagesDoubleStar(t *testing.T) {
	assertSamePaths(t, listPaths(t, nestedSite, `pages("articles/**")`, false),
		"/articles/flat", "/articles/2021/ngrok", "/articles/2021/deep/post")

	assertSamePaths(t, listPaths(t, nestedSite, `pages("**")`, false),
		"/q", "/about", "/articles", "/articles/flat", "/articles/2021/ngrok",
		"/articles/2021/deep/post", "/docs/api/a", "/docs/v1/api/b",
		"/docs/v1/x/api/c", "/docs/other/d")

	assertSamePaths(t, listPaths(t, nestedSite, `pages("docs/**/api")`, false),
		"/docs/api/a", "/docs/v1/api/b", "/docs/v1/x/api/c")

	files := map[string]string{
		"ab/x.md":  "---\ntitle: X\n---\n",
		"a/y.md":   "---\ntitle: Y\n---\n",
		"b/z.md":   "---\ntitle: Z\n---\n",
		"a/b/w.md": "---\ntitle: W\n---\n",
	}
	single := listPaths(t, files, `pages("a*")`, false)
	assertSamePaths(t, single, "/ab/x", "/a/y", "/a/b")
	assertPaths(t, listPaths(t, files, `pages("a**")`, false), single...)
}

func TestPagesDoubleStarGroupingNodes(t *testing.T) {
	files := map[string]string{
		"logs/index.jsx":     `<p>logs</p>`,
		"logs/2026/01-02.md": "---\ntitle: One\n---\n",
		"logs/2026/01-03.md": "---\ntitle: Two\n---\n",
		"logs/2025/12-31.md": "---\ntitle: Old\n---\n",
		"logs/sub.jsx":       `<p>{pages("logs").map(p => p.path).join(",")}|{pages("logs/2026").length}</p>`,
		"logs/tree.jsx":      `<p>{pages("logs").filter(p => p.isDir).map(y => y.subpages.length).join(",")}</p>`,
	}
	// logs/2025 and logs/2026 are grouping nodes: folders with no index
	assertSamePaths(t, listPaths(t, files, `pages("logs/**")`, false),
		"/logs/2026/01-02", "/logs/2026/01-03", "/logs/2025/12-31", "/logs/sub", "/logs/tree")
	// a plain path and subpages still return them
	assertSamePaths(t, listPaths(t, files, `pages("logs")`, false),
		"/logs/2025", "/logs/2026", "/logs/sub", "/logs/tree")
	out, err := render(t, writeSite(t, files), "logs/tree", false)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, out, "<p>1,2</p>")
	// a pattern without ** keeps returning them
	assertSamePaths(t, listPaths(t, files, `pages("*")`, false),
		"/logs/2025", "/logs/2026", "/logs/sub", "/logs/tree")
}

// legacyPages is pages() as it was before pages-queries, for comparison.
func legacyPages(s *Site, name string) (pages []*Page) {
	if !strings.Contains(name, "*") {
		if s.IsPage(name) {
			return s.Page(name).Subpages()
		}
		return nil
	}
	for pathname, page := range s.pages {
		if m, err := path.Match(name, pathname); err == nil && m {
			pages = append(pages, page.Subpages()...)
		}
	}
	return pages
}

func TestPagesSingleStarUnchanged(t *testing.T) {
	fixture := writeSite(t, nestedSite)
	for _, dir := range []string{"web", fixture} {
		site := NewSite(dir, false)
		if err := site.ParseAll(); err != nil {
			t.Fatal(err)
		}
		queries := []string{"*/*", "*/*/*", "docs/*"}
		for p := range site.pages {
			if p == "." {
				continue
			}
			queries = append(queries, p, "/"+p, p+"/*", p+"/*/*")
		}
		for _, q := range queries {
			got, err := site.Pages(q)
			if err != nil {
				t.Fatal(err)
			}
			pathsOf := func(pages []*Page) (out []string) {
				for _, p := range pages {
					out = append(out, p.path)
				}
				slices.Sort(out)
				return out
			}
			if g, w := pathsOf(got), pathsOf(legacyPages(site, q)); !slices.Equal(g, w) {
				t.Errorf("%s %q: got %v, want %v", dir, q, g, w)
			}
		}
	}
}

func TestPagesRootNotMatched(t *testing.T) {
	// a single * no longer matches the root, whose path is empty for matching
	files := map[string]string{
		"blog/a.md":  "---\ntitle: A\n---\n",
		"notes/b.md": "---\ntitle: B\n---\n",
		"other.md":   "---\ntitle: O\n---\n",
	}
	assertSamePaths(t, listPaths(t, files, `pages("*")`, false), "/blog/a", "/notes/b")
	// and the root is never in its own listing
	assertSamePaths(t, listPaths(t, files, `pages(".")`, false), "/blog", "/notes", "/other", "/q")
}

func TestPagesObjectForm(t *testing.T) {
	for _, q := range []string{"articles", "articles/**", "docs/*/api", "docs/**"} {
		str := listPaths(t, nestedSite, `pages("`+q+`")`, false)
		obj := listPaths(t, nestedSite, `pages({ path: "`+q+`" })`, false)
		assertPaths(t, obj, str...)
	}
	assertPaths(t, listPaths(t, nestedSite, `pages({})`, false),
		listPaths(t, nestedSite, `pages("**")`, false)...)

	files := map[string]string{
		"logs/2026/a.md": "---\ntitle: A\ndate: 2026-01-03\n---\n",
		"logs/2026/b.md": "---\ntitle: B\ndate: 2026-01-01\n---\n",
		"articles/c.md":  "---\ntitle: C\ndate: 2026-01-02\n---\n",
		"articles/d.md":  "---\ntitle: D\ndate: 2025-06-01\n---\n",
		"docs/e.md":      "---\ntitle: E\n---\n",
		"docs/index.jsx": `<p>docs</p>`,
		"other/x.md":     "---\ntitle: X\ndate: 2026-01-04\n---\n",
		"logs/index.jsx": `<p>logs</p>`,
	}
	// a list of paths is one union, once each, sorted as one listing
	got := listPaths(t, files, `pages({ path: ["logs/**", "articles/*.md", "articles/**", "docs", "logs/**"] }).filter(p => p.date)`, false)
	assertPaths(t, got, "/articles/d", "/logs/2026/b", "/articles/c", "/logs/2026/a")
	all := listPaths(t, files, `pages({ path: ["logs/**", "articles/**", "docs"] })`, false)
	if len(all) != len(slices.Compact(slices.Sorted(slices.Values(all)))) {
		t.Errorf("duplicates in %v", all)
	}
	assertPaths(t, listPaths(t, files, `pages({ path: [] })`, false))
}

func TestPagesWhere(t *testing.T) {
	files := map[string]string{
		"a.md":        "---\ntitle: A\ntags: [plan9, go]\nkind: log\nyear: 2026\n---\n",
		"b.md":        "---\ntitle: B\ntags: [go]\nkind: article\nyear: \"2026\"\n---\n",
		"c.md":        "---\ntitle: C\ntags: plan9\nkind: log\npinned: true\n---\n",
		"d.md":        "---\ntitle: D\nkind: Log\n---\n",
		"sec/e.md":    "---\ntitle: E\ntags: [plan9]\n---\n",
		"sec/f/g.md":  "---\ntitle: G\n---\n",
		"sec/f/h.md":  "---\ntitle: H\ntags: [plan9]\n---\n",
		"sec/f.md":    "---\ntitle: F\ntags: [plan9]\n---\n",
		"sec/sub.jsx": `<p>{pages({ path: "sec", where: { tags: "plan9" } }).map(p => p.slug + ":" + p.subpages.map(s => s.slug).join("+")).join(",")}</p>`,
	}
	cases := map[string][]string{
		`{ kind: "log" }`:                         {"/a", "/c"},
		`{ tags: "plan9" }`:                       {"/a", "/c", "/sec/e", "/sec/f", "/sec/f/h"},
		`{ tags: ["plan9", "go"] }`:               {"/a", "/b", "/c", "/sec/e", "/sec/f", "/sec/f/h"},
		`{ tags: "go", kind: "log" }`:             {"/a"},
		`{ tags: "go", kind: "nope" }`:            {},
		`{ missing: "x" }`:                        {},
		`{ year: 2026 }`:                          {"/a"},
		`{ year: "2026" }`:                        {"/b"},
		`{ year: [2026, "2026"] }`:                {"/a", "/b"},
		`{ pinned: true }`:                        {"/c"},
		`{ pinned: false }`:                       {},
		`{ kind: "LOG" }`:                         {},
		`{ kind: ["Log", "article"] }`:            {"/b", "/d"},
		`{ tags: [] }`:                            {},
		`{ title: "A", kind: "log", year: 2026 }`: {"/a"},
	}
	for where, want := range cases {
		t.Run(where, func(t *testing.T) {
			assertSamePaths(t, listPaths(t, files, `pages({ where: `+where+` })`, false), want...)
		})
	}
	// where filters the result, never its subpages
	out, err := render(t, writeSite(t, files), "sec/sub", false)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, out, "<p>e:,f:g+h</p>")
}

func TestPagesQueryErrors(t *testing.T) {
	files := map[string]string{"a/b.md": "---\ntitle: B\n---\n"}
	for expr, msg := range map[string]string{
		`pages("lo[gs/*")`:                     `pages("lo[gs/*")`,
		`pages("a/**/[")`:                      `pages("a/**/[")`,
		`pages({ path: "lo[gs/*" })`:           `pages({"path":"lo[gs/*"})`,
		`pages({ path: ["a", "[*"] })`:         `pages({"path":["a","[*"]})`,
		`pages({ paths: "a" })`:                `unknown key "paths"`,
		`pages({ path: 1 })`:                   `path must be a string`,
		`pages({ path: ["a", 1] })`:            `path must be a string`,
		`pages({ path: null })`:                `path must be a string`,
		`pages({ where: { tags: { a: 1 } } })`: `where.tags must be`,
		`pages({ where: { tags: null } })`:     `where.tags must be`,
		`pages({ where: { tags: [[1]] } })`:    `where.tags must be`,
		`pages({ where: "x" })`:                `where must be an object`,
		`pages(["a"])`:                         `query must be a string or an object`,
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := renderExpr(t, files, expr, false)
			if err == nil || !strings.Contains(err.Error(), msg) {
				t.Errorf("got error %v, want one containing %q", err, msg)
			}
		})
	}
}

func TestPagesOrder(t *testing.T) {
	files := map[string]string{
		"logs/2026/a.md": "---\ntitle: LA\ndate: 2026-01-02\n---\n",
		"x/a.md":         "---\ntitle: XA\ndate: 2026-01-02\n---\n",
		"logs/2026/b.md": "---\ntitle: LB\ndate: 2026-01-02\n---\n",
		"logs/2026/c.md": "---\ntitle: LC\ndate: 2026-01-02\n---\n",
		"logs/old.md":    "---\ntitle: Old\ndate: 2020-05-05\n---\n",
		"logs/z.md":      "---\ntitle: Z\n---\n",
		"logs/y.md":      "---\ntitle: Y\ndate: not-a-date\n---\n",
		"x/y.md":         "---\ntitle: XY\n---\n",
	}
	want := []string{"/logs/y", "/x/y", "/logs/z", "/logs/old", "/logs/2026/a", "/x/a", "/logs/2026/b", "/logs/2026/c"}
	for i := 0; i < 20; i++ {
		assertPaths(t, listPaths(t, files, `pages({ path: ["logs/**", "x"] })`, false), want...)
	}
	// subpages use the same order
	site := NewSite(writeSite(t, files), false)
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range site.Page("logs/2026").Subpages() {
		got = append(got, p.path)
	}
	assertPaths(t, got, "logs/2026/a", "logs/2026/b", "logs/2026/c")
}

func TestPagesDrafts(t *testing.T) {
	files := map[string]string{
		"blog/a.md":         "---\ntitle: A\ntags: [x]\n---\n",
		"blog/2026/b.md":    "---\ntitle: B\ntags: [x]\ndraft: true\n---\n",
		"blog/2026/c.md":    "---\ntitle: C\ntags: [x]\ndate: 2999-01-01\n---\n",
		"blog/2026/d.md":    "---\ntitle: D\ntags: [x]\n---\n",
		"blog/index.jsx":    `<p>blog</p>`,
		"blog/2026/e.md":    "---\ntitle: E\ndraft: true\n---\n",
		"blog/2026/x/y.md":  "---\ntitle: Y\ntags: [x]\n---\n",
		"blog/2026/x/z.jsx": `<p>z</p>`,
	}
	for _, expr := range []string{
		`pages("blog/2026")`,
		`pages("blog/*")`,
		`pages("blog/**")`,
		`pages({ path: "blog/**" })`,
		`pages({ path: ["blog/2026", "blog/**"] })`,
		`pages({ where: { tags: "x" } })`,
	} {
		t.Run(expr, func(t *testing.T) {
			built := listPaths(t, files, expr, false)
			served := listPaths(t, files, expr, true)
			for _, d := range []string{"/blog/2026/b", "/blog/2026/c", "/blog/2026/e"} {
				if slices.Contains(built, d) {
					t.Errorf("build: draft %s in %v", d, built)
				}
			}
			if !slices.Contains(served, "/blog/2026/b") || !slices.Contains(served, "/blog/2026/c") {
				t.Errorf("serve: drafts missing from %v", served)
			}
			if !slices.Contains(built, "/blog/2026/d") {
				t.Errorf("build: /blog/2026/d missing from %v", built)
			}
		})
	}
}
