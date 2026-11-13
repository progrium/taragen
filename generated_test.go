package taragen

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tagTemplate = "params(() => pages(\"posts\").flatMap(p => p.tags ?? []).map(tag => ({tag})));\n" +
	"data({title: `#${page.params.tag}`});\n" +
	"<ul>{pages({path: \"posts\", where: {tags: page.params.tag}}).map(p => <li>{p.title}</li>)}</ul>"

func taggedPosts() map[string]string {
	return map[string]string{
		"posts/a.md": "---\ntitle: A\ntags: [go, web]\n---\nA\n",
		"posts/b.md": "---\ntitle: B\ntags: [go]\n---\nB\n",
	}
}

func withFiles(base map[string]string, more map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range more {
		out[k] = v
	}
	return out
}

func expectBuildError(t *testing.T, files map[string]string, wants ...string) {
	t.Helper()
	err := buildQuiet(t, writeSite(t, files))
	if err == nil {
		t.Fatalf("expected the build to fail mentioning %q", wants)
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}

func TestGeneratedTag(t *testing.T) {
	dir := writeSite(t, withFiles(taggedPosts(), map[string]string{
		"tags/[tag].jsx": tagTemplate,
	}))
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "tags/go/index.html"), "<li>A</li>", "<li>B</li>")
	web := readOut(t, dir, "tags/web/index.html")
	assertContains(t, web, "<li>A</li>")
	if strings.Contains(web, "<li>B</li>") {
		t.Errorf("tags/web lists B:\n%s", web)
	}
	if exists(dir, "tags/[tag]") || exists(dir, "tags/[tag]/index.html") {
		t.Error("the template itself was built as a page")
	}

	site := parsed(t, dir, false)
	p := site.Page("tags/web")
	if p.data["title"] != "#web" || p.Slug() != "web" || p.Template() != "tags/[tag].jsx" {
		t.Errorf("tags/web: title %v, slug %q, template %q", p.data["title"], p.Slug(), p.Template())
	}
	if got := p.data[Params].(map[string]any)["tag"]; got != "web" {
		t.Errorf("page.params.tag: %v", got)
	}
}

// A _defaults.jsx exports only the names it defines, so it can't replace a
// generated page's params with the params() builtin it was run alongside.
func TestGeneratedWithDefaults(t *testing.T) {
	dir := writeSite(t, withFiles(taggedPosts(), map[string]string{
		"_defaults.jsx":  "siteURL = \"https://example.com\";\n",
		"tags/[tag].jsx": tagTemplate,
	}))
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "tags/go/index.html"), "<li>A</li>", "<li>B</li>")

	p := parsed(t, dir, false).Page("tags/web")
	if p.data["title"] != "#web" || p.data["siteURL"] != "https://example.com" {
		t.Errorf("tags/web: title %v, siteURL %v", p.data["title"], p.data["siteURL"])
	}
	if _, ok := p.data["partial"]; ok {
		t.Error("a builtin global was copied into page data by _defaults.jsx")
	}
}

func TestGeneratedNested(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"logs/2026/10-30.md": "---\ndate: 2026-10-30\n---\nOct\n",
		"logs/2026/11-01.md": "---\ndate: 2026-11-01\n---\nNov 1\n",
		"logs/2026/11-09.md": "---\ndate: 2026-11-09\n---\nNov 9\n",
		"logs/[year]/[month].jsx": "params(() => pages(\"logs/*\").map(l => ({year: l.date.slice(0, 4), month: l.date.slice(5, 7)})));\n" +
			"<ul>{pages(\"logs/*\").filter(l => !l.params && l.date.startsWith(`${page.params.year}-${page.params.month}`)).map(l => <li>{l.date}</li>)}</ul>",
	})
	mustBuild(t, dir)
	nov := readOut(t, dir, "logs/2026/11/index.html")
	assertContains(t, nov, "<li>2026-11-01</li>", "<li>2026-11-09</li>")
	if strings.Contains(nov, "2026-10-30") {
		t.Errorf("November lists October:\n%s", nov)
	}
	assertContains(t, readOut(t, dir, "logs/2026/10/index.html"), "<li>2026-10-30</li>")

	site := parsed(t, dir, false)
	if got := site.Page("logs/2026/11").Slug(); got != "11" {
		t.Errorf("slug: %q", got)
	}
}

