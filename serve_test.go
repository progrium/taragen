package taragen

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func serve(t *testing.T, site *Site, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	site.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestServeHTTP(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"_layout.jsx":  `(content) => <html><body>{content}</body></html>`,
		"index.jsx":    `<p>home</p>`,
		"docs/a.md":    "# A\n",
		"feed.xml.jsx": `data({layout: "", contentType: "application/xml", preamble: "<?xml?>"}); <rss></rss>`,
		"broken.jsx":   `data({layout: "_layouts/nope"}); <p/>`,
		"img/logo.svg": `<svg/>`,
	})
	site := NewSite(dir, false)
	captureStdout(t, func() {
		if err := site.ParseAll(); err != nil {
			t.Fatal(err)
		}
	})

	rec := serve(t, site, "/")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "text/html" {
		t.Errorf("/: status %d, type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	assertContains(t, rec.Body.String(), "<p>home</p>", `new WebSocket("/.live-reload")`)
	if body := rec.Body.String(); strings.Index(body, "WebSocket") > strings.Index(body, "</body>") {
		t.Errorf("live reload script should be inside the body:\n%s", body)
	}

	assertContains(t, serve(t, site, "/docs/a/").Body.String(), "<h1>A</h1>")

	rec = serve(t, site, "/feed.xml")
	if rec.Header().Get("Content-Type") != "application/xml" {
		t.Errorf("feed.xml: type %q", rec.Header().Get("Content-Type"))
	}
	if got := rec.Body.String(); got != "<?xml?>\n<rss></rss>" {
		t.Errorf("feed.xml: got %q", got)
	}

	rec = serve(t, site, "/broken")
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "_layouts/nope") {
		t.Errorf("broken: status %d: %s", rec.Code, rec.Body.String())
	}

	rec = serve(t, site, "/img/logo.svg")
	if rec.Code != 200 || rec.Body.String() != "<svg/>" {
		t.Errorf("static file: status %d: %q", rec.Code, rec.Body.String())
	}

	if rec := serve(t, site, "/missing"); rec.Code != http.StatusNotFound {
		t.Errorf("missing: status %d", rec.Code)
	}
}

func TestInjectLiveReloadWithoutBody(t *testing.T) {
	var out bytes.Buffer
	in := bytes.NewBufferString("<p>fragment</p>")
	injectLiveReload(&out, *in)
	if out.String() != "<p>fragment</p>" {
		t.Errorf("got %q", out.String())
	}
}

func dialLiveReload(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/.live-reload"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestLiveReloadUpgrade(t *testing.T) {
	site := NewSite(writeSite(t, nil), false)
	srv := httptest.NewServer(site)
	defer srv.Close()

	conn := dialLiveReload(t, srv)
	conn.Close()

	// a plain request to the endpoint is not upgraded
	if rec := serve(t, site, "/.live-reload"); rec.Code != http.StatusBadRequest {
		t.Errorf("status %d", rec.Code)
	}
}

// waitForClose waits for the server to close conn, which tells the browser to
// reload.
func waitForClose(t *testing.T, conn *websocket.Conn, touch func()) {
	t.Helper()
	closed := make(chan struct{})
	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				close(closed)
				return
			}
		}
	}()
	deadline := time.After(10 * time.Second)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-closed:
			return
		case <-tick.C:
			// keep changing the file in case the watcher wasn't ready yet
			touch()
		case <-deadline:
			t.Fatal("live reload connection was not closed")
		}
	}
}

func TestWatchReloadsOnChange(t *testing.T) {
	dir := writeSite(t, map[string]string{
		"index.jsx":         `<p>one</p>`,
		"_assets/style.css": `body {}`,
	})
	site := NewSite(dir, false)
	captureStdout(t, func() {
		if err := site.ParseAll(); err != nil {
			t.Fatal(err)
		}
	})
	site.WatchForReloads()
	srv := httptest.NewServer(site)
	defer srv.Close()

	// a page change re-parses, so a new page is served afterwards
	conn := dialLiveReload(t, srv)
	waitForClose(t, conn, func() {
		os.WriteFile(filepath.Join(dir, "new.jsx"), []byte(`<p>new</p>`), 0644)
	})
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(serve(t, site, "/new").Body.String(), "<p>new</p>") {
		if time.Now().After(deadline) {
			t.Fatal("new page was not picked up")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// a static change only reloads
	conn = dialLiveReload(t, srv)
	n := 0
	waitForClose(t, conn, func() {
		n++
		os.WriteFile(filepath.Join(dir, "_assets/style.css"), []byte(strings.Repeat("x", n)), 0644)
	})
}
