package taragen

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// RedirectsFile is the root file listing redirects, in a subset of Netlify's
// _redirects format.
const RedirectsFile = "_redirects"

// RedirectRule is one rule from the _redirects file.
type RedirectRule struct {
	// Line is the rule's line number in _redirects, from 1.
	Line int
	// From is the normalized source path. For a wildcard rule it is the path
	// the wildcard is below, without "/*".
	From string
	// Wildcard is true when the rule's from ended in "/*".
	Wildcard bool
	// To is the target as written: a path, possibly with a query string, or
	// an absolute http or https URL, possibly containing :splat.
	To string
	// Status is the redirect's HTTP status.
	Status int
}

func (r *RedirectRule) String() string {
	return fmt.Sprintf("%s line %d", RedirectsFile, r.Line)
}

// offSite reports whether the rule points to an absolute URL.
func (r *RedirectRule) offSite() bool {
	return !strings.HasPrefix(r.To, "/")
}

// toPath returns the target's path without its query string.
func (r *RedirectRule) toPath() string {
	p, _, _ := strings.Cut(r.To, "?")
	return p
}

// match reports whether a cleaned, escaped request path is below the rule's
// wildcard, and what the wildcard matched.
func (r *RedirectRule) match(urlPath string) (splat string, ok bool) {
	if !r.Wildcard {
		return "", false
	}
	prefix := strings.TrimSuffix(r.From, "/") + "/"
	if !strings.HasPrefix(urlPath, prefix) || len(urlPath) == len(prefix) {
		return "", false
	}
	return urlPath[len(prefix):], true
}

// target returns the rule's target with :splat replaced.
func (r *RedirectRule) target(splat string) string {
	return strings.ReplaceAll(r.To, ":splat", splat)
}

var (
	redirectStatuses = map[string]int{"301": 301, "302": 302, "307": 307, "308": 308}
	placeholder      = regexp.MustCompile(`:[A-Za-z_]`)
)

// parseRedirects parses the supported subset of Netlify's _redirects format:
// "from to [status]" per line, with blank lines and "#" comment lines
// ignored. Anything else is an error naming the file and line.
func parseRedirects(src []byte) ([]*RedirectRule, error) {
	var rules []*RedirectRule
	scanner := bufio.NewScanner(bytes.NewReader(src))
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		rule, err := parseRedirectRule(strings.Fields(text))
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", RedirectsFile, line, err)
		}
		rule.Line = line
		rules = append(rules, rule)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", RedirectsFile, err)
	}
	return rules, nil
}

func parseRedirectRule(fields []string) (*RedirectRule, error) {
	if len(fields) < 2 {
		return nil, fmt.Errorf("a rule needs a from path and a target")
	}
	if len(fields) > 3 {
		return nil, fmt.Errorf("unsupported extra fields %q; a rule is from, to, and an optional status", strings.Join(fields[3:], " "))
	}
	rule := &RedirectRule{Status: 301}
	if len(fields) == 3 {
		status, ok := redirectStatuses[fields[2]]
		if !ok {
			return nil, fmt.Errorf("unsupported status %q; use 301, 302, 307, or 308", fields[2])
		}
		rule.Status = status
	}

	from := fields[0]
	if strings.HasSuffix(from, "/*") {
		rule.Wildcard = true
		from = strings.TrimSuffix(from, "*")
	}
	if strings.Contains(from, "*") {
		return nil, fmt.Errorf("from %q: * may only end the path, as /*", fields[0])
	}
	if strings.Contains(from, ":") {
		return nil, fmt.Errorf("from %q: placeholders are not supported", fields[0])
	}
	from, err := normalizeURLPath(from)
	if err != nil {
		return nil, fmt.Errorf("from %q: %w", fields[0], err)
	}
	rule.From = from

	to := fields[1]
	rest := to
	switch {
	case strings.HasPrefix(to, "http://"), strings.HasPrefix(to, "https://"):
		// only the part after the host can hold placeholders
		host := strings.TrimPrefix(strings.TrimPrefix(to, "http://"), "https://")
		host, rest, _ = strings.Cut(host, "/")
		rest = "/" + rest
		if host == "" {
			return nil, fmt.Errorf("to %q: missing host", to)
		}
	case strings.HasPrefix(to, "//"), !strings.HasPrefix(to, "/"):
		return nil, fmt.Errorf("to %q: must be a path starting with / or an http or https URL", to)
	}
	if strings.Contains(rest, ":splat") && !rule.Wildcard {
		return nil, fmt.Errorf("to %q: :splat needs a from ending in /*", to)
	}
	if placeholder.MatchString(strings.ReplaceAll(rest, ":splat", "")) {
		return nil, fmt.Errorf("to %q: placeholders other than :splat are not supported", to)
	}
	if strings.ContainsAny(to, `#\`) {
		return nil, fmt.Errorf(`to %q: must not contain # or \`, to)
	}
	rule.To = to
	return rule, nil
}

