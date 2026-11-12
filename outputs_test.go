package taragen

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const txtTemplate = "data({ contentType: \"text/plain; charset=utf-8\" });\n`${page.title ?? page.path}\\n\\n${page.src}`"

func buildQuiet(t *testing.T, dir string) error {
	t.Helper()
	var err error
	captureStdout(t, func() { err = build(t, dir) })
	return err
}

func mustBuild(t *testing.T, dir string) {
	t.Helper()
	if err := buildQuiet(t, dir); err != nil {
		t.Fatal(err)
	}
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, "_public", name))
	return err == nil
}

func parsed(t *testing.T, dir string, drafts bool) *Site {
	t.Helper()
	site := NewSite(dir, drafts)
	var err error
	captureStdout(t, func() { err = site.ParseAll() })
	if err != nil {
		t.Fatal(err)
	}
	return site
}

func TestOutputTxt(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":      `(content) => <html><body>{content}</body></html>`,
		"_outputs/txt.jsx": txtTemplate,
		"index.jsx":        `data({title: "Home"}); <p>home & away</p>`,
		"about.md":         "---\ntitle: About <me>\n---\n# About\n",
	})
	mustBuild(t, dir)
	if got, want := readOut(t, dir, "about.txt"), "About <me>\n\n---\ntitle: About <me>\n---\n# About\n"; got != want {
		t.Errorf("about.txt: expected %q, got %q", want, got)
	}
	if got, want := readOut(t, dir, "index.txt"), "Home\n\ndata({title: \"Home\"}); <p>home & away</p>"; got != want {
		t.Errorf("index.txt: expected %q, got %q", want, got)
	}
	// the page itself is unchanged and keeps its layout
	assertContains(t, readOut(t, dir, "about/index.html"), "<body>", "<h1>About</h1>")
}

func TestOutputSuffix(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/card.png.jsx": "`card for ${page.path}`",
		"about.jsx":             `<a href={page.outputs.card}>card</a>`,
	})
	mustBuild(t, dir)
	if got := readOut(t, dir, "about.card.png"); got != "card for /about" {
		t.Errorf("about.card.png: got %q", got)
	}
	assertContains(t, readOut(t, dir, "about/index.html"), `<a href="/about.card.png">card</a>`)

	site := parsed(t, dir, false)
	want := map[string]any{"card": "/about.card.png"}
	if got := site.Page("about").data[Outputs]; !reflect.DeepEqual(got, want) {
		t.Errorf("page.outputs: expected %v, got %v", want, got)
	}
}

func TestOutputNearestWins(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx":         "`root`",
		"_outputs/card.png.jsx":    "`root card`",
		"logs/_outputs/txt.jsx":    "`logs`",
		"logs/_outputs/card.jsx":   "`logs card`",
		"about.jsx":                `<p/>`,
		"logs/index.jsx":           `<p/>`,
		"logs/2024/first.md":       "first\n",
		"notes/second.md":          "second\n",
		"notes/_outputs/.skip.jsx": "`ignored`",
	})
	mustBuild(t, dir)
	for name, want := range map[string]string{
		"about.txt":            "root",
		"notes/second.txt":     "root",
		"logs.txt":             "logs",
		"logs/2024/first.txt":  "logs",
		"about.card.png":       "root card",
		"logs/2024/first.card": "logs card",
	} {
		if got := readOut(t, dir, name); got != want {
			t.Errorf("%s: expected %q, got %q", name, want, got)
		}
	}
	// the nearer card template has a different suffix, which replaces the
	// root one
	if exists(dir, "logs/2024/first.card.png") {
		t.Error("logs/2024/first.card.png should not be written")
	}
	site := parsed(t, dir, false)
	want := map[string]any{"txt": "/logs/2024/first.txt", "card": "/logs/2024/first.card"}
	if got := site.Page("logs/2024/first").data[Outputs]; !reflect.DeepEqual(got, want) {
		t.Errorf("page.outputs: expected %v, got %v", want, got)
	}
}

