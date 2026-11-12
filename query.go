package taragen

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"
	"time"
)

// comparePages is the total order of every listing: pages without a valid
// date first, then dated pages oldest first, ties broken by slug, then path.
func comparePages(a, b *Page) int {
	ta, okA := validDate(a)
	tb, okB := validDate(b)
	if okA != okB {
		if okA {
			return 1
		}
		return -1
	}
	if c := ta.Compare(tb); okA && c != 0 {
		return c
	}
	if c := strings.Compare(a.Slug(), b.Slug()); c != 0 {
		return c
	}
	return strings.Compare(a.path, b.path)
}

func validDate(p *Page) (time.Time, bool) {
	s, _ := p.data[Date].(string)
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

func sortPages(pages []*Page) {
	slices.SortFunc(pages, comparePages)
}

// isPattern reports whether a query is a pattern rather than a plain path.
func isPattern(query string) bool {
	return strings.Contains(query, "*")
}

// splitPath splits a page path into segments for matching. The root's path
// is empty, so it has no segments.
func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return nil
	}
	return strings.Split(p, "/")
}

// parsePattern splits a pattern into segments and checks each one, so a
// malformed pattern is an error even when nothing would reach the bad part.
func parsePattern(pattern string) ([]string, error) {
	segs := splitPath(pattern)
	for _, seg := range segs {
		if seg == "**" {
			continue
		}
		if _, err := path.Match(seg, ""); err != nil {
			return nil, fmt.Errorf("%w: %q", err, pattern)
		}
	}
	return segs, nil
}

// matchSegments matches a page path against a parsed pattern. A "**" segment
// matches zero or more whole segments; any other segment is matched against
// exactly one path segment with path.Match.
func matchSegments(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			for i := 0; i <= len(name); i++ {
				if matchSegments(pat[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if m, _ := path.Match(pat[0], name[0]); !m {
			return false
		}
		pat, name = pat[1:], name[1:]
	}
	return len(name) == 0
}

// listing returns the pages for one path or pattern, unsorted.
func (s *Site) listing(query string) ([]*Page, error) {
	if !isPattern(query) {
		if s.IsPage(query) {
			return s.Page(query).Subpages(), nil
		}
		return nil, nil
	}
	pat, err := parsePattern(query)
	if err != nil {
		return nil, err
	}
	// grouping nodes (folders without an index) hold the tree together but
	// have no content, so recursive listings leave them out
	recursive := slices.Contains(pat, "**")
	var pages []*Page
	for pathname, page := range s.pages {
		if !matchSegments(pat, splitPath(pathname)) {
			continue
		}
		for _, sub := range page.Subpages() {
			if recursive && len(sub.Source()) == 0 {
				continue
			}
			pages = append(pages, sub)
		}
	}
	return pages, nil
}

// Pages returns the listing for a path or pattern, as pages(query) in JSX.
func (s *Site) Pages(query string) ([]*Page, error) {
	return s.Query([]string{query}, nil)
}

// Query returns the union of the listings for each path or pattern, once
// each, keeping only pages whose data matches every key in where, sorted as
// one listing.
func (s *Site) Query(paths []string, where map[string]any) ([]*Page, error) {
	seen := make(map[*Page]bool)
	var pages []*Page
	for _, q := range paths {
		listing, err := s.listing(q)
		if err != nil {
			return nil, err
		}
		for _, p := range listing {
			if seen[p] || !matchWhere(p.data, where) {
				continue
			}
			seen[p] = true
			pages = append(pages, p)
		}
	}
	sortPages(pages)
	return pages, nil
}

// matchWhere reports whether data matches every key in where.
func matchWhere(data Data, where map[string]any) bool {
	for key, want := range where {
		got, ok := data[key]
		if !ok {
			return false
		}
		wants, isList := want.([]any)
		if !isList {
			wants = []any{want}
		}
		if !slices.ContainsFunc(wants, func(w any) bool { return matchValue(got, w) }) {
			return false
		}
	}
	return true
}

// matchValue reports whether a page value equals a wanted scalar, or is a
// list with an element equal to it.
func matchValue(got, want any) bool {
	if v := reflect.ValueOf(got); v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		for i := 0; i < v.Len(); i++ {
			if strictEqual(v.Index(i).Interface(), want) {
				return true
			}
		}
		return false
	}
	return strictEqual(got, want)
}

// strictEqual compares like JS ===: numbers of any Go type compare by value,
// and values of different kinds are never equal.
func strictEqual(a, b any) bool {
	na, okA := toNumber(a)
	nb, okB := toNumber(b)
	if okA || okB {
		return okA && okB && na == nb
	}
	switch a := a.(type) {
	case string:
		b, ok := b.(string)
		return ok && a == b
	case bool:
		b, ok := b.(bool)
		return ok && a == b
	}
	return false
}

func toNumber(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}

// pageQuery is the parsed argument of pages().
type pageQuery struct {
	paths []string
	where map[string]any
}

// parsePageQuery parses the exported value of the object form of pages().
func parsePageQuery(arg map[string]any) (q pageQuery, err error) {
	q.paths = []string{"**"}
	for key, val := range arg {
		switch key {
		case "path":
			switch v := val.(type) {
			case string:
				q.paths = []string{v}
			case []any:
				q.paths = nil
				for _, p := range v {
					s, ok := p.(string)
					if !ok {
						return q, fmt.Errorf("path must be a string or a list of strings")
					}
					q.paths = append(q.paths, s)
				}
			default:
				return q, fmt.Errorf("path must be a string or a list of strings")
			}
		case "where":
			where, ok := val.(map[string]any)
			if !ok {
				return q, fmt.Errorf("where must be an object")
			}
			for k, w := range where {
				if !validWhereValue(w, true) {
					return q, fmt.Errorf("where.%s must be a string, number, boolean, or a list of them", k)
				}
			}
			q.where = where
		default:
			return q, fmt.Errorf("unknown key %q", key)
		}
	}
	return q, nil
}

func validWhereValue(v any, allowList bool) bool {
	switch v := v.(type) {
	case string, bool:
		return true
	case []any:
		if !allowList {
			return false
		}
		for _, e := range v {
			if !validWhereValue(e, false) {
				return false
			}
		}
		return true
	}
	_, ok := toNumber(v)
	return ok
}

// describeQuery formats a pages() argument for error messages.
func describeQuery(arg any) string {
	b, err := json.Marshal(arg)
	if err != nil {
		return fmt.Sprintf("%v", arg)
	}
	return string(b)
}