func TestGeneratedFillsDir(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"logs/2026/11-09.md":    "---\ntitle: Nov 9\n---\nx\n",
		"logs/2026/11-10.md":    "---\ntitle: Nov 10\n---\ny\n",
		"logs/[year]/index.jsx": "params([{year: 2026}]);\n<ul>{pages(`logs/${page.params.year}`).map(l => <li>{l.title}</li>)}</ul>",
	})
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "logs/2026/index.html"), "<li>Nov 9</li>", "<li>Nov 10</li>")

	site := parsed(t, dir, false)
	p := site.Page("logs/2026")
	if !p.IsDir() || p.Template() != "logs/[year]/index.jsx" {
		t.Errorf("logs/2026: isDir %v, template %q", p.IsDir(), p.Template())
	}
	if got := p.data[Params].(map[string]any)["year"]; got != "2026" {
		t.Errorf("a number value should become a string, got %#v", got)
	}
}

func TestGeneratedMissingParent(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"logs/2026/11-09.md":      "---\ndate: 2026-11-09\n---\nx\n",
		"logs/[year]/[month].jsx": "params([{year: \"2026\", month: \"11\"}, {year: \"2027\", month: \"01\"}]);\n<p>{page.path}</p>",
		"index.jsx":               "<ul>{pages(\"logs/*\").map(p => <li>{p.path}</li>)}</ul>",
	})
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "logs/2027/01/index.html"), "<p>/logs/2027/01</p>")
	assertContains(t, readOut(t, dir, "index.html"), "<li>/logs/2027/01</li>", "<li>/logs/2026/11</li>", "<li>/logs/2026/11-09</li>")
	if exists(dir, "logs/2027/index.html") {
		t.Error("a bare directory was built")
	}
	site := parsed(t, dir, false)
	if p := site.Page("logs/2027"); !p.IsDir() || !p.virtualDir {
		t.Errorf("logs/2027 should be a bare directory")
	}
}

func TestGeneratedRealWins(t *testing.T) {
	dir := writeSite(t, withFiles(taggedPosts(), map[string]string{
		"tags/[tag].jsx": tagTemplate,
		"tags/go.md":     "# The Go hub\n",
	}))
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "tags/go/index.html"), "<h1>The Go hub</h1>")
	assertContains(t, readOut(t, dir, "tags/web/index.html"), "<li>A</li>")
	if p := parsed(t, dir, false).Page("tags/go"); p.Template() != "" || p.data[Params] != nil {
		t.Error("the real page should not be generated")
	}
}

func TestGeneratedCollision(t *testing.T) {
	expectBuildError(t, map[string]string{
		"a/[x].jsx":       `params([{x: "one"}]); <p/>`,
		"a/[y]/index.jsx": `params([{y: "one"}]); <p/>`,
	}, "/a/one", "a/[x].jsx", "a/[y]/index.jsx")
	expectBuildError(t, map[string]string{
		"a/[x].jsx":        `params([{x: "one"}]); <p/>`,
		"a/one/index.html": `static`,
	}, "route collision", "a/one/index.html", "a/[x].jsx")
}

func TestGeneratedParamsRealOnly(t *testing.T) {
	dir := writeSite(t, withFiles(taggedPosts(), map[string]string{
		"tags/[tag].jsx": tagTemplate,
		// sorts after tags/[tag].jsx, and would see its pages if params
		// could see generated pages
		"zz/[n].jsx":     "params(() => pages(\"tags\").map(p => ({n: p.slug})));\n<p/>",
		"tags/index.jsx": "<ul>{pages(\"tags\").map(p => <li>{p.slug}</li>)}</ul>",
	}))
	mustBuild(t, dir)
	if exists(dir, "zz/go/index.html") {
		t.Error("params saw a generated page")
	}
	assertContains(t, readOut(t, dir, "tags/index.html"), "<li>go</li>", "<li>web</li>")
}

