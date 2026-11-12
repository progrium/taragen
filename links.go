package taragen

import (
	"bytes"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

const (
	// Links is the page data key for the pages a page's body links to.
	Links = "links"
	// Backlinks is the page data key for the pages that link to a page.
	Backlinks = "backlinks"
	// SiteURL is the page data key for the site's absolute origin, such as
	// "https://example.com", used to recognize absolute links to the site.
	SiteURL = "siteURL"
)

// setGraphData sets the page's links and backlinks data from the graph built
// by the last link pass, or empty lists before there is one. They replace
// any value the page set itself.
func (p *Page) setGraphData() {
	links := p.links
	if links == nil {
		links = []any{}
	}
	backlinks := p.backlinks
	if backlinks == nil {
		backlinks = []any{}
	}
	p.data[Links] = links
	p.data[Backlinks] = backlinks
}

// linkBase returns the URL a relative link in the page resolves against, as
// build publishes the page: "<path>/" for an extensionless page or the root,
// and "<path>" for a page whose name keeps an extension, like feed.xml.
func (p *Page) linkBase() string {
	u := p.URL()
	if u == "/" {
		return u
	}
	if filepath.Ext(p.path) == "" {
		return u + "/"
	}
	return u
}

// normalizeLink resolves an href found in a page published at base to a site
// path, such as "/docs/intro". It returns ok false for links that can't point
// at the site: other schemes, empty hrefs, and absolute URLs whose origin
// isn't siteURL.
func normalizeLink(href, base, siteURL string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" {
		return "", false
	}
	u, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	if u.Scheme != "" || u.Host != "" {
		site, err := url.Parse(siteURL)
		if siteURL == "" || err != nil || site.Host == "" {
			return "", false
		}
		if !strings.EqualFold(u.Host, site.Host) {
			return "", false
		}
		if u.Scheme != "" && !strings.EqualFold(u.Scheme, site.Scheme) {
			return "", false
		}
		u = &url.URL{Path: u.Path}
		if u.Path == "" {
			u.Path = "/"
		}
	} else if u.Opaque != "" {
		return "", false
	}
	// url.Parse percent-decodes Path, and ResolveReference removes "." and
	// ".." segments.
	resolved := (&url.URL{Path: base}).ResolveReference(&url.URL{Path: u.Path, RawQuery: u.RawQuery})
	p := path.Clean("/" + resolved.Path)
	return p, true
}

// bodyLinks returns the href of every <a> element in an HTML body, in order.
func bodyLinks(body []byte) []string {
	var hrefs []string
	z := html.NewTokenizer(bytes.NewReader(body))
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return hrefs
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			if string(name) != "a" || !hasAttr {
				continue
			}
			for {
				key, val, more := z.TagAttr()
				if string(key) == "href" {
					hrefs = append(hrefs, string(val))
					break
				}
				if !more {
					break
				}
			}
		}
	}
}

// linkTarget returns the page a route stands for as a link target, or nil.
// Only a page's own URL counts: outputs are the same page in another form,
// and static files aren't pages. A new kind of route that stands in for a
// page decides here whether links to it count.
func linkTarget(r *Route) *Page {
	if r == nil || r.Kind != RoutePage {
		return nil
	}
	return r.Page
}

// linkPages builds the link graph from the route table: it collects the
// links in every HTML page's body, then inverts them into backlinks. Drafts
// are neither sources nor targets unless the site shows drafts.
func (s *Site) linkPages() {
	var pages []*Page
	for _, r := range s.routes.Routes() {
		if p := linkTarget(r); p != nil && !slices.Contains(pages, p) {
			pages = append(pages, p)
		}
	}
	visible := func(p *Page) bool {
		return s.drafts || !p.Draft()
	}

	referrers := make(map[*Page][]*Page)
	for _, p := range pages {
		p.links = []any{}
		if !visible(p) || !isHTML(p.ContentType()) {
			continue
		}
		siteURL, _ := p.data[SiteURL].(string)
		base := p.linkBase()
		var targets []string
		seen := make(map[*Page]bool)
		for _, href := range bodyLinks(p.Body()) {
			u, ok := normalizeLink(href, base, siteURL)
			if !ok {
				continue
			}
			target := linkTarget(s.routes.Lookup(u))
			if target == nil || target == p || seen[target] || !visible(target) {
				continue
			}
			seen[target] = true
			targets = append(targets, target.URL())
			referrers[target] = append(referrers[target], p)
		}
		slices.Sort(targets)
		for _, t := range targets {
			p.links = append(p.links, t)
		}
	}

	for _, p := range pages {
		refs := referrers[p]
		sortPages(refs)
		p.backlinks = make([]any, 0, len(refs))
		for _, ref := range refs {
			p.backlinks = append(p.backlinks, referrerData(ref))
		}
	}
	for _, p := range pages {
		p.setGraphData()
	}
}

// referrerData is the data of a page that links to another, as it appears
// in backlinks: a copy without links, backlinks, or subpages, so the graph
// has no cycles.
func referrerData(p *Page) map[string]any {
	d := make(map[string]any, len(p.data))
	for k, v := range p.data {
		switch k {
		case Links, Backlinks, "subpages":
			continue
		}
		d[k] = v
	}
	return d
}
