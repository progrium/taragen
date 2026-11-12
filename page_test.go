package taragen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// render renders the page at pagepath after parsing the whole site.
func render(t *testing.T, dir, pagepath string, drafts bool) (string, error) {
	t.Helper()
	site := NewSite(dir, drafts)
	if err := site.ParseAll(); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err := site.Page(pagepath).Render(&buf)
	return buf.String(), err
}

func mustRender(t *testing.T, dir, pagepath string) string {
	t.Helper()
	out, err := render(t, dir, pagepath, false)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func assertContains(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestGlobalsAreLoadedFromParentDirs(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_globals.jsx":         "var site = \"Root\";\nvar color = \"red\";\nfunction shout(s) { return s.toUpperCase(); }",
		"blog/_globals.jsx":    `var color = "blue";`,
		"blog/2024/post.jsx":   `<p>{site}|{color}|{shout("hi")}</p>`,
		"about.jsx":            `<p>{site}|{color}</p>`,
		"_partials/footer.jsx": `() => <footer>{site}</footer>`,
		"blog/2024/footer.jsx": `<div>{partial("_partials/footer")}</div>`,
	})
	assertContains(t, mustRender(t, dir, "blog/2024/post"), "<p>Root|blue|HI</p>")
	assertContains(t, mustRender(t, dir, "about"), "<p>Root|red</p>")
	assertContains(t, mustRender(t, dir, "blog/2024/footer"), "<footer>Root</footer>")
}

func TestDefaultsSetPageData(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_globals.jsx":       `var author = "Jeff";`,
		"_defaults.jsx":      "var byline = \"by \" + author;\nvar section = \"main\";",
		"blog/_defaults.jsx": `var section = "blog";`,
		"blog/a.md":          "---\ntitle: A\n---\n[[ page \"byline\" ]] in [[ page \"section\" ]]\n",
		"blog/b.md":          "---\ntitle: B\nsection: custom\n---\nb\n",
		"blog/index.jsx":     `<ul>{pages("blog").map(p => <li>{p.title}:{p.section}</li>)}</ul>`,
	})
	assertContains(t, mustRender(t, dir, "blog/a"), "by Jeff in blog")
	assertContains(t, mustRender(t, dir, "blog"), "<li>A:blog</li>", "<li>B:custom</li>")
}

