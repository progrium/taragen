package taragen

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

// Params is the page data key holding a generated page's parameter set.
const Params = "params"

// paramSegment matches a path segment that is a parameter, like [year].
var paramSegment = regexp.MustCompile(`^\[([a-z][a-z0-9]*)\]$`)

// pageTemplate is a page source with a bracketed parameter in its path, which
// renders one generated page per parameter set.
type pageTemplate struct {
	// file is the source path relative to the site root, such as
	// "logs/[year]/[month].jsx".
	file string
	// pattern is the URL pattern without a leading slash, such as
	// "logs/[year]/[month]".
	pattern string
	// names are the parameter names, in path order.
	names []string
}

// pathSegment returns the part of a site entry's name that can be a
// parameter: a directory's whole name, or a file's name before its
// extensions.
func pathSegment(name string, isDir bool) string {
	if isDir {
		return name
	}
	seg, _, _ := strings.Cut(name, ".")
	return seg
}

// checkSegment reports whether a site entry's name is a parameter. A name
// with brackets that isn't exactly [name], or with brackets in its
// extensions, is an error, so a typo can't silently become a page.
func checkSegment(rel string, name string, isDir bool) (bool, error) {
	seg := pathSegment(name, isDir)
	if paramSegment.MatchString(seg) && !strings.ContainsAny(name[len(seg):], "[]") {
		return true, nil
	}
	if strings.ContainsAny(name, "[]") {
		return false, fmt.Errorf("%s: a bracketed name must be a whole path segment like [name], with a lowercase letter followed by lowercase letters and digits", rel)
	}
	return false, nil
}

// templateParams returns the parameter names in a URL pattern, in order,
// and fails on a name used twice.
func templateParams(pattern string) ([]string, error) {
	var names []string
	for _, seg := range strings.Split(pattern, "/") {
		m := paramSegment.FindStringSubmatch(pathSegment(seg, false))
		if m == nil {
			continue
		}
		for _, n := range names {
			if n == m[1] {
				return nil, fmt.Errorf("parameter [%s] appears more than once", n)
			}
		}
		names = append(names, m[1])
	}
	return names, nil
}