func TestOutputBadTemplateName(t *testing.T) {
	for name, file := range map[string]string{
		"uppercase":   "_outputs/Txt.jsx",
		"digit first": "_outputs/1x.jsx",
		"dash":        "_outputs/plain-text.jsx",
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeSite(t, map[string]string{file: "``", "index.jsx": `<p/>`})
			if err := buildQuiet(t, dir); err == nil || !strings.Contains(err.Error(), "invalid output template name") {
				t.Errorf("expected an invalid name error, got %v", err)
			}
		})
	}
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx":    "``",
		"_outputs/txt.md.jsx": "``",
		"index.jsx":           `<p/>`,
	})
	if err := buildQuiet(t, dir); err == nil || !strings.Contains(err.Error(), `both define output "txt"`) {
		t.Errorf("expected a duplicate name error, got %v", err)
	}
}

func TestOutputOptOut(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx":      "`txt`",
		"_outputs/card.png.jsx": "`card`",
		"all.md":                "all\n",
		"notxt.md":              "---\noutputs: { txt: false }\n---\nx\n",
		"none.md":               "---\noutputs: false\n---\nx\n",
		"jsx.jsx":               `data({outputs: {card: false, nosuch: false}}); <p>{JSON.stringify(page.outputs)}</p>`,
		"quiet/_defaults.jsx":   `outputs = {card: false}`,
		"quiet/a.md":            "a\n",
		"quiet/b.md":            "---\noutputs: false\n---\nb\n",
		"quiet/c.md":            "---\noutputs: { txt: true }\n---\nc\n",
		"off/_defaults.jsx":     `outputs = false`,
		"off/a.jsx":             `<p/>`,
	})
	mustBuild(t, dir)
	site := parsed(t, dir, false)
	for page, want := range map[string]map[string]any{
		"all":     {"txt": "/all.txt", "card": "/all.card.png"},
		"notxt":   {"card": "/notxt.card.png"},
		"none":    {},
		"jsx":     {"txt": "/jsx.txt"},
		"quiet/a": {"txt": "/quiet/a.txt"},
		"quiet/b": {},
		"quiet/c": {"txt": "/quiet/c.txt", "card": "/quiet/c.card.png"},
		"off/a":   {},
	} {
		if got := site.Page(page).data[Outputs]; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: page.outputs expected %v, got %v", page, want, got)
		}
		for _, name := range []string{"txt", "card.png"} {
			file := page + "." + name
			_, on := want[strings.TrimSuffix(name, ".png")]
			if exists(dir, file) != on {
				t.Errorf("%s: expected written=%v", file, on)
			}
		}
	}
	// a page reading page.outputs never sees the opt-out it set
	assertContains(t, readOut(t, dir, "jsx/index.html"), `<p>{"txt":"/jsx.txt"}</p>`)
}

func TestOutputSkips(t *testing.T) {
	files := map[string]string{
		"_outputs/txt.jsx": "`txt`",
		"index.jsx":        `<p/>`,
		"draft.md":         "---\ndraft: true\n---\nx\n",
		"future.md":        "---\ndate: 2999-01-01\n---\nx\n",
		"bare/a.md":        "a\n",
		"feed.xml.jsx":     `data({layout: "", contentType: "application/xml"}); <rss/>`,
	}
	dir := writeSite(t, files)
	mustBuild(t, dir)
	for _, name := range []string{"draft.txt", "future.txt", "bare.txt", "feed.xml.txt", "feed.txt"} {
		if exists(dir, name) {
			t.Errorf("%s should not be written", name)
		}
	}
	for _, name := range []string{"index.txt", "bare/a.txt"} {
		if !exists(dir, name) {
			t.Errorf("%s should be written", name)
		}
	}

	site := parsed(t, dir, true)
	if rec := serve(t, site, "/draft.txt"); rec.Code != 200 || rec.Body.String() != "txt" {
		t.Errorf("serve draft.txt: status %d: %q", rec.Code, rec.Body.String())
	}
	for _, url := range []string{"/feed.xml.txt", "/bare.txt"} {
		if rec := serve(t, site, url); rec.Code != http.StatusNotFound {
			t.Errorf("serve %s: status %d", url, rec.Code)
		}
	}
	if got := site.Page("feed.xml").data[Outputs]; !reflect.DeepEqual(got, map[string]any{}) {
		t.Errorf("feed.xml: page.outputs should be empty, got %v", got)
	}
	if got := site.Page("bare").data[Outputs]; !reflect.DeepEqual(got, map[string]any{}) {
		t.Errorf("bare: page.outputs should be empty, got %v", got)
	}
}