func TestGlobalsAndDefaultsErrors(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"globals syntax":      {"index.jsx": `<p/>`, "_globals.jsx": `var x = ;`},
		"globals throws":      {"index.jsx": `<p/>`, "_globals.jsx": `var x = nope.y;`},
		"defaults syntax":     {"index.jsx": `<p/>`, "_defaults.jsx": `var x = ;`},
		"defaults throws":     {"index.jsx": `<p/>`, "_defaults.jsx": `var x = nope.y;`},
		"globals unreadable":  {"index.jsx": `<p/>`, "_globals.jsx/x": ``},
		"defaults unreadable": {"index.jsx": `<p/>`, "_defaults.jsx/x": ``},
	} {
		t.Run(name, func(t *testing.T) {
			if err := NewSite(writeSite(t, files), false).ParseAll(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestTemplatePages(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_defaults.jsx":      `var section = "Docs";`,
		"_partials/nav.tmpl": `<nav>[[ "x" ]]</nav>`,
		"_partials/hi.jsx":   `(name) => <b>hi {name}</b>`,
		"page.tmpl":          "---\ntitle: Tmpl\n---\n<h1>[[ page \"section\" ]]</h1>[[ partial \"_partials/nav\" ]][[ partial \"_partials/hi\" \"you\" ]][[ safeHTML \"<i>ok</i>\" ]]",
		"plain.tmpl":         `<p>no frontmatter</p>`,
	})
	assertContains(t, mustRender(t, dir, "page"), "<h1>Docs</h1>", "<nav>x</nav>", "<b>hi you</b>", "<i>ok</i>")
	assertContains(t, mustRender(t, dir, "plain"), "<p>no frontmatter</p>")
}

func TestTemplateErrors(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"bad syntax":        {"page.tmpl": `[[ if ]]`},
		"missing page key":  {"page.tmpl": `[[ page "nope" ]]`},
		"missing partial":   {"page.tmpl": `[[ partial "_partials/nope" ]]`},
		"bad frontmatter":   {"page.tmpl": "---\ntitle: [\n---\nx"},
		"short frontmatter": {"page.md": "intro\n---\nno closing"},
		"markdown template": {"page.md": `[[ nope ]]`},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := render(t, writeSite(t, files), "page", false); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestTemplateLayout(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.tmpl":       "---\nsiteName: Mine\n---\n<html><head><title>[[ page \"title\" ]] - [[ page \"siteName\" ]]</title></head><body>[[ content ]][[ partial \"_partials/foot\" ]]</body></html>",
		"_partials/foot.jsx": `() => <footer>f</footer>`,
		"post.md":            "---\ntitle: Post\n---\nHello *there*\n",
	})
	assertContains(t, mustRender(t, dir, "post"),
		"<title>Post - Mine</title>",
		"<em>there</em>",
		"<footer>f</footer>",
	)
}

func TestNestedDefaultLayouts(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":      `(content) => <html><body class="root">{content}</body></html>`,
		"docs/_layout.jsx": `(content) => <article class="docs">{content}</article>`,
		"docs/a.md":        "# A\n",
		"index.jsx":        `<p>home</p>`,
	})
	out := mustRender(t, dir, "docs/a")
	assertContains(t, out, `<body class="root">`, `<article class="docs">`)
	if strings.Index(out, "root") > strings.Index(out, "docs") {
		t.Errorf("docs layout should be inside the root layout:\n%s", out)
	}
	if out := mustRender(t, dir, "."); strings.Contains(out, "docs") {
		t.Errorf("docs layout applied to the root page:\n%s", out)
	}
}

func TestExplicitLayoutChain(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":            `(content) => <div class="default">{content}</div>`,
		"_layouts/base.jsx":      `(content) => <html><body class="base">{content}</body></html>`,
		"_layouts/post.tmpl":     "---\nlayout: _layouts/base\n---\n<article>[[ content ]]</article>",
		"post.md":                "---\nlayout: _layouts/post\n---\nbody\n",
		"bare.jsx":               `data({layout: ""}); <p>bare</p>`,
		"missing.md":             "---\nlayout: _layouts/nope\n---\nx\n",
		"badlayout.md":           "---\nlayout: _layouts/bad\n---\nx\n",
		"_layouts/bad.jsx":       `(content) => <p>{{a: 1}}</p>`,
		"badtmpl.md":             "---\nlayout: _layouts/badtmpl\n---\nx\n",
		"_layouts/badtmpl.tmpl":  `[[ if ]]`,
		"badfront.md":            "---\nlayout: _layouts/badfront\n---\nx\n",
		"_layouts/badfront.tmpl": "---\na: [\n---\nx",
	})
	out := mustRender(t, dir, "post")
	assertContains(t, out, `<body class="base">`, "<article>", "<p>body</p>")
	if strings.Contains(out, "default") {
		t.Errorf("default layout should not apply when a layout is set:\n%s", out)
	}
	out = mustRender(t, dir, "bare")
	if strings.Contains(out, "default") || !strings.Contains(out, "<p>bare</p>") {
		t.Errorf("an empty layout should render without one:\n%s", out)
	}
	for _, page := range []string{"missing", "badlayout", "badtmpl", "badfront"} {
		if _, err := render(t, dir, page, false); err == nil {
			t.Errorf("%s: expected an error", page)
		}
	}
}

func TestRenderRestoresLayout(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx": `(content) => <main>{content}</main>`,
		"index.jsx":   `<p>home</p>`,
	})
	site := NewSite(dir, false)
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	var first, second bytes.Buffer
	if err := site.Page(".").Render(&first); err != nil {
		t.Fatal(err)
	}
	if err := site.Page(".").Render(&second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Errorf("second render differs:\n%s\nvs\n%s", first.String(), second.String())
	}
}

