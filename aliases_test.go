package taragen

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func buildOutput(t *testing.T, dir string) (string, error) {
	t.Helper()
	var err error
	out := captureStdout(t, func() { err = build(t, dir) })
	return out, err
}

func assertStub(t *testing.T, got, target string) {
	t.Helper()
	assertContains(t, got,
		`<meta http-equiv="refresh" content="0; url=`+target+`">`,
		`<link rel="canonical" href="`+target+`">`,
		`<meta name="robots" content="noindex">`,
		`<a href="`+target+`">`,
	)
}

func assertBuildError(t *testing.T, dir string, wants ...string) {
	t.Helper()
	err := buildQuiet(t, dir)
	if err == nil {
		t.Fatalf("expected an error containing %q", wants)
	}
	assertContains(t, err.Error(), wants...)
}

func TestAliasStub(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":       `(content) => <html><body class="layout">{content}</body></html>`,
		"articles/ngrok.md": "---\ntitle: Ngrok\naliases: [/old/path]\n---\n# Ngrok\n",
	})
	mustBuild(t, dir)
	got := readOut(t, dir, "old/path/index.html")
	want := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Redirecting to /articles/ngrok</title>
<link rel="canonical" href="/articles/ngrok">
<meta name="robots" content="noindex">
<meta http-equiv="refresh" content="0; url=/articles/ngrok">
</head>
<body><a href="/articles/ngrok">/articles/ngrok</a></body>
</html>
`
	if got != want {
		t.Errorf("stub: expected\n%s\ngot\n%s", want, got)
	}
	// the page itself is unchanged
	assertContains(t, readOut(t, dir, "articles/ngrok/index.html"), `class="layout"`, "<h1>Ngrok</h1>")
}

func TestAliasDefaults(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"articles/_defaults.jsx":    `aliases = [page.path.replace(/^\/articles/, "/blog")]`,
		"articles/index.jsx":        `<p>articles</p>`,
		"articles/2021/ngrok.md":    "# Ngrok\n",
		"articles/2022/own.md":      "---\naliases: /elsewhere\n---\n# Own\n",
		"articles/bare/sub/page.md": "# Deep\n",
	})
	mustBuild(t, dir)
	// articles/index.jsx is page /articles, whose directory is the root, so
	// articles/_defaults.jsx doesn't apply to it (see the aliases Updates)
	if exists(dir, "blog/index.html") {
		t.Error("articles/_defaults.jsx should not apply to the articles index")
	}
	assertStub(t, readOut(t, dir, "blog/2021/ngrok/index.html"), "/articles/2021/ngrok")
	assertStub(t, readOut(t, dir, "blog/bare/sub/page/index.html"), "/articles/bare/sub/page")
	// a page's own aliases replace the default
	assertStub(t, readOut(t, dir, "elsewhere/index.html"), "/articles/2022/own")
	if exists(dir, "blog/2022/own/index.html") {
		t.Error("a page's own aliases should replace the defaults")
	}
	// bare directories (2021, bare, bare/sub) get no stubs
	for _, name := range []string{"blog/2021/index.html", "blog/bare/index.html", "blog/bare/sub/index.html"} {
		if exists(dir, name) {
			t.Errorf("bare directory got a stub at %s", name)
		}
	}
}

func TestAliasNormalize(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"a.md": "---\naliases: /one/\n---\n# A\n",
		"b.jsx": `data({aliases: ["/two/", "/two", "/three"]});
<ul>{page.aliases.map((a) => <li>{a}</li>)}</ul>`,
	})
	site := parsed(t, dir, false)
	if got, want := site.Page("a").data[Aliases], []any{"/one"}; !reflect.DeepEqual(got, want) {
		t.Errorf("a: page.aliases: expected %v, got %v", want, got)
	}
	if got, want := site.Page("b").Aliases(), []string{"/two", "/three"}; !reflect.DeepEqual(got, want) {
		t.Errorf("b: aliases: expected %v, got %v", want, got)
	}
	mustBuild(t, dir)
	assertStub(t, readOut(t, dir, "one/index.html"), "/a")
	assertStub(t, readOut(t, dir, "two/index.html"), "/b")
	assertContains(t, readOut(t, dir, "b/index.html"), "<li>/two</li>", "<li>/three</li>")

	for name, alias := range map[string]string{
		"dotdot":    `["/a/../../etc"]`,
		"dot":       `["/a/./b"]`,
		"empty":     `["/a//b"]`,
		"query":     `["/a?b=c"]`,
		"fragment":  `["/a#b"]`,
		"backslash": `["/a\\b"]`,
		"relative":  `["a/b"]`,
		"number":    `[1]`,
		"object":    `{a: "/b"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeSite(t, map[string]string{"p.jsx": `data({aliases: ` + alias + `}); <p/>`})
			assertBuildError(t, dir, "page /p")
		})
	}
}

