package taragen

import (
	"bytes"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"tractor.dev/toolkit-go/engine/fs/watchfs"
)

var reloaders sync.Map

func watchForReloads(dir string, s *Site) {
	wfs := watchfs.New(os.DirFS("/").(fs.StatFS))
	w, err := wfs.Watch(strings.TrimPrefix(dir, "/"), &watchfs.Config{
		Recursive: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	root := strings.Trim(dir, "/")
	for event := range w.Iter() {
		rel := strings.TrimPrefix(strings.TrimPrefix(event.Path, root), "/")
		reparse, reload := classifyChange(rel, event.IsDir())
		if !reload {
			continue
		}
		if reparse {
			if err := s.ParseAll(); err != nil {
				log.Println("parse:", err)
			}
		}
		reloaders.Range(func(key, value any) bool {
			if conn, ok := key.(*websocket.Conn); ok {
				conn.Close()
			}
			return true
		})
	}
}

// classifyChange decides what a change to rel (a path relative to the site
// root) needs. Any file in the site reloads the browser, so CSS and other
// static edits show up. Page sources and directories also re-parse the site
// to pick up new and removed pages. Dot-prefixed paths and the build output
// in _public are ignored.
func classifyChange(rel string, isDir bool) (reparse, reload bool) {
	if rel == "" {
		return true, true
	}
	parts := strings.Split(rel, "/")
	if parts[0] == "_public" {
		return false, false
	}
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return false, false
		}
	}
	// a deleted directory can't be stat'd, so treat extensionless names as
	// possible directories
	if isDir || path.Ext(parts[len(parts)-1]) == "" {
		return true, true
	}
	for ext := range Formats {
		if strings.HasSuffix(rel, ext) {
			return true, true
		}
	}
	return false, true
}

func injectLiveReload(w io.Writer, buf bytes.Buffer) {
	htmlContent := buf.String()
	headEndIndex := strings.Index(strings.ToLower(htmlContent), "</body>")
	if headEndIndex == -1 {
		w.Write([]byte(htmlContent))
		return
	}
	before := htmlContent[:headEndIndex]
	after := htmlContent[headEndIndex:]
	w.Write([]byte(before))
	w.Write([]byte(`<script>(new WebSocket("/.live-reload")).onclose = () => window.location.reload();</script>`))
	w.Write([]byte(after))
}

func handleLiveReload(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Adjust this to match your needs
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	reloaders.Store(conn, true)
	defer func() {
		reloaders.Delete(conn)
		conn.Close()
	}()

	// Keep the connection open
	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}
}