func TestNonHTMLPages(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"feed.xml.jsx":    `data({contentType: "application/xml", preamble: "<?xml version=\"1.0\"?>"}); <rss><item>  a  </item></rss>`,
		"robots.txt.tmpl": "---\ncontentType: text/plain\n---\nUser-agent: *",
	})
	if err := build(t, dir); err != nil {
		t.Fatal(err)
	}
	if got, want := readOut(t, dir, "feed.xml"), "<?xml version=\"1.0\"?>\n<rss><item>  a  </item></rss>"; got != want {
		t.Errorf("feed.xml: expected %q, got %q", want, got)
	}
	if got, want := readOut(t, dir, "robots.txt"), "\nUser-agent: *"; got != want {
		t.Errorf("robots.txt: expected %q, got %q", want, got)
	}
}

func TestPageParseErrors(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx": `<p>{</p>`,
	})
	err := NewSite(dir, false).ParseAll()
	if err == nil || !strings.Contains(err.Error(), "error parsing JSX") {
		t.Errorf("expected a JSX syntax error, got %v", err)
	}

	site := NewSite(writeSite(t, nil), false)
	if err := site.Page("nope").Parse(); err == nil || !strings.Contains(err.Error(), "unable to find page source") {
		t.Errorf("expected a missing source error, got %v", err)
	}
}

func TestJSXPanicIsAnError(t *testing.T) {
	dir := writeSite(t, map[string]string{"index.jsx": `<p/>`})
	site := NewSite(dir, false)
	p := site.Page(".")
	// a nil site makes partial() panic in Go, outside the JS runtime
	p.site = nil
	p.data[Source] = `<div>{partial("x")}</div>`
	p.globals = builtinGlobals(p)
	if _, _, err := new(JSXParser).Parse(p); err == nil {
		t.Fatal("expected the panic to be returned as an error")
	}
}

func TestDirectoryPages(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"notes/a.md": "---\ntitle: A\n---\na\n",
	})
	site := NewSite(dir, false)
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	if !site.Page("notes").IsDir() {
		t.Error("notes should be a directory page")
	}
	if site.Page("notes/a").IsDir() {
		t.Error("notes/a should not be a directory page")
	}
	captureStdout(t, func() {
		if err := site.GenerateAll(filepath.Join(dir, "_public"), true); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := os.Stat(filepath.Join(dir, "_public/notes/index.html")); err == nil {
		t.Error("a directory without an index should not be rendered")
	}
}

func TestDraftsAndOrdering(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"blog/b.md":      "---\ntitle: B\ndate: 2024-02-01\n---\nb\n",
		"blog/a.md":      "---\ntitle: A\ndate: 2024-03-01\n---\na\n",
		"blog/c.md":      "---\ntitle: C\ndate: 2023-01-01\n---\nc\n",
		"blog/draft.md":  "---\ntitle: Draft\ndraft: true\n---\nd\n",
		"blog/future.md": "---\ntitle: Future\ndate: 2999-01-01\n---\nf\n",
		"blog/nodate.md": "---\ntitle: NoDate\n---\nn\n",
		"index.jsx":      `<p>{pages("blog").map(p => p.title).join(",")}</p>`,
	})
	captureStdout(t, func() {
		if err := build(t, dir); err != nil {
			t.Fatal(err)
		}
	})
	for _, name := range []string{"blog/draft/index.html", "blog/future/index.html"} {
		if _, err := os.Stat(filepath.Join(dir, "_public", name)); err == nil {
			t.Errorf("%s should not be built", name)
		}
	}
	// pages without a date come first, then dated pages oldest first
	assertContains(t, readOut(t, dir, "index.html"), "<p>NoDate,C,B,A</p>")

	withDrafts, err := render(t, dir, ".", true)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, withDrafts, "Draft", "Future")

	site := NewSite(dir, false)
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	if got := site.Page("blog/b").Time().Format("2006-01-02"); got != "2024-02-01" {
		t.Errorf("Time: got %s", got)
	}
}