// readRedirects reads the site's root _redirects file, if there is one.
func (s *Site) readRedirects() (filename string, rules []*RedirectRule, err error) {
	filename = filepath.Join(s.dir, RedirectsFile)
	src, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	rules, err = parseRedirects(src)
	return filename, rules, err
}

// addRedirectRoutes adds rows for the _redirects file: the file itself, a
// redirect row for each rule without a wildcard, and one for each path a
// wildcard rule expands to. It runs after every page, output, static, and
// alias row is in the table.
func (s *Site) addRedirectRoutes(t *RouteTable, filename string, rules []*RedirectRule) error {
	if filename == "" {
		return nil
	}
	if err := t.Add(s.staticRoute(filename)); err != nil {
		return err
	}
	for _, rule := range rules {
		if rule.Wildcard {
			continue
		}
		r := &Route{Kind: RouteRedirect, URL: rule.From, File: urlFile(rule.From), Rule: rule, Target: rule.To}
		if !rule.offSite() {
			if target := t.Lookup(rule.toPath()); target != nil && target.Kind == RoutePage && !isHTML(target.Page.ContentType()) {
				r.Page = target.Page
			}
		}
		if !r.isCopy() && !stubbable(r.URL) {
			s.warn("%s: %s has an extension, so build writes no stub for it", rule, rule.From)
		}
		if err := t.Add(r); err != nil {
			return err
		}
	}

	// Wildcard rows are expanded against the HTML pages already in the
	// table, so they never claim a path another row has; the first rule
	// wins between rules.
	var pages []*Route
	for _, r := range t.Routes() {
		if r.Kind == RoutePage && isHTML(r.Page.ContentType()) {
			pages = append(pages, r)
		}
	}
	for _, rule := range rules {
		if !rule.Wildcard {
			continue
		}
		before, after, ok := strings.Cut(rule.toPath(), ":splat")
		if rule.offSite() || !ok || strings.Contains(after, ":splat") {
			s.warn("%s: %s/* can't be expanded into stubs, so it only works on hosts that read %s", rule, strings.TrimSuffix(rule.From, "/"), RedirectsFile)
			continue
		}
		for _, page := range pages {
			url := page.URL
			if len(url) <= len(before)+len(after) || !strings.HasPrefix(url, before) || !strings.HasSuffix(url, after) {
				continue
			}
			splat := url[len(before) : len(url)-len(after)]
			from := path.Join(rule.From, splat)
			if !stubbable(from) || t.Lookup(from) != nil {
				continue
			}
			r := &Route{Kind: RouteRedirect, URL: from, File: urlFile(from), Rule: rule, Target: rule.target(splat), Page: page.Page}
			if err := t.Add(r); err != nil {
				// another row writes the same file
				continue
			}
		}
	}
	return nil
}

// matchRedirect matches a request path against the wildcard rules in order,
// returning the first match's target, or "".
func (s *Site) matchRedirect(escapedPath string) (rule *RedirectRule, target string) {
	cleaned := path.Clean("/" + escapedPath)
	for _, rule := range s.redirects {
		if splat, ok := rule.match(cleaned); ok {
			return rule, rule.target(splat)
		}
	}
	return nil, ""
}

// warn records a warning from the last ParseAll; build prints them.
func (s *Site) warn(format string, args ...any) {
	s.warnings = append(s.warnings, fmt.Sprintf(format, args...))
}
