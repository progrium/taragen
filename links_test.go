package taragen

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// linksOf returns a page's links data as strings.
func linksOf(t *testing.T, site *Site, pagepath string) []string {
	t.Helper()
	links, ok := site.Page(pagepath).data[Links].([]any)
	if !ok {
		t.Fatalf("%s: links is %T, not a list", pagepath, site.Page(pagepath).data[Links])
	}
	out := []string{}
	for _, l := range links {
		out = append(out, l.(string))
	}
	return out
}

// backlinksOf returns the paths of a page's backlinks.
func backlinksOf(t *testing.T, site *Site, pagepath string) []string {
	t.Helper()
	backlinks, ok := site.Page(pagepath).data[Backlinks].([]any)
	if !ok {
		t.Fatalf("%s: backlinks is %T, not a list", pagepath, site.Page(pagepath).data[Backlinks])
	}
	out := []string{}
	for _, b := range backlinks {
		out = append(out, b.(map[string]any)[Path].(string))
	}
	return out
}

func assertList(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: expected %q, got %q", what, want, got)
	}
}

func TestLinksFromBody(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx": `(content) => <html><body><nav><a href="/nav">nav</a></nav>{content}</body></html>`,
		"nav.md":      "# Nav\n",
		"a.md":        "# A\n\nSee [b](/b) and <a href=\"/c\">c</a>.\n",
		"b.jsx":       `<p><a href="/a">a</a></p>`,
		"c.tmpl":      `<p><a href="/a">a</a> <a href="/b">b</a></p>`,
	})
	site := parsed(t, dir, false)
	assertList(t, "a links", linksOf(t, site, "a"), "/b", "/c")
	assertList(t, "b links", linksOf(t, site, "b"), "/a")
	assertList(t, "c links", linksOf(t, site, "c"), "/a", "/b")
	assertList(t, "nav backlinks", backlinksOf(t, site, "nav"))
	assertList(t, "a backlinks", backlinksOf(t, site, "a"), "/b", "/c")
}

func TestLinksNormalize(t *testing.T) {
	for _, tt := range []struct {
		href, base, want string
	}{
		{"bar", "/foo/", "/foo/bar"},
		{"bar", "/feed.xml", "/bar"},
		{"../bar", "/docs/foo/", "/docs/bar"},
		{"./bar/", "/docs/", "/docs/bar"},
		{"/bar/", "/foo/", "/bar"},
		{"/bar#section", "/foo/", "/bar"},
		{"/bar?x=1#y", "/foo/", "/bar"},
		{"/b%61r", "/foo/", "/bar"},
		{"../../../bar", "/foo/", "/bar"},
		{"/", "/foo/", "/"},
		{"#top", "/foo/", "/foo"},
		{"?page=2", "/foo/", "/foo"},
		{"", "/foo/", ""},
	} {
		got, ok := normalizeLink(tt.href, tt.base, "")
		if tt.want == "" {
			if ok {
				t.Errorf("%q from %s: expected skip, got %q", tt.href, tt.base, got)
			}
			continue
		}
		if !ok || got != tt.want {
			t.Errorf("%q from %s: expected %q, got %q (ok %v)", tt.href, tt.base, tt.want, got, ok)
		}
	}

	// relative links from an extensionless page and from feed.xml, which
	// build publishes at /docs/foo/ and /docs/feed.xml
	dir := writeSite(t, map[string]string{
		"docs/foo.jsx":      `<p><a href="bar">bar</a> <a href="../bar/#x">again</a></p>`,
		"docs/feed.xml.jsx": `<p><a href="bar">bar</a></p>`,
		"docs/foo/bar.md":   "# foo/bar\n",
		"docs/bar.md":       "# bar\n",
	})
	site := parsed(t, dir, false)
	assertList(t, "foo links", linksOf(t, site, "docs/foo"), "/docs/bar", "/docs/foo/bar")
	assertList(t, "feed.xml links", linksOf(t, site, "docs/feed.xml"), "/docs/bar")
}