// checkParamValue checks that a parameter value is a single path segment
// that the walker doesn't treat specially.
func checkParamValue(v string) error {
	switch {
	case v == "":
		return fmt.Errorf("value is empty")
	case strings.ContainsAny(v, `/\`):
		return fmt.Errorf("value %q contains a slash", v)
	case v == "." || v == "..":
		return fmt.Errorf("value %q is not a path segment", v)
	case strings.HasPrefix(v, ".") || strings.HasPrefix(v, "_"):
		return fmt.Errorf("value %q starts with %q", v, v[:1])
	}
	return nil
}

// fill returns the page path for a parameter set.
func (t *pageTemplate) fill(set map[string]string) string {
	p := t.pattern
	for _, n := range t.names {
		p = strings.Replace(p, "["+n+"]", set[n], 1)
	}
	return p
}

// paramsStatement returns a template's one leading params(...) statement.
func (t *pageTemplate) paramsStatement(src []byte) ([]byte, error) {
	calls, _, err := leadingCalls(src)
	if err != nil {
		return nil, err
	}
	var stmt []byte
	for _, c := range calls {
		if c.name != "params" {
			continue
		}
		if stmt != nil {
			return nil, fmt.Errorf("more than one params(...) statement")
		}
		stmt = src[c.start:c.end]
	}
	if stmt == nil {
		return nil, fmt.Errorf("a template needs a leading params(...) statement")
	}
	return stmt, nil
}

// paramSets runs a template's params(...) statement alone in a fresh VM,
// with the globals from its real ancestor directories and the built-in
// pages(), and returns its parameter sets, validated and with duplicates
// removed, in the order they were returned.
func (s *Site) paramSets(t *pageTemplate) (sets []map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		if err != nil {
			err = fmt.Errorf("%s: params: %w", t.file, err)
		}
	}()
	src, err := os.ReadFile(filepath.Join(s.dir, filepath.FromSlash(t.file)))
	if err != nil {
		return nil, err
	}
	stmt, err := t.paramsStatement(src)
	if err != nil {
		return nil, err
	}

	// a probe page at the pattern finds the globals along it; bracketed
	// directories can't hold _globals.jsx, so only real ancestors count
	probe := &Page{path: t.pattern, site: s, data: Data{}}
	if err := probe.loadGlobals(); err != nil {
		return nil, err
	}
	vm, prg, err := setupJSX(t.file, stmt)
	if err != nil {
		return nil, err
	}
	for k, v := range probe.globals {
		if k == "page" || k == Params {
			continue
		}
		if err := vm.Set(k, v); err != nil {
			return nil, err
		}
	}
	var arg goja.Value
	if err := vm.Set(Params, func(call goja.FunctionCall) goja.Value {
		arg = call.Argument(0)
		return goja.Undefined()
	}); err != nil {
		return nil, err
	}
	if _, err := vm.RunProgram(prg); err != nil {
		return nil, err
	}
	if fn, ok := goja.AssertFunction(arg); ok {
		if arg, err = fn(goja.Undefined()); err != nil {
			return nil, err
		}
	}
	arr, ok := arg.(*goja.Object)
	if !ok || arr.ClassName() != "Array" {
		return nil, fmt.Errorf("params must be an array of parameter sets, or a function returning one")
	}

	seen := make(map[string]bool)
	length := arr.Get("length").ToInteger()
	for i := int64(0); i < length; i++ {
		set, err := paramSet(t, arr.Get(strconv.FormatInt(i, 10)))
		if err != nil {
			return nil, fmt.Errorf("set %d: %w", i, err)
		}
		key := t.fill(set)
		if seen[key] {
			continue
		}
		seen[key] = true
		sets = append(sets, set)
	}
	return sets, nil
}

// paramSet converts and validates one parameter set: an object with exactly
// the template's parameter names as keys, and string or number values.
func paramSet(t *pageTemplate, v goja.Value) (map[string]string, error) {
	obj, ok := v.(*goja.Object)
	if !ok || obj.ClassName() != "Object" {
		return nil, fmt.Errorf("a parameter set must be an object, got %v", v)
	}
	set := make(map[string]string)
	for _, k := range obj.Keys() {
		if !contains(t.names, k) {
			return nil, fmt.Errorf("unknown parameter %q (the template's parameters are %s)", k, strings.Join(t.names, ", "))
		}
		val := obj.Get(k)
		switch val.Export().(type) {
		case string, int64, float64:
		default:
			return nil, fmt.Errorf("parameter %q: values must be strings or numbers, got %v", k, val)
		}
		s := val.String()
		if err := checkParamValue(s); err != nil {
			return nil, fmt.Errorf("parameter %q: %w", k, err)
		}
		set[k] = s
	}
	for _, n := range t.names {
		if _, ok := set[n]; !ok {
			return nil, fmt.Errorf("missing parameter %q", n)
		}
	}
	return set, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// generatePages runs the parameters pass over the templates, after the real
// pages are parsed, then registers and parses the generated pages. It
// returns the paths of the pages it added, generated or bare directories.
func (s *Site) generatePages(templates []*pageTemplate) ([]string, error) {
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].file < templates[j].file
	})

	// run every params statement first, so each sees only real pages
	type generated struct {
		t   *pageTemplate
		set map[string]string
	}
	byPath := make(map[string]generated)
	var order []string
	for _, t := range templates {
		sets, err := s.paramSets(t)
		if err != nil {
			return nil, err
		}
		for _, set := range sets {
			gp := t.fill(set)
			if other, ok := byPath[gp]; ok {
				return nil, fmt.Errorf("generated page collision at %s: %s and %s", path.Clean("/"+gp), other.t.file, t.file)
			}
			byPath[gp] = generated{t, set}
			order = append(order, gp)
		}
	}

	// register: a real source wins, and a bare directory is filled
	var added []string
	var pages []*Page
	for _, gp := range order {
		g := byPath[gp]
		if existing, ok := s.pages[gp]; ok && !existing.virtualDir && len(existing.Source()) > 0 {
			continue
		}
		params := make(map[string]any, len(g.set))
		for k, v := range g.set {
			params[k] = v
		}
		p := &Page{
			path:     gp,
			site:     s,
			template: g.t.file,
			params:   params,
			data: Data{
				Path:   path.Clean("/" + gp),
				Slug:   path.Base(gp),
				Params: params,
			},
		}
		s.pages[gp] = p
		pages = append(pages, p)
		added = append(added, gp)
	}

	// a generated page whose parent directory has no page gets a bare
	// directory, as walk adds for real directories
	for _, p := range pages {
		for dir := path.Dir(p.path); dir != "."; dir = path.Dir(dir) {
			if _, ok := s.pages[dir]; ok {
				break
			}
			s.pages[dir] = &Page{
				path:       dir,
				site:       s,
				virtualDir: true,
				data: Data{
					Path:  path.Clean("/" + dir),
					Slug:  path.Base(dir),
					IsDir: true,
				},
			}
			added = append(added, dir)
		}
	}

	for _, p := range pages {
		if err := p.Parse(); err != nil {
			return nil, fmt.Errorf("%s (%s): %w", p.URL(), p.template, err)
		}
	}
	return added, nil
}