func TestGeneratedDrafts(t *testing.T) {
	files := map[string]string{
		"logs/2026/10-01.md":      "---\ndate: 2026-10-01\n---\nx\n",
		"logs/2026/11-01.md":      "---\ndate: 2026-11-01\ndraft: true\n---\ny\n",
		"logs/[year]/[month].jsx": "params(() => pages(\"logs/*\").map(l => ({year: l.date.slice(0, 4), month: l.date.slice(5, 7)})));\n<p/>",
		"secret/[s].jsx":          "params([{s: \"x\"}]); data({draft: true}); <p/>",
	}
	dir := writeSite(t, files)
	mustBuild(t, dir)
	if !exists(dir, "logs/2026/10/index.html") || exists(dir, "logs/2026/11/index.html") {
		t.Error("build: a month holding only a draft should have no page")
	}
	if exists(dir, "secret/x/index.html") {
		t.Error("build: a generated draft was written")
	}

	site := parsed(t, dir, true)
	if rec := serve(t, site, "/logs/2026/11"); rec.Code != 200 {
		t.Errorf("serve: the draft month should have a page, got %d", rec.Code)
	}
	if rec := serve(t, site, "/secret/x"); rec.Code != 200 {
		t.Errorf("serve: generated draft: %d", rec.Code)
	}
}

func TestGeneratedParamErrors(t *testing.T) {
	for name, set := range map[string]string{
		"missing key": `{}`,
		"extra key":   `{tag: "a", other: "b"}`,
		"slash":       `{tag: "a/b"}`,
		"dotdot":      `{tag: ".."}`,
		"underscore":  `{tag: "_a"}`,
		"dot":         `{tag: ".a"}`,
		"empty":       `{tag: ""}`,
		"backslash":   `{tag: "a\\b"}`,
		"bool":        `{tag: true}`,
		"not object":  `"a"`,
	} {
		t.Run(name, func(t *testing.T) {
			expectBuildError(t, map[string]string{
				"tags/[tag].jsx": "params([" + set + "]); <p/>",
			}, "tags/[tag].jsx")
		})
	}
	for name, src := range map[string]string{
		"not an array": `params({tag: "a"}); <p/>`,
		"no params":    `data({}); <p/>`,
		"two params":   `params([]); params([]); <p/>`,
		"throws":       `params(() => { throw new Error("boom") }); <p/>`,
		"in body":      "params([{tag: \"a\"}]);\n<p>{params([])}</p>",
	} {
		t.Run(name, func(t *testing.T) {
			expectBuildError(t, map[string]string{"tags/[tag].jsx": src}, "params")
		})
	}
	expectBuildError(t, map[string]string{"about.jsx": `params([]); <p/>`}, "about.jsx", "only allowed in a template")
	expectBuildError(t, map[string]string{
		"_layout.jsx": `(c) => <div>{params([])}{c}</div>`,
		"about.jsx":   `<p/>`,
	}, "params() is only allowed")
}

func TestGeneratedBadNames(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"partial bracket":    {"logs/[year]-archive.jsx": `params([]); <p/>`},
		"bad name":           {"logs/[Year].jsx": `params([]); <p/>`},
		"bracket in ext":     {"logs/x.[y].jsx": `<p/>`},
		"markdown template":  {"tags/[tag].md": "# x\n"},
		"go template":        {"tags/[tag].tmpl": "x"},
		"static in dir":      {"logs/[year]/index.jsx": `params([]); <p/>`, "logs/[year]/style.css": "x"},
		"underscore in dir":  {"logs/[year]/index.jsx": `params([]); <p/>`, "logs/[year]/_layout.jsx": "(c) => c"},
		"repeated parameter": {"a/[x]/[x].jsx": `params([]); <p/>`},
		"bad dir name":       {"a/[x]y/index.jsx": `<p/>`},
	} {
		t.Run(name, func(t *testing.T) {
			expectBuildError(t, files)
		})
	}
}