func TestLinksAbsolute(t *testing.T) {
	body := `<p><a href="https://example.com/about">abs</a> <a href="//example.com/docs/">proto</a> <a href="mailto:me@example.com">mail</a> <a href="javascript:void(0)">js</a> <a href="https://other.com/about">other</a></p>`
	files := map[string]string{
		"about.md":     "# About\n",
		"docs.md":      "# Docs\n",
		"set.jsx":      `data({siteURL: "https://example.com"}); ` + body,
		"http.jsx":     `data({siteURL: "http://example.com"}); ` + body,
		"unset.jsx":    body,
		"different.md": "---\nsiteURL: https://elsewhere.com\n---\n" + body,
	}
	site := parsed(t, writeSite(t, files), false)
	assertList(t, "siteURL set", linksOf(t, site, "set"), "/about", "/docs")
	assertList(t, "siteURL with another scheme", linksOf(t, site, "http"), "/docs")
	assertList(t, "siteURL unset", linksOf(t, site, "unset"))
	assertList(t, "siteURL different", linksOf(t, site, "different"))

	// siteURL set once in the root _defaults.jsx applies everywhere
	files["_defaults.jsx"] = `siteURL = "https://example.com"`
	site = parsed(t, writeSite(t, files), false)
	assertList(t, "siteURL from defaults", linksOf(t, site, "unset"), "/about", "/docs")
}

func TestLinksSkip(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_outputs/txt.jsx":  "`text`",
		"about.md":          "# About\n",
		"img/logo.svg":      `<svg/>`,
		"_assets/style.css": `body {}`,
		"self.jsx": `<p>
			<a href="/self">self</a> <a href="#top">top</a> <a href="">empty</a>
			<a href="/img/logo.svg">logo</a> <a href="/_assets/style.css">css</a>
			<a href="/nowhere">404</a> <a href="/about.txt">txt</a> <a href="/self.txt">own txt</a>
		</p>`,
	})
	site := parsed(t, dir, false)
	assertList(t, "self links", linksOf(t, site, "self"))
	assertList(t, "self backlinks", backlinksOf(t, site, "self"))
	assertList(t, "about backlinks", backlinksOf(t, site, "about"))
}

func TestLinksNonHTML(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_blog/a.md": "# A\n",
		"posts/a.md": "# A\n",
		"posts/b.md": "# B\n",
		"feed.xml.jsx": `data({contentType: "application/xml"});
			<rss>{pages("/posts").map(p => <item><a href={p.path}>{p.path}</a></item>)}</rss>`,
	})
	site := parsed(t, dir, false)
	assertList(t, "feed links", linksOf(t, site, "feed.xml"))
	assertList(t, "a backlinks", backlinksOf(t, site, "posts/a"))
	assertList(t, "b backlinks", backlinksOf(t, site, "posts/b"))
}

func TestLinksListingPage(t *testing.T) {
	files := map[string]string{
		"index.jsx":  `<ul>{pages("/posts").map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>`,
		"posts/a.md": "---\ntitle: A\n---\n# A\n",
		"posts/b.md": "---\ntitle: B\n---\n# B\n",
		// sorts before posts in the walk, so it is parsed first too
		"aaa/index.jsx": `<ul>{pages("/posts").map(p => <li><a href={p.path}>{p.title}</a></li>)}</ul>`,
	}
	site := parsed(t, writeSite(t, files), false)
	assertList(t, "root links", linksOf(t, site, "."), "/posts/a", "/posts/b")
	assertList(t, "aaa links", linksOf(t, site, "aaa"), "/posts/a", "/posts/b")
	assertList(t, "a backlinks", backlinksOf(t, site, "posts/a"), "/", "/aaa")
}

