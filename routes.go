package taragen

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// RouteKind says what produces a route's URL.
type RouteKind string

const (
	// RoutePage is a page with a source file, or a page generated from a
	// template.
	RoutePage RouteKind = "page"
	// RouteOutput is an output of a page, from an _outputs template.
	RouteOutput RouteKind = "output"
	// RouteStatic is a file copied verbatim.
	RouteStatic RouteKind = "static"
)

// Route is one URL the site serves and what produces it.
type Route struct {
	Kind RouteKind
	// URL is the path the route is served at, such as "/", "/about",
	// "/about.txt", or "/img/logo.svg".
	URL string
	// File is where build writes the route, relative to the output
	// directory, such as "index.html" or "about/index.html".
	File string
	// Page is the page for page and output routes.
	Page *Page
	// Output is the output for output routes.
	Output *Output
	// Source is the absolute source filename for static routes.
	Source string
}

// Draft reports whether the route is a draft: a draft page or an output of
// one. build skips drafts and serve serves them.
func (r *Route) Draft() bool {
	return r.Page != nil && r.Page.Draft()
}

// String describes the route's producer, for error messages.
func (r *Route) String() string {
	switch r.Kind {
	case RoutePage:
		if r.Page.template != "" {
			return fmt.Sprintf("generated page %s (%s)", r.URL, r.Page.template)
		}
		return fmt.Sprintf("page %s (%s)", r.URL, r.Page.file)
	case RouteOutput:
		return fmt.Sprintf("output %s of page %s (%s)", r.Output.Name, r.Page.URL(), r.Output.Template)
	default:
		return fmt.Sprintf("%s file %s", r.Kind, r.File)
	}
}

// RouteTable maps every URL the site serves to what produces it. Each URL,
// and each output file, has exactly one producer.
type RouteTable struct {
	byURL  map[string]*Route
	byFile map[string]*Route
}

func NewRouteTable() *RouteTable {
	return &RouteTable{
		byURL:  make(map[string]*Route),
		byFile: make(map[string]*Route),
	}
}

func routeKey(urlPath string) string {
	return strings.Trim(urlPath, "/")
}

// Add adds a route. A route whose URL or output file is already taken is an
// error naming both producers.
func (t *RouteTable) Add(r *Route) error {
	if other, ok := t.byURL[routeKey(r.URL)]; ok {
		return fmt.Errorf("route collision at %s: %s and %s", r.URL, other, r)
	}
	if other, ok := t.byFile[r.File]; ok {
		return fmt.Errorf("route collision at %s: %s and %s", r.File, other, r)
	}
	t.byURL[routeKey(r.URL)] = r
	t.byFile[r.File] = r
	return nil
}

// Lookup returns the route for a request path, ignoring leading and
// trailing slashes, or nil.
func (t *RouteTable) Lookup(urlPath string) *Route {
	if t == nil {
		return nil
	}
	return t.byURL[routeKey(urlPath)]
}

// Routes returns every route, ordered by output file.
func (t *RouteTable) Routes() []*Route {
	if t == nil {
		return nil
	}
	routes := make([]*Route, 0, len(t.byFile))
	for _, r := range t.byFile {
		routes = append(routes, r)
	}
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].File < routes[j].File
	})
	return routes
}

// pageRoute returns the route for a page with a source. A page without an
// extension (and the root) is written to <path>/index.html for a clean URL,
// and one that keeps an extension, like feed.xml, is written as-is.
func pageRoute(p *Page) *Route {
	file := p.path
	if filepath.Ext(p.path) == "" || p.path == "." {
		file = path.Join(p.path, "index.html")
	}
	return &Route{Kind: RoutePage, URL: p.URL(), File: file, Page: p}
}

// staticRoute returns the route for a file copied verbatim from the site.
func (s *Site) staticRoute(filename string) *Route {
	rel := filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(filename, s.dir), string(filepath.Separator)))
	return &Route{Kind: RouteStatic, URL: "/" + rel, File: rel, Source: filename}
}

// buildRoutes builds the route table from the parsed pages and the static
// files found by walk.
func (s *Site) buildRoutes(pagepaths []string, statics []string) (*RouteTable, error) {
	t := NewRouteTable()
	for _, filename := range statics {
		if err := t.Add(s.staticRoute(filename)); err != nil {
			return nil, err
		}
	}
	for _, pagepath := range pagepaths {
		p := s.pages[strings.TrimPrefix(pagepath, "/")]
		if p == nil || len(p.Source()) == 0 {
			// bare directories are grouping nodes for pages(), not routes
			continue
		}
		if err := t.Add(pageRoute(p)); err != nil {
			return nil, err
		}
		for _, o := range p.outputs {
			url := o.URL()
			r := &Route{Kind: RouteOutput, URL: url, File: strings.TrimPrefix(url, "/"), Page: p, Output: o}
			if err := t.Add(r); err != nil {
				return nil, err
			}
		}
	}
	return t, nil
}
