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
	dir   string
	pages map[string]*Page

	mu sync.Mutex
}

func NewSite(dir string) *Site {
	return &Site{
		dir:   dir,
		pages: make(map[string]*Page),
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
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	return s.walk(func(filename string, pagepath string) error {
		if pagepath == "" {
			// static file
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
		} else {
			// page
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

func (s *Site) Pages(name string) (pages []*Page) {
	if !strings.Contains(name, "*") {
		if s.IsPage(name) {
			return s.Page(name).Subpages()
		}
		return nil
	}
	for pathname, page := range s.pages {
		m, err := path.Match(name, pathname)
		if err != nil {
			continue // TODO: log error
		}
		if m {
			pages = append(pages, page.Subpages()...)
		}
	}
	sortPages(pages)
	return pages
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
		return RenderJSX(partialSrc, globals, args...)
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
		if info.IsDir() {
			return nil
		}

		isPage := false
		for key := range Formats {
			if strings.HasSuffix(info.Name(), key) {
				isPage = true
			}
		}
		if !isPage {
			return fn(curPath, "")
		}

		return fn(curPath, s.normalizePath(curPath))
	})
}

func (s *Site) ParseAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pages = make(map[string]*Page)

	return s.walk(func(filename string, pagepath string) error {
		if pagepath == "" {
			return nil
		}

		log.Println("parse:", pagepath)
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
	if s.IsPage(normalPath) {
		var buf bytes.Buffer
		page := s.Page(normalPath)
		err := page.Render(&buf)
		if err != nil {
			log.Println(normalPath, ":", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", page.ContentType())
		injectLiveReload(w, buf)
		return
	}

	http.ServeFile(w, r, path.Join(s.dir, normalPath))
}