func TestBacklinks(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"target.md":    "# Target\n",
		"lonely.md":    "# Lonely\n",
		"b.md":         "---\ndate: 2026-01-02\n---\n[t](/target) [again](/target/) [t](target)\n",
		"a.md":         "---\ndate: 2026-01-01\n---\n[t](/target)\n",
		"z.md":         "[t](/target)\n",
		"dir/index.md": "[t](/target) [b](/b)\n",
		"dir/sub.md":   "# sub\n",
	})
	site := parsed(t, dir, false)
	// undated first, then by date, like pages()
	assertList(t, "target backlinks", backlinksOf(t, site, "target"), "/dir", "/z", "/a", "/b")
	assertList(t, "lonely backlinks", backlinksOf(t, site, "lonely"))
	assertList(t, "b links", linksOf(t, site, "b"), "/target")

	for _, b := range site.Page("target").data[Backlinks].([]any) {
		ref := b.(map[string]any)
		for _, key := range []string{Links, Backlinks, "subpages"} {
			if _, ok := ref[key]; ok {
				t.Errorf("backlink %s has %s", ref[Path], key)
			}
		}
		if ref[Body] == nil || ref[Slug] == nil {
			t.Errorf("backlink %s is missing page data: %v", ref[Path], ref)
		}
	}

	// rendered through JSX, the lists work like arrays
	dir = writeSite(t, map[string]string{
		"_layout.jsx": `(content) => <html><body>{content}<p id="n">{page.backlinks.filter(p => p.path.startsWith("/logs/")).length} logs link here</p><ul>{page.backlinks.map(p => <li>{p.path}</li>)}</ul><p id="l">{page.links.join(",")}</p></body></html>`,
		"docs/x.md":   "# X\n",
		"logs/1.md":   "[x](/docs/x)\n",
		"logs/2.md":   "[x](/docs/x)\n",
		"other.md":    "[x](/docs/x) [1](/logs/1)\n",
	})
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "docs/x/index.html"), "2 logs link here", "<li>/logs/1</li>", "<li>/other</li>")
	assertContains(t, readOut(t, dir, "other/index.html"), "/docs/x,/logs/1")
}

func TestBacklinksDrafts(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"target.md":       "# Target\n[d](/draft-target)\n",
		"draft.md":        "---\ndraft: true\n---\n[t](/target)\n",
		"future.md":       "---\ndate: 2999-01-01\n---\n[t](/target)\n",
		"draft-target.md": "---\ndraft: true\n---\n# hidden\n",
		"public.md":       "[t](/target)\n",
	})
	site := parsed(t, dir, false)
	assertList(t, "build: target backlinks", backlinksOf(t, site, "target"), "/public")
	assertList(t, "build: target links", linksOf(t, site, "target"))

	site = parsed(t, dir, true)
	assertList(t, "serve: target backlinks", backlinksOf(t, site, "target"), "/draft", "/public", "/future")
	assertList(t, "serve: target links", linksOf(t, site, "target"), "/draft-target")
}

func TestBacklinksVisible(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx": `(content) => <html><body>{content}<p id="own">{page.backlinks.length} link here</p></body></html>`,
		"index.jsx":   `data({layout: "_home"}); <p>home</p>`,
		"_home.jsx":   `(content) => <html><body>{pages("/docs").map(p => <p>{p.path}: {p.backlinks.length}</p>)}</body></html>`,
		"docs/a.md":   "# A\n",
		"docs/b.md":   "# B\n",
		"x.md":        "[a](/docs/a) [b](/docs/b)\n",
		"y.md":        "[a](/docs/a)\n",
	})
	mustBuild(t, dir)
	assertContains(t, readOut(t, dir, "docs/a/index.html"), "2 link here")
	assertContains(t, readOut(t, dir, "index.html"), "/docs/a: 2", "/docs/b: 1")
}

func TestBacklinksStable(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"target.jsx": `<ul>{page.backlinks.map(p => <li><a href={p.path}>{p.path}</a></li>)}</ul>`,
		"a.md":       "[t](/target)\n",
		"b.md":       "[t](/target)\n",
	})
	site := parsed(t, dir, false)
	render := func() string {
		var buf bytes.Buffer
		if err := site.Page("target").Render(&buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	first := render()
	assertContains(t, first, `<a href="/a">`, `<a href="/b">`)
	if second := render(); second != first {
		t.Errorf("second render differs:\n%s\n%s", first, second)
	}
	assertList(t, "target links", linksOf(t, site, "target"))
	assertList(t, "a backlinks", backlinksOf(t, site, "a"))
	assertList(t, "target backlinks", backlinksOf(t, site, "target"), "/a", "/b")
}

func TestServeBacklinks(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx": `(content) => <html><body>{content}<ul>{page.backlinks.map(p => <li>from {p.path}</li>)}</ul></body></html>`,
		"target.md":   "# Target\n",
		"a.md":        "# A\n",
	})
	site := parsed(t, dir, true)
	if body := serve(t, site, "/target").Body.String(); strings.Contains(body, "from /a") {
		t.Fatalf("backlink before the link exists:\n%s", body)
	}

	// what the watcher does when a page source changes
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n[t](/target)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	captureStdout(t, func() {
		if err := site.ParseAll(); err != nil {
			t.Fatal(err)
		}
	})
	assertContains(t, serve(t, site, "/target").Body.String(), "<li>from /a</li>")
}