func TestGeneratedDedupe(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"a/[x].jsx": `params([{x: "one"}, {x: "one"}, {x: 1}, {x: "1"}]); <p>{page.params.x}</p>`,
		"b/[y].jsx": `params([]); <p/>`,
	})
	mustBuild(t, dir)
	if !exists(dir, "a/one/index.html") || !exists(dir, "a/1/index.html") {
		t.Error("missing generated pages")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "_public", "a"))
	if len(entries) != 2 {
		t.Errorf("expected 2 pages under a/, got %d", len(entries))
	}
	if exists(dir, "b") {
		t.Error("an empty array should generate nothing")
	}
}

func TestGeneratedOutputs(t *testing.T) {
	dir := writeSite(t, withFiles(taggedPosts(), map[string]string{
		"_outputs/txt.jsx": "data({contentType: \"text/plain\"}); `txt for ${page.path}`",
		"tags/[tag].jsx":   "params([{tag: \"go\"}]);\n<a href={page.outputs.txt}>txt</a>",
	}))
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "tags/go/index.html"), `<a href="/tags/go.txt">txt</a>`)
	if got := readOut(t, dir, "tags/go.txt"); got != "txt for /tags/go" {
		t.Errorf("tags/go.txt: %q", got)
	}
}

func TestGeneratedLayouts(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":             `(c) => <html><body>{c}</body></html>`,
		"logs/_layout.jsx":        `(c) => <main class={page.kind}>{c}</main>`,
		"logs/_defaults.jsx":      `var kind = "log";`,
		"logs/2026/_defaults.jsx": `var year = "y2026";`,
		"logs/2026/11-09.md":      "x\n",
		"logs/[year]/[month].jsx": "params([{year: \"2026\", month: \"11\"}]);\n<p>{page.year}</p>",
	})
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "logs/2026/11/index.html"), "<body>", `<main class="log">`, "<p>y2026</p>")
}

func TestServeGenerated(t *testing.T) {
	dir := writeSite(t, withFiles(taggedPosts(), map[string]string{
		"_layout.jsx":    `(c) => <html><body>{c}</body></html>`,
		"tags/[tag].jsx": tagTemplate,
	}))
	site := parsed(t, dir, false)
	rec := serve(t, site, "/tags/go")
	if rec.Code != 200 {
		t.Fatalf("/tags/go: %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "<li>A</li>", `new WebSocket("/.live-reload")`)
	if rec := serve(t, site, "/tags/nosuchtag"); rec.Code != http.StatusNotFound {
		t.Errorf("/tags/nosuchtag: %d", rec.Code)
	}
	if rec := serve(t, site, "/tags/[tag]"); rec.Code == 200 {
		t.Errorf("the template itself was served")
	}

	if err := os.WriteFile(filepath.Join(dir, "posts/c.md"), []byte("---\ntitle: C\ntags: [new]\n---\nC\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if reparse, _ := classifyChange("posts/c.md", false); !reparse {
		t.Fatal("a Markdown change should re-parse")
	}
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	assertContains(t, serve(t, site, "/tags/new").Body.String(), "<li>C</li>")
}

func TestGeneratedLinksAndAliases(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"about.jsx":      "<p>{(page.backlinks ?? []).map(p => p.path).join(\",\")}</p>",
		"tags/[tag].jsx": "params([{tag: \"go\"}]);\ndata({aliases: [`/t/${page.params.tag}`]});\n<a href=\"/about\">about</a>",
		"index.jsx":      "<a href=\"/tags/go\">go</a>",
	})
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "about/index.html"), "/tags/go")
	site := parsed(t, dir, false)
	if got := site.Page("tags/go").data["backlinks"]; got == nil {
		t.Error("a link to a generated page should make a backlink")
	}
	if rec := serve(t, site, "/t/go"); rec.Code != http.StatusMovedPermanently {
		t.Errorf("alias of a generated page: status %d", rec.Code)
	}
}
