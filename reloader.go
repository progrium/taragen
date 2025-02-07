package taragen

import (
	"bytes"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"tractor.dev/toolkit-go/engine/fs/watchfs"
)

var reloaders sync.Map

func WatchForReloads(dir string) {
	wfs := watchfs.New(os.DirFS("/").(fs.StatFS))
	w, err := wfs.Watch(strings.TrimPrefix(dir, "/"), &watchfs.Config{
		Recursive: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	for event := range w.Iter() {
		if strings.HasSuffix(event.Path, ExtMarkdown) ||
			strings.HasSuffix(event.Path, ExtJSX) ||
			strings.HasSuffix(event.Path, ExtTemplate) {

			reloaders.Range(func(key, value any) bool {
				if conn, ok := key.(*websocket.Conn); ok {
					conn.Close()
				}
				return true
			})
		}
	}
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
