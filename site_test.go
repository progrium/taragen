package taragen

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func writeSite(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func readOut(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "_public", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func build(t *testing.T, dir string) error {
	t.Helper()
	return NewSite(dir, false).GenerateAll(filepath.Join(dir, "_public"), true)
}

func TestBuildCopiesAssets(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx":              `<p>home</p>`,
		"_assets/style.css":      `body {}`,
		"_assets/img/logo.svg":   `<svg/>`,
		"_assets/raw.jsx":        `not rendered`,
		"_assets/.DS_Store":      `junk`,
		"_partials/header.jsx":   `() => <header/>`,
		"_layout.jsx":            `(content) => <html><body>{content}</body></html>`,
		"docs/_assets/local.css": `nested assets are not copied`,
	})
	if err := build(t, dir); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"_assets/style.css":    `body {}`,
		"_assets/img/logo.svg": `<svg/>`,
		"_assets/raw.jsx":      `not rendered`,
	} {
		if got := readOut(t, dir, name); got != want {
			t.Errorf("%s: expected %q, got %q", name, want, got)
		}
	}
	for _, name := range []string{"_assets/.DS_Store", "_partials", "_layout.jsx", "docs/_assets"} {
		if _, err := os.Stat(filepath.Join(dir, "_public", name)); err == nil {
			t.Errorf("%s should not be in the output", name)
		}
	}
}

func TestBuildCleansStaleOutput(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx":              `<p>home</p>`,
		"_public/old/index.html": `stale`,
		"_public/.git/HEAD":      `ref: refs/heads/gh-pages`,
	})
	if err := build(t, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_public/old")); err == nil {
		t.Error("stale output was not removed")
	}
	if got := readOut(t, dir, ".git/HEAD"); got != `ref: refs/heads/gh-pages` {
		t.Errorf("dot entries should be kept, got %q", got)
	}
	if !strings.Contains(readOut(t, dir, "index.html"), "<p>home</p>") {
		t.Error("index.html not rendered")
	}
}

func TestBuildFailsOnPartialError(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"missing": {
			"index.jsx": `<div>{partial("_partials/nope")}</div>`,
		},
		"throws": {
			"index.jsx":         `<div>{partial("_partials/bad")}</div>`,
			"_partials/bad.jsx": `() => <p>{undefinedThing.x}</p>`,
		},
		"in layout": {
			"index.jsx":         `<p>home</p>`,
			"_layout.jsx":       `(content) => <html><body>{partial("_partials/bad")}{content}</body></html>`,
			"_partials/bad.jsx": `() => <p>{{a: 1}}</p>`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeSite(t, files)
			err := build(t, dir)
			if err == nil {
				t.Fatal("expected the build to fail")
			}
			if !strings.Contains(err.Error(), "partial _partials/") {
				t.Errorf("error should name the partial: %v", err)
			}
		})
	}
}

func TestPartialsAndContentAreNotEscaped(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":       `(content) => <html><head><title>{page.title}</title></head><body>{partial("_partials/nav", page.title)}{content}</body></html>`,
		"_partials/nav.jsx": `(title) => <nav title={title}><a href="/">Home</a></nav>`,
		"post.md":           "---\ntitle: Tips & \"Tricks\"\n---\nA *post* & more.\n",
		"index.jsx":         `<ul>{pages(".").filter(p => p.title).map(p => <li>{raw(p.body)}</li>)}</ul>`,
	})
	if err := build(t, dir); err != nil {
		t.Fatal(err)
	}
	post := readOut(t, dir, "post/index.html")
	for _, want := range []string{
		`<title>Tips &amp; "Tricks"</title>`,
		`<nav title="Tips &amp; &#34;Tricks&#34;"><a href="/">Home</a></nav>`,
		`<em>post</em> &amp; more.`,
	} {
		if !strings.Contains(post, want) {
			t.Errorf("post: missing %q in:\n%s", want, post)
		}
	}
	if index := readOut(t, dir, "index.html"); !strings.Contains(index, `<p>A <em>post</em> &amp; more.</p>`) {
		t.Errorf("index: raw body not inserted:\n%s", index)
	}
}

func TestBuildRendersDirIndexOnce(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"docs/index.jsx": `<p>docs</p>`,
		"docs/a.md":      "# A\n",
	})
	site := NewSite(dir, false)
	out := captureStdout(t, func() {
		if err := site.GenerateAll(filepath.Join(dir, "_public"), true); err != nil {
			t.Fatal(err)
		}
	})
	if n := strings.Count(out, "rendered: docs\n"); n != 1 {
		t.Errorf("docs rendered %d times:\n%s", n, out)
	}
}

func TestOutputIsDeterministic(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx": `<div a="1" b="2" c="3" d="4" e="5" f="6" g="7" h="8" style={{a: "1", b: "2", c: "3"}}>x</div>`,
	})
	var first string
	for i := 0; i < 20; i++ {
		if err := build(t, dir); err != nil {
			t.Fatal(err)
		}
		out := readOut(t, dir, "index.html")
		if i == 0 {
			first = out
			if !strings.Contains(out, `a="1" b="2" c="3" d="4" e="5" f="6" g="7" h="8" style="a:1;b:2;c:3;"`) {
				t.Fatalf("attributes out of source order:\n%s", out)
			}
		} else if out != first {
			t.Fatalf("build %d differs:\n%s\nvs\n%s", i, out, first)
		}
	}
}

// Run with -race: requests render while the watcher re-parses the site.
func TestServeWhileReparsing(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx":   `<ul>{pages("docs").map(p => <li>{p.title}</li>)}</ul>`,
		"docs/a.md":   "---\ntitle: A\n---\na\n",
		"docs/b.md":   "---\ntitle: B\n---\nb\n",
		"_layout.jsx": `(content) => <html><body>{content}</body></html>`,
	})
	site := NewSite(dir, true)
	if err := site.ParseAll(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				site.ParseAll()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				rec := httptest.NewRecorder()
				site.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
				if rec.Code != 200 {
					t.Errorf("status %d: %s", rec.Code, rec.Body.String())
				}
			}
		}()
	}
	wg.Wait()
}

func TestClassifyChange(t *testing.T) {
	tests := []struct {
		path            string
		isDir           bool
		reparse, reload bool
	}{
		{"index.jsx", false, true, true},
		{"docs/a.md", false, true, true},
		{"_layout.tmpl", false, true, true},
		{"_assets/style.css", false, false, true},
		{"logo.png", false, false, true},
		{"docs", true, true, true},
		{"docs/removed", false, true, true},
		{"_public/index.html", false, false, false},
		{".git/index", false, false, false},
		{"docs/.a.md.swp", false, false, false},
	}
	for _, tt := range tests {
		reparse, reload := classifyChange(tt.path, tt.isDir)
		if reparse != tt.reparse || reload != tt.reload {
			t.Errorf("%s: expected reparse=%v reload=%v, got %v %v", tt.path, tt.reparse, tt.reload, reparse, reload)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		buf.ReadFrom(r)
		done <- buf.String()
	}()
	fn()
	w.Close()
	os.Stdout = orig
	return <-done
}
