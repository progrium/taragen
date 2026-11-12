package taragen

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type Site struct {
	dir    string
	pages  map[string]*Page
	drafts bool

	mu sync.Mutex
}

func NewSite(dir string, drafts bool) *Site {
	return &Site{
		dir:    dir,
		pages:  make(map[string]*Page),
		drafts: drafts,
	}
}

func (s *Site) WatchForReloads() {
	go watchForReloads(s.dir, s)
}

func (s *Site) GenerateAll(dest string, clobber bool) (err error) {
	if err = s.ParseAll(); err != nil {
		return
	}

	if dest == "" {
		return fmt.Errorf("dest is required")
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}

	if clobber {
		if err := cleanDir(dest); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	copyFile := func(filename string) error {
		b, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		targetPath := path.Join(dest, strings.TrimPrefix(filename, s.dir))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, b, 0644); err != nil {
			return err
		}
		fmt.Println("copied:", strings.TrimPrefix(filename, s.dir))
		return nil
	}

	// _assets is pruned from the page tree like every underscore directory,
	// but pages link to /_assets/..., so it is copied verbatim.
	if err := s.walkAssets(copyFile); err != nil {
		return err
	}

	// a directory and its index file normalize to the same page path
	rendered := make(map[string]bool)

	return s.walk(func(filename string, pagepath string) error {
		if pagepath == "" {
			return copyFile(filename)
		} else {
			if rendered[pagepath] {
				return nil
			}
			rendered[pagepath] = true
			p := s.Page(pagepath)
			if len(p.Source()) == 0 {
				fmt.Println("skip dir:", pagepath)
				return nil
			}
			if p.Draft() {
				fmt.Println("skip draft:", pagepath)
				return nil
			}
			var targetPath string
			if filepath.Ext(p.path) == "" || p.path == "." {
				targetPath = path.Join(dest, p.path, "index.html")
			} else {
				targetPath = path.Join(dest, p.path)
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			f, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if err := p.Render(f); err != nil {
				f.Close()
				return err
			}
			f.Close()
			fmt.Println("rendered:", p.path)
		}
		return nil
	})
}

// AssetsDir is the root-level directory copied verbatim by build.
const AssetsDir = "_assets"

// walkAssets calls fn for every file under the site's _assets directory,
// skipping dot-prefixed names.
func (s *Site) walkAssets(fn func(filename string) error) error {
	root := filepath.Join(s.dir, AssetsDir)
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return nil
	}
	return filepath.Walk(root, func(curPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		return fn(curPath)
	})
}

// cleanDir removes everything in dir except dot-prefixed entries at its top
// level, so a checked-out deploy repo (_public/.git) survives a rebuild.
func cleanDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (s *Site) Page(normalPath string) *Page {
	normalPath = strings.TrimPrefix(normalPath, "/")
	if _, ok := s.pages[normalPath]; !ok {
		s.pages[normalPath] = &Page{
			path: normalPath,
			site: s,
			data: Data{
				Path: path.Clean("/" + normalPath),
				Slug: filepath.Base(normalPath),
			},
		}
	}
	return s.pages[normalPath]
}

func (s *Site) IsPage(path string) bool {
	normalPath := strings.TrimPrefix(path, "/")
	_, ok := s.pages[normalPath]
	return ok
}

func (s *Site) Partial(name string, globals map[string]any, args ...any) ([]byte, error) {
	partialPath := path.Join(s.dir, name)
	isJSX := true
	partialSrc, err := os.ReadFile(partialPath + ExtJSX)
	if err != nil {
		partialSrc, err = os.ReadFile(partialPath + ExtTemplate)
		if err != nil {
			return nil, err
		}
		isJSX = false
	}
	if isJSX {
		return RenderJSX(partialPath, partialSrc, globals, args...)
	}
	return RenderTemplate(name, partialSrc, nil)
}

func (s *Site) walk(fn func(filename string, pagepath string) error) error {
	return filepath.Walk(s.dir, func(curPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			if info.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), "_") {
			if info.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		isPage := false
		for key := range Formats {
			if strings.HasSuffix(info.Name(), key) {
				isPage = true
			}
		}
		if !isPage && !info.IsDir() {
			return fn(curPath, "")
		}

		return fn(curPath, s.normalizePath(curPath))
	})
}

func (s *Site) ParseAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pages = make(map[string]*Page)
	parsed := make(map[string]bool)

	return s.walk(func(filename string, pagepath string) error {
		if pagepath == "" || parsed[pagepath] {
			return nil
		}
		parsed[pagepath] = true

		if err := s.Page(pagepath).Parse(); err != nil {
			return err
		}

		return nil
	})
}

func (s *Site) normalizePath(p string) string {
	normalized := strings.TrimPrefix(p, s.dir)
	normalized = strings.TrimSuffix(normalized, filepath.Ext(normalized))
	normalized = strings.TrimSuffix(normalized, "/index")
	if normalized == "" {
		normalized = "."
	}
	return normalized
}

func (s *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/.live-reload" {
		handleLiveReload(w, r)
		return
	}

	normalPath := strings.Trim(r.URL.Path, "/")
	if normalPath == "" {
		normalPath = "."
	}

	// Rendering reads the page map and mutates page data, and the watcher
	// replaces the map in ParseAll, so both run under the site lock.
	var buf bytes.Buffer
	var page *Page
	var contentType string
	var err error
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.IsPage(normalPath) {
			page = s.Page(normalPath)
			err = page.Render(&buf)
			contentType = page.ContentType()
		}
	}()

	if page != nil {
		if err != nil {
			log.Println(normalPath, ":", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		injectLiveReload(w, buf)
		return
	}

	http.ServeFile(w, r, path.Join(s.dir, normalPath))
}