func TestPagesGlob(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"blog/a.md":  "---\ntitle: A\n---\na\n",
		"notes/b.md": "---\ntitle: B\n---\nb\n",
		"other.md":   "---\ntitle: O\n---\no\n",
		"index.jsx":  `<p>{pages("[bn]*").map(p => p.title).join(",")}|{pages("[").length}|{pages("nope").length}</p>`,
	})
	assertContains(t, mustRender(t, dir, "."), "<p>A,B|0|0</p>")
}

func TestGenerateAll(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx":     `<p>home</p>`,
		"img/logo.png":  `png`,
		"_public/keep":  `kept`,
		".hidden/x.jsx": `<p>hidden</p>`,
	})
	site := NewSite(dir, false)
	if err := site.GenerateAll("", false); err == nil {
		t.Error("expected an error for an empty dest")
	}
	captureStdout(t, func() {
		if err := site.GenerateAll(filepath.Join(dir, "_public"), false); err != nil {
			t.Fatal(err)
		}
	})
	if got := readOut(t, dir, "keep"); got != "kept" {
		t.Error("existing output should be kept without clobber")
	}
	if got := readOut(t, dir, "img/logo.png"); got != "png" {
		t.Errorf("static file not copied: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "_public/.hidden")); err == nil {
		t.Error("dot directories should not be built")
	}

	bad := writeSite(t, map[string]string{"index.jsx": `<p>{nope}</p>`})
	if err := build(t, bad); err == nil {
		t.Error("expected a render error to fail the build")
	}
}

func TestRawNodeString(t *testing.T) {
	if got := rawNode("<b>x</b>").String(); got != "<b>x</b>" {
		t.Errorf("got %q", got)
	}
	n := hyperNode{tag: "p", children: []hyperNode{{text: "a & b"}}}
	if got := n.String(); got != "<p>a &amp; b</p>" {
		t.Errorf("got %q", got)
	}
}

func TestMoreErrorPaths(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx":         `<p>home</p>`,
		".DS_Store":         `junk`,
		"_assets/.git/x":    `skip`,
		"blog/2024/deep.md": "---\ntitle: Deep\n---\nd\n",
		"blog/a.md":         "---\ntitle: A\n---\na\n",
		"list.jsx":          `<p>{pages("blog").map(p => p.title).join(",")}|{pages("x*").length}</p>`,
	})
	assertContains(t, mustRender(t, dir, "list"), "<p>,A|0</p>")

	// rendering a page with no source fails
	site := NewSite(dir, false)
	if err := site.Page("nope").Render(&bytes.Buffer{}); err == nil {
		t.Error("expected an error rendering a missing page")
	}

	// a top-level value that isn't a node
	if _, err := RenderJSX("bad", []byte(`({a: 1})`), nil); err == nil {
		t.Error("expected an error for a non-node result")
	}
	// an unterminated data() statement in a page
	bad := writeSite(t, map[string]string{"index.jsx": `data({title: "x"`})
	if err := NewSite(bad, false).ParseAll(); err == nil {
		t.Error("expected an error for an unterminated data() statement")
	}

	// a layout error only shows up when rendering, so it fails the build there
	bad = writeSite(t, map[string]string{"index.jsx": `data({layout: "_nope"}); <p/>`})
	captureStdout(t, func() {
		if err := build(t, bad); err == nil {
			t.Error("expected a missing layout to fail the build")
		}
	})

	// dest exists but isn't a directory
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, []byte("x"), 0644)
	if err := site.GenerateAll(file, true); err == nil {
		t.Error("expected an error cleaning a file dest")
	}
	if err := site.GenerateAll(filepath.Join(file, "out"), false); err == nil {
		t.Error("expected an error creating a dest under a file")
	}

	captureStdout(t, func() {
		if err := build(t, dir); err != nil {
			t.Fatal(err)
		}
	})
	for _, name := range []string{".DS_Store", "_assets/.git"} {
		if _, err := os.Stat(filepath.Join(dir, "_public", name)); err == nil {
			t.Errorf("%s should not be in the output", name)
		}
	}
}