func TestAliasEscaping(t *testing.T) {
	dir := writeSite(t, map[string]string{
		`a&b"c.md`: "---\naliases: [/old]\n---\n# A\n",
	})
	mustBuild(t, dir)
	got := readOut(t, dir, "old/index.html")
	assertStub(t, got, "/a&amp;b&#34;c")
	assertContains(t, got, "<title>Redirecting to /a&amp;b\"c</title>", `/a&amp;b"c</a>`)
}

func TestAliasExtension(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"about.md":     "---\naliases: [/old.html]\n---\n# About\n",
		"feed.xml.jsx": `data({layout: "", contentType: "application/xml", preamble: "<?xml?>", aliases: ["/rss.xml"]}); <rss></rss>`,
	})
	mustBuild(t, dir)
	assertStub(t, readOut(t, dir, "old.html"), "/about")
	if got, want := readOut(t, dir, "rss.xml"), readOut(t, dir, "feed.xml"); got != want {
		t.Errorf("rss.xml: expected a copy of feed.xml %q, got %q", want, got)
	}
}

func TestAliasDrafts(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"post.md": "---\ndraft: true\naliases: [/old-post]\n---\n# Post\n",
	})
	mustBuild(t, dir)
	if exists(dir, "old-post/index.html") {
		t.Error("a draft's alias should be skipped in build")
	}
	rec := serve(t, parsed(t, dir, true), "/old-post")
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/post" {
		t.Errorf("serve: status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAliasCollision(t *testing.T) {
	for name, tc := range map[string]struct {
		files map[string]string
		wants []string
	}{
		"own path": {
			map[string]string{"a.md": "---\naliases: [/a]\n---\n"},
			[]string{"page /a (a.md)", "alias /a of page /a"},
		},
		"another page": {
			map[string]string{"a.md": "---\naliases: [/b]\n---\n", "b.md": "# B\n"},
			[]string{"page /b (b.md)", "alias /b of page /a (a.md)"},
		},
		"another alias": {
			map[string]string{"a.md": "---\naliases: [/old]\n---\n", "b.md": "---\naliases: [/old]\n---\n"},
			[]string{"alias /old of page /a (a.md)", "alias /old of page /b (b.md)"},
		},
		"static file": {
			map[string]string{"a.md": "---\naliases: [/logo.svg]\n---\n", "logo.svg": "<svg/>"},
			[]string{"static file logo.svg", "alias /logo.svg of page /a"},
		},
		"output": {
			map[string]string{"_outputs/txt.jsx": txtTemplate, "a.md": "# A\n", "b.md": "---\naliases: [/a.txt]\n---\n"},
			[]string{"output txt of page /a", "alias /a.txt of page /b"},
		},
		"redirects rule": {
			map[string]string{"a.md": "---\naliases: [/old]\n---\n", "_redirects": "/old /elsewhere\n"},
			[]string{"alias /old of page /a", "redirect /old (_redirects line 1)"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertBuildError(t, writeSite(t, tc.files), append([]string{"route collision"}, tc.wants...)...)
		})
	}
}

func TestRedirectsFile(t *testing.T) {
	redirects := `# from            to                               status
/newsletter       https://buttondown.com/progrium  302

/talks            /articles?tag=talks
/old-feed         /feed.xml
/old.html         /about
/data.json        /about
`
	dir := writeSite(t, map[string]string{
		"_redirects":   redirects,
		"about.md":     "# About\n",
		"feed.xml.jsx": `data({layout: "", contentType: "application/xml"}); <rss></rss>`,
	})
	out, err := buildOutput(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := readOut(t, dir, "_redirects"); got != redirects {
		t.Errorf("_redirects: expected a verbatim copy, got %q", got)
	}
	assertStub(t, readOut(t, dir, "newsletter/index.html"), "https://buttondown.com/progrium")
	assertStub(t, readOut(t, dir, "talks/index.html"), "/articles?tag=talks")
	assertStub(t, readOut(t, dir, "old.html"), "/about")
	if got, want := readOut(t, dir, "old-feed/index.html"), readOut(t, dir, "feed.xml"); got != want {
		t.Errorf("old-feed: expected a copy of feed.xml %q, got %q", want, got)
	}
	if exists(dir, "data.json") {
		t.Error("an HTML stub should not be written at /data.json")
	}
	assertContains(t, out, "warning: _redirects line 7: /data.json has an extension")

	for name, tc := range map[string]struct{ line, want string }{
		"rewrite":      {"/a /b 200", `unsupported status "200"`},
		"not found":    {"/a /b 404", `unsupported status "404"`},
		"force":        {"/a /b 301!", `unsupported status "301!"`},
		"placeholder":  {"/news/:year /blog/:year", "placeholders"},
		"to named":     {"/news/* /blog/:year", "placeholders other than :splat"},
		"condition":    {"/store id=:id /blog/:id 301", "extra fields"},
		"extra fields": {"/a /b 301 Country=us", "extra fields"},
		"one field":    {"/a", "needs a from path and a target"},
		"star inside":  {"/a/*/b /c", "* may only end the path"},
		"relative to":  {"/a b", "must be a path starting with /"},
		"proto rel":    {"/a //evil.com", "must be a path starting with /"},
		"bad from":     {"/a/../b /c", "must not contain a .. segment"},
		"splat no *":   {"/a /b/:splat", ":splat needs a from ending in /*"},
		"no host":      {"/a https:///b", "missing host"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeSite(t, map[string]string{"_redirects": "# comment\n\n" + tc.line + "\n"})
			assertBuildError(t, dir, "_redirects line 3", tc.want)
		})
	}
}

