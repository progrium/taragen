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
	// redirects are the _redirects rules, and warnings what the last
	// ParseAll found that build should report
	redirects []*RedirectRule
	warnings  []string

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

	for _, w := range s.warnings {
		fmt.Println("warning:", w)
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
	if r.Kind == RouteRedirect && !r.isCopy() && !stubbable(r.URL) {
		// warned about in ParseAll; only hosts that read _redirects get it
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
	case RouteAlias, RouteRedirect:
		if r.isCopy() {
			err = r.Page.Render(f)
		} else {
			err = writeStub(f, r.Target)
		}
		if err != nil {
			return err
		}
		if r.isCopy() {
			fmt.Println("copied:", r.URL, "from", r.Page.URL())
		} else {
			fmt.Println("redirect:", r.URL, "->", r.Target)
		}
		return f.Close()
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

// walk calls fn for every page source, directory, and static file in the
// site, and template for every page source with a bracketed parameter in its
// path. Bracketed directories are walked but aren't pages, and anything in
// one that can't be a template is an error.
func (s *Site) walk(fn func(filename string, pagepath string) error, template func(t *pageTemplate) error) error {
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
		rel, err := filepath.Rel(s.dir, curPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		inParamDir := false
		if rel != "." {
			for _, dir := range strings.Split(path.Dir(rel), "/") {
				if paramSegment.MatchString(dir) {
					inParamDir = true
				}
			}
		}
		if strings.HasPrefix(info.Name(), "_") {
			if inParamDir {
				return fmt.Errorf("%s: _-prefixed files aren't allowed inside a bracketed directory", rel)
			}
			if info.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		isParam := false
		if rel != "." {
			if isParam, err = checkSegment(rel, info.Name(), info.IsDir()); err != nil {
				return err
			}
		}

		isPage := false
		for key := range Formats {
			if strings.HasSuffix(info.Name(), key) {
				isPage = true
			}
		}

		if isParam || inParamDir {
			switch {
			case info.IsDir():
				// a bracketed directory is a pattern, not a page
				return nil
			case !isPage:
				return fmt.Errorf("%s: static files aren't allowed inside a bracketed directory", rel)
			case filepath.Ext(info.Name()) != ExtJSX:
				return fmt.Errorf("%s: templates must be .jsx", rel)
			}
			pattern := strings.TrimPrefix(s.normalizePath(curPath), "/")
			names, err := templateParams(pattern)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			return template(&pageTemplate{file: rel, pattern: pattern, names: names})
		}

		if !isPage && !info.IsDir() {
			return fn(curPath, "")
		}

		return fn(curPath, s.normalizePath(curPath))
	})
}

// ParseAll resets the page map, parses every page found by walk, generates
// the pages for every template from the real pages, builds the route table,
// and builds the link graph.
func (s *Site) ParseAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pages = make(map[string]*Page)
	s.routes = nil
	s.redirects, s.warnings = nil, nil

	redirectsFile, rules, err := s.readRedirects()
	if err != nil {
		return err
	}

	var pagepaths, statics []string
	var templates []*pageTemplate
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
	}, func(t *pageTemplate) error {
		templates = append(templates, t)
		return nil
	}); err != nil {
		return err
	}

	generated, err := s.generatePages(templates)
	if err != nil {
		return err
	}
	for _, gp := range generated {
		// a generated page can fill a bare directory walk already found
		if !parsed["/"+gp] {
			pagepaths = append(pagepaths, gp)
		}
	}

	// A body that called pages() saw only the pages parsed before it, so it
	// is parsed again now that the whole page set exists, generated pages
	// included, and its links
	// don't depend on walk order.
	for _, pagepath := range pagepaths {
		if p := s.Page(pagepath); p.usesPages {
			if err := p.Parse(); err != nil {
				return err
			}
		}
	}

	routes, err := s.buildRoutes(pagepaths, statics, redirectsFile, rules)
	if err != nil {
		return err
	}
	s.routes = routes
	s.redirects = rules
	s.linkPages()
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
	var redirect string
	var status int
	var err error
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		route = s.routes.Lookup(r.URL.Path)
		if route == nil {
			// wildcard rules come after the route table and before disk
			if rule, target := s.matchRedirect(r.URL.EscapedPath()); rule != nil {
				redirect, status = target, rule.Status
			}
			return
		}
		switch route.Kind {
		case RoutePage:
			err = route.Page.Render(&buf)
			contentType = route.Page.ContentType()
		case RouteOutput:
			contentType, err = route.Output.Render(&buf)
		case RouteAlias, RouteRedirect:
			if route.isCopy() {
				err = route.Page.Render(&buf)
				contentType = route.Page.ContentType()
				return
			}
			redirect, status = route.Target, http.StatusMovedPermanently
			if route.Rule != nil {
				status = route.Rule.Status
			}
		}
	}()

	if redirect != "" {
		// a rule keeps the request's query string when its target has none
		if (route == nil || route.Rule != nil) && r.URL.RawQuery != "" && !strings.Contains(redirect, "?") {
			redirect += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, redirect, status)
		return
	}
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