func TestOutputsMap(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx": "`txt`",
		"_layout.jsx":      `(content) => <html><head><link rel="alternate" type="text/plain" href={page.outputs.txt}/></head><body>{content}</body></html>`,
		"index.jsx":        `<ul>{pages("posts").map(p => <li><a href={p.outputs.txt}>{p.slug}</a></li>)}</ul>`,
		"posts/a.md":       "a\n",
		"posts/b.md":       "---\noutputs: false\n---\nb\n",
		"self.jsx":         `<p>{page.outputs.txt}</p>`,
	})
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "posts/a/index.html"), `<link rel="alternate" type="text/plain" href="/posts/a.txt"/>`)
	if out := readOut(t, dir, "posts/b/index.html"); strings.Contains(out, "href") {
		t.Errorf("posts/b should have no alternate link:\n%s", out)
	}
	assertContains(t, readOut(t, dir, "index.html"), `<a href="/posts/a.txt">a</a>`, `<a>b</a>`)
	assertContains(t, readOut(t, dir, "self/index.html"), `<p>/self.txt</p>`)
}

func TestOutputContentType(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx":      "`txt`",
		"_outputs/card.png.jsx": "`png`",
		"_outputs/raw.zzz.jsx":  "`?`",
		"_outputs/json.jsx":     `data({contentType: "application/json", preamble: "// json"}); "{}"`,
		"_outputs/htm.html.jsx": `() => <html><body><p>hi</p></body></html>`,
		"index.jsx":             `<p/>`,
	})
	site := parsed(t, dir, false)
	for url, want := range map[string]string{
		"/index.txt":      "text/plain; charset=utf-8",
		"/index.card.png": "image/png",
		"/index.raw.zzz":  "application/octet-stream",
		"/index.json":     "application/json",
		"/index.htm.html": "text/html; charset=utf-8",
	} {
		rec := serve(t, site, url)
		if rec.Code != 200 || rec.Header().Get("Content-Type") != want {
			t.Errorf("%s: status %d, expected type %q, got %q", url, rec.Code, want, rec.Header().Get("Content-Type"))
		}
	}
	if got := serve(t, site, "/index.json").Body.String(); got != "// json\n{}" {
		t.Errorf("json preamble: got %q", got)
	}
	// HTML outputs get live reload; others never do
	assertContains(t, serve(t, site, "/index.htm.html").Body.String(), "<p>hi</p>", "/.live-reload")
	mustBuild(t, dir)
	if got := readOut(t, dir, "index.htm.html"); strings.Contains(got, "live-reload") || strings.Contains(got, "DOCTYPE") {
		t.Errorf("built html output: %q", got)
	}
}

func TestOutputDataIsolated(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx": `data({title: "changed", contentType: "text/plain"}); page.title`,
		"_layout.jsx":      `(content) => <html><head><title>{page.title}</title></head><body>{content}</body></html>`,
		"index.jsx":        `data({title: "Home"}); <p/>`,
	})
	site := parsed(t, dir, false)
	if got := serve(t, site, "/index.txt").Body.String(); got != "Home" {
		t.Errorf("output sees page.title %q", got)
	}
	if got := site.Page(".").data["title"]; got != "Home" {
		t.Errorf("page title changed to %v", got)
	}
	if ct := site.Page(".").ContentType(); ct != "text/html" {
		t.Errorf("page content type changed to %v", ct)
	}
	assertContains(t, serve(t, site, "/").Body.String(), "<title>Home</title>")
}