func TestRedirectsWildcard(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_redirects": `/blog/*      /articles/:splat
/blog/*      /other/:splat
/ext/*       https://example.com/:splat
/nosplat/*   /articles
`,
		"articles/index.jsx":     `<p>articles</p>`,
		"articles/2021/ngrok.md": "# Ngrok\n",
		"articles/2021/kept.md":  "# Kept\n",
		"articles/draft.md":      "---\ndraft: true\n---\n# Draft\n",
		"articles/feed.xml.jsx":  `data({layout: "", contentType: "application/xml"}); <rss/>`,
		"other/x.md":             "# X\n",
		"blog/2021/kept.md":      "# Real page under the rule\n",
	})
	out, err := buildOutput(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	assertStub(t, readOut(t, dir, "blog/2021/ngrok/index.html"), "/articles/2021/ngrok")
	// the second rule expands only where the first left room
	assertStub(t, readOut(t, dir, "blog/x/index.html"), "/other/x")
	// a path already in the route table is skipped
	assertContains(t, readOut(t, dir, "blog/2021/kept/index.html"), "Real page under the rule")
	for _, name := range []string{"blog/index.html", "blog/draft/index.html", "blog/feed.xml", "ext/index.html", "nosplat/index.html"} {
		if exists(dir, name) {
			t.Errorf("unexpected stub %s", name)
		}
	}
	assertContains(t, out,
		"warning: _redirects line 3: /ext/* can't be expanded",
		"warning: _redirects line 4: /nosplat/* can't be expanded",
	)
}

func TestServeRedirects(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_redirects": `/newsletter  https://buttondown.com/progrium  302
/talks       /articles?tag=talks
/temp        /about  307
/blog/*      /articles/:splat
/ext/*       https://example.com/x/:splat  308
`,
		"about.md":               "---\naliases: [/old-about]\n---\n# About\n",
		"articles/2021/ngrok.md": "# Ngrok\n",
		"feed.xml.jsx":           `data({layout: "", contentType: "application/xml", aliases: ["/rss.xml"]}); <rss></rss>`,
	})
	site := parsed(t, dir, true)
	for path, want := range map[string]struct {
		code     int
		location string
	}{
		"/old-about":         {301, "/about"},
		"/old-about/":        {301, "/about"},
		"/newsletter":        {302, "https://buttondown.com/progrium"},
		"/talks?x=1":         {301, "/articles?tag=talks"},
		"/temp?x=1":          {307, "/about?x=1"},
		"/blog/2021/ngrok":   {301, "/articles/2021/ngrok"},
		"/blog/2099/unbuilt": {301, "/articles/2099/unbuilt"},
		"/blog/a%3Fb?q=1":    {301, "/articles/a%3Fb?q=1"},
		"/ext/a/b":           {308, "https://example.com/x/a/b"},
	} {
		rec := serve(t, site, path)
		if rec.Code != want.code || rec.Header().Get("Location") != want.location {
			t.Errorf("%s: expected %d %q, got %d %q", path, want.code, want.location, rec.Code, rec.Header().Get("Location"))
		}
	}

	rec := serve(t, site, "/rss.xml")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/xml" || rec.Body.String() != "<rss></rss>" && !strings.Contains(rec.Body.String(), "<rss></rss>") {
		t.Errorf("/rss.xml: status %d, type %q: %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}

	// _redirects is an extensionless root file, so editing it reparses
	if reparse, reload := classifyChange(RedirectsFile, false); !reparse || !reload {
		t.Errorf("classifyChange(_redirects) = %v, %v", reparse, reload)
	}
	if err := os.WriteFile(filepath.Join(dir, RedirectsFile), []byte("/newsletter /about 301\n"), 0644); err != nil {
		t.Fatal(err)
	}
	captureStdout(t, func() {
		if err := site.ParseAll(); err != nil {
			t.Fatal(err)
		}
	})
	if rec := serve(t, site, "/newsletter"); rec.Code != 301 || rec.Header().Get("Location") != "/about" {
		t.Errorf("after edit: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := serve(t, site, "/blog/2021/ngrok"); rec.Code != 404 {
		t.Errorf("removed wildcard rule: expected 404, got %d", rec.Code)
	}
}
