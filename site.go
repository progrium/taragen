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
	routes *RouteTable
	drafts bool

	mu sync.Mutex

	fontsMu sync.Mutex
	fonts   siteFonts
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

	for _, r := range s.routes.Routes() {
		if err := s.generate(dest, r); err != nil {
			return err
		}
	}
	return nil
}

// generate writes one route to dest: it copies a static file, or renders a
// page or output, skipping drafts.
func (s *Site) generate(dest string, r *Route) error {
	if r.Draft() {
		fmt.Println("skip draft:", r.URL)
		return nil
	}
	targetPath := filepath.Join(dest, filepath.FromSlash(r.File))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	if r.Kind == RouteStatic {
		b, err := os.ReadFile(r.Source)
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, b, 0644); err != nil {
			return err
		}
		fmt.Println("copied:", r.URL)
		return nil
	}

	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer f.Close()
	switch r.Kind {
	case RoutePage:
		err = r.Page.Render(f)
	case RouteOutput:
		_, err = r.Output.Render(f)
	default:
		err = fmt.Errorf("unknown route kind: %s", r.Kind)
	}
	if err != nil {
		return err
	}
	if r.Kind == RoutePage {
		fmt.Println("rendered:", r.Page.path)
	} else {
		fmt.Println("rendered:", r.File)
	}
	return f.Close()
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

// ParseAll resets the page map, parses every page found by walk, and builds
// the route table.
func (s *Site) ParseAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pages = make(map[string]*Page)
	s.routes = nil

	if err := s.refreshFonts(); err != nil {
		return err
	}

	var pagepaths, statics []string
	parsed := make(map[string]bool)

	// _assets is pruned from the page tree like every underscore directory,
	// but pages link to /_assets/..., so its files are static routes.
	if err := s.walkAssets(func(filename string) error {
		statics = append(statics, filename)
		return nil
	}); err != nil {
		return err
	}

	if err := s.walk(func(filename string, pagepath string) error {
		if pagepath == "" {
			statics = append(statics, filename)
			return nil
		}
		// a directory and its index file normalize to the same page path
		if parsed[pagepath] {
			return nil
		}
		parsed[pagepath] = true
		pagepaths = append(pagepaths, pagepath)
		return s.Page(pagepath).Parse()
	}); err != nil {
		return err
	}

	routes, err := s.buildRoutes(pagepaths, statics)
	if err != nil {
		return err
	}
	s.routes = routes
	return nil
}

// Routes returns the route table built by the last ParseAll.
func (s *Site) Routes() *RouteTable {
	return s.routes
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

	// Rendering reads the page map and mutates page data, and the watcher
	// replaces the map and route table in ParseAll, so both run under the
	// site lock.
	var buf bytes.Buffer
	var route *Route
	var contentType string
	var err error
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		route = s.routes.Lookup(r.URL.Path)
		if route == nil {
			return
		}
		switch route.Kind {
		case RoutePage:
			err = route.Page.Render(&buf)
			contentType = route.Page.ContentType()
		case RouteOutput:
			contentType, err = route.Output.Render(&buf)
		}
	}()

	if route == nil {
		http.ServeFile(w, r, path.Join(s.dir, path.Clean("/"+r.URL.Path)))
		return
	}
	if route.Kind == RouteStatic {
		http.ServeFile(w, r, route.Source)
		return
	}
	if err != nil {
		log.Println(route.URL, ":", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	if isHTML(contentType) {
		injectLiveReload(w, buf)
		return
	}
	w.Write(buf.Bytes())
}