func TestOutputErrors(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx": `page.nope.x`,
		"index.jsx":        `<p/>`,
	})
	err := buildQuiet(t, dir)
	if err == nil || !strings.Contains(err.Error(), "output txt of /") {
		t.Errorf("expected the build to fail naming the output, got %v", err)
	}
	site := parsed(t, dir, false)
	if rec := serve(t, site, "/index.txt"); rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "output txt") {
		t.Errorf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRouteCollision(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"output and static file": {
			"_outputs/txt.jsx": "`x`",
			"foo.jsx":          `<p/>`,
			"foo.txt":          `static`,
		},
		"output and page": {
			"_outputs/txt.jsx": "`x`",
			"foo.jsx":          `<p/>`,
			"foo.txt.jsx":      `data({contentType: "text/plain"}); "page"`,
		},
		"page and static file": {
			"foo.jsx":        `<p/>`,
			"foo/index.html": `static`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeSite(t, files)
			err := buildQuiet(t, dir)
			if err == nil {
				t.Fatal("expected the build to fail")
			}
			msg := err.Error()
			if !strings.Contains(msg, "route collision") || strings.Count(msg, "foo") < 3 {
				t.Errorf("error should name both producers: %v", err)
			}
		})
	}
}

func TestRouteSingleRender(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx": "`x`",
		"docs/index.jsx":   `<p>docs</p>`,
		"docs/a.md":        "# A\n",
	})
	var err error
	out := captureStdout(t, func() { err = build(t, dir) })
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"rendered: docs\n", "rendered: docs.txt\n", "rendered: docs/a\n"} {
		if n := strings.Count(out, line); n != 1 {
			t.Errorf("%q printed %d times:\n%s", line, n, out)
		}
	}
	site := parsed(t, dir, false)
	var urls []string
	for _, r := range site.Routes().Routes() {
		urls = append(urls, string(r.Kind)+" "+r.URL+" "+r.File)
	}
	want := []string{
		"output /docs.txt docs.txt",
		"output /docs/a.txt docs/a.txt",
		"page /docs/a docs/a/index.html",
		"page /docs docs/index.html",
	}
	if !reflect.DeepEqual(urls, want) {
		t.Errorf("routes:\n%s\nexpected:\n%s", strings.Join(urls, "\n"), strings.Join(want, "\n"))
	}
}

func TestServeOutput(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":      `(content) => <html><body>{content}</body></html>`,
		"_outputs/txt.jsx": "`one`",
		"about.jsx":        `<p>about</p>`,
		"notes.txt":        `static notes`,
	})
	site := parsed(t, dir, true)
	rec := serve(t, site, "/about.txt")
	if rec.Code != 200 || rec.Body.String() != "one" {
		t.Errorf("about.txt: status %d: %q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "WebSocket") {
		t.Error("live reload script injected into a text output")
	}
	if got := serve(t, site, "/notes.txt").Body.String(); got != "static notes" {
		t.Errorf("static file: %q", got)
	}

	// template edits show on the next request
	os.WriteFile(filepath.Join(dir, "_outputs/txt.jsx"), []byte("`two`"), 0644)
	if got := serve(t, site, "/about.txt").Body.String(); got != "two" {
		t.Errorf("after edit: %q", got)
	}
	// a new template shows after the watcher re-parses
	os.WriteFile(filepath.Join(dir, "_outputs/md.jsx"), []byte("page.src"), 0644)
	if rec := serve(t, site, "/about.md"); rec.Code != http.StatusNotFound {
		t.Errorf("about.md before reparse: status %d", rec.Code)
	}
	if reparse, _ := classifyChange("_outputs/md.jsx", false); !reparse {
		t.Error("a change to an output template should re-parse")
	}
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	if got := serve(t, site, "/about.md").Body.String(); got != "<p>about</p>" {
		t.Errorf("after reparse: %q", got)
	}
}
