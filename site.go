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
)

type Site struct {
	dir   string
	pages map[string]*Page
}

func NewSite(dir string) *Site {
	return &Site{
		dir:   dir,
		pages: make(map[string]*Page),
	}
}

func (s *Site) WatchForReloads() {
	go watchForReloads(s.dir)
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

	var pages []*Page
	for _, page := range s.pages {
		pages = append(pages, page)
	}
	sortPages(pages)

	for _, page := range pages {
		if page.IsDir() {
			continue
		}
		fmt.Println(page.path)
		targetPath := path.Join(dest, page.path, "index.html")
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		f, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		if err := page.Render(f); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}

func (s *Site) Pages(name string) []*Page {
	return s.Page(name).Subpages()
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
	p, ok := s.pages[normalPath]
	if ok && !p.IsDir() {
		return true
	}
	return false
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

func (s *Site) ParseAll() error {
	return filepath.Walk(s.dir, func(curPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), ".") {
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
		if !info.IsDir() {
			found := false
			for key := range Formats {
				if strings.HasSuffix(info.Name(), key) {
					found = true
				}
			}
			if !found {
				return nil
			}
		}

		// normalize path
		normalizedPath := strings.TrimPrefix(curPath, s.dir)
		normalizedPath = strings.TrimSuffix(normalizedPath, filepath.Ext(normalizedPath))
		normalizedPath = strings.TrimSuffix(normalizedPath, "/index")
		if normalizedPath == "" {
			normalizedPath = "."
		}

		if err := s.Page(normalizedPath).Parse(); err != nil {
			return err
		}

		return nil
	})
}

func (s *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/.live-reload" {
		handleLiveReload(w, r)
		return
	}

	normalPath := strings.TrimPrefix(r.URL.Path, "/")
	if normalPath == "" {
		normalPath = "."
	}
	if s.IsPage(normalPath) {
		w.Header().Set("Content-Type", "text/html")
		var buf bytes.Buffer
		err := s.Page(normalPath).Render(&buf)
		if err != nil {
			log.Println(normalPath, ":", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		injectLiveReload(w, buf)
		return
	}

	http.ServeFile(w, r, path.Join(s.dir, normalPath))
}
