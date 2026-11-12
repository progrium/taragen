package taragen

import (
	"fmt"
	"io"
	"path"
	"strings"
)

// Aliases is the page data key listing old URL paths that redirect to the
// page.
const Aliases = "aliases"

// normalizeURLPath checks and normalizes an alias or redirect path: it must
// start with "/", has its trailing "/" removed (except for "/" itself), and
// must not contain "?", "#", "\", an empty segment, or a "." or ".." segment,
// so it can't be written outside the output directory.
func normalizeURLPath(p string) (string, error) {
	if !strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("must start with /")
	}
	if strings.ContainsAny(p, `?#\`) {
		return "", fmt.Errorf(`must not contain ?, #, or \`)
	}
	if p == "/" {
		return p, nil
	}
	p = strings.TrimSuffix(p, "/")
	for _, seg := range strings.Split(p[1:], "/") {
		switch seg {
		case "":
			return "", fmt.Errorf("must not contain an empty segment")
		case ".", "..":
			return "", fmt.Errorf("must not contain a %s segment", seg)
		}
	}
	return p, nil
}

// urlFile maps a URL path to the file build writes it to, the way page paths
// are mapped: a last segment without an extension (and the root) is written
// to <path>/index.html, and one with an extension is written as-is.
func urlFile(urlPath string) string {
	rel := strings.Trim(urlPath, "/")
	if rel == "" || path.Ext(rel) == "" {
		return path.Join(rel, "index.html")
	}
	return rel
}

// normalizeAliases reads the page's aliases from its data, checks and
// normalizes them, and writes the normalized list back to page.aliases.
func (p *Page) normalizeAliases() error {
	v, ok := p.data[Aliases]
	if !ok || v == nil {
		p.aliases = nil
		return nil
	}
	var raw []any
	switch v := v.(type) {
	case string:
		raw = []any{v}
	case []string:
		for _, s := range v {
			raw = append(raw, s)
		}
	case []any:
		raw = v
	default:
		return fmt.Errorf("page %s: aliases must be a string or a list of strings, got %T", p.URL(), v)
	}
	var aliases []string
	seen := make(map[string]bool)
	for _, entry := range raw {
		s, ok := entry.(string)
		if !ok {
			return fmt.Errorf("page %s: alias %v is not a string", p.URL(), entry)
		}
		alias, err := normalizeURLPath(s)
		if err != nil {
			return fmt.Errorf("page %s: invalid alias %q: %w", p.URL(), s, err)
		}
		if seen[alias] {
			continue
		}
		seen[alias] = true
		aliases = append(aliases, alias)
	}
	p.aliases = aliases
	list := make([]any, len(aliases))
	for i, a := range aliases {
		list[i] = a
	}
	p.data[Aliases] = list
	return nil
}

// Aliases returns the page's normalized aliases, as of its last parse.
func (p *Page) Aliases() []string {
	return p.aliases
}

// writeStub writes a redirect stub page pointing at target. Stubs don't run
// layouts, globals, or output templates, so they are the same on every build.
func writeStub(w io.Writer, target string) error {
	a, t := escapeAttr(target), escapeText(target)
	_, err := fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Redirecting to %s</title>
<link rel="canonical" href="%s">
<meta name="robots" content="noindex">
<meta http-equiv="refresh" content="0; url=%s">
</head>
<body><a href="%s">%s</a></body>
</html>
`, t, a, a, a, t)
	return err
}

// isCopy reports whether an alias or redirect route serves a copy of a
// non-HTML page rather than redirecting to it.
func (r *Route) isCopy() bool {
	return r.Page != nil && !isHTML(r.Page.ContentType())
}

// stubbable reports whether build can write an HTML stub at the route's URL:
// an HTML stub at a URL with another extension would be served with the
// wrong type.
func stubbable(urlPath string) bool {
	ext := path.Ext(urlPath)
	return ext == "" || ext == ".html"
}
