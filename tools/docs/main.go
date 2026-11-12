// Command docs splits MANUAL.md into the website's docs pages.
//
// The manual stays one Markdown file in the repository. This tool cuts it at
// its "##" headings into numbered chapters under web/docs/manual, the
// "### " sections of "## Tutorials" into web/docs/tutorials, and the
// reference tables (and troubleshooting) into web/docs/reference. It rewrites
// the manual's "#anchor" links to point at the pages they now live on, turns
// pipe tables into HTML (taragen's Markdown has no table extension), and
// copies the whole manual to web/manual.txt.
//
// Run it from the repository root after editing MANUAL.md:
//
//	go run ./tools/docs
package main

import (
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	manualPath = "MANUAL.md"
	webDir     = "web"
)

// parts groups the manual's chapters for the sidebar. Each entry names the
// first chapter of a part; later chapters stay in the part before them.
var parts = []struct{ first, name string }{
	{"Install", "getting started"},
	{"How a site is laid out", "building a site"},
	{"Writing JSX", "formats"},
	{"Listing pages with `pages()`", "content"},
	{"Non-HTML pages and feeds", "outputs"},
	{"Generated pages", "site structure"},
	{"Code highlighting", "shipping"},
}

// shortSlugs replaces a few heading-derived slugs with shorter page names.
var shortSlugs = map[string]string{
	"listing-pages-with-pages":   "listing-pages",
	"how-a-site-is-laid-out":     "site-layout",
	"built-in-jsx-globals":       "jsx-globals",
	"output-template-data-keys":  "output-keys",
	"_redirects-syntax":          "redirects",
	"file-naming-summary":        "file-names",
	"errors-and-troubleshooting": "errors",
}

// aliases keeps the old docs URLs working.
var aliases = map[string][]string{
	"/docs/manual/quick-start":                                 {"/docs/getting-started", "/docs/install", "/start"},
	"/docs/tutorials/set-up-a-blog":                            {"/docs/tutorials/setting-up-a-blog", "/docs/tutorials/rss-feed"},
	"/docs/tutorials/highlight-the-current-page-in-navigation": {"/docs/tutorials/active-nav-state"},
	"/docs/tutorials/format-dates":                             {"/docs/tutorials/date-formatter-utility"},
}

// shortTitles are the sidebar labels for long tutorial titles.
var shortTitles = map[string]string{
	"Add yearly archive pages":                 "Yearly archive pages",
	"Build a docs section with a sidebar":      "Docs sidebar",
	"Highlight the current page in navigation": "Current page in nav",
	"Offer a plain-text version of every page": "Plain-text pages",
	"Show what links to a page":                "What links here",
	"Move a section without breaking links":    "Move a section",
}

type section struct {
	title string
	level int      // 2 or 3
	lines []string // body, without the heading line
}

type doc struct {
	kind  string // manual, tutorials, reference
	slug  string
	title string
	num   int
	part  string
	lines []string // body; its headings already demoted
	shift int      // how many levels headings were raised by
	toc   [][2]string
}

func (d *doc) path() string { return "/docs/" + d.kind + "/" + d.slug }

func main() {
	src, err := os.ReadFile(manualPath)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")

	h2s := splitSections(lines)
	var docs []*doc
	anchors := map[string]string{} // manual anchor -> new href
	seen := map[string]int{}
	anchor := func(title string) string {
		a := slugify(title)
		n := seen[a]
		seen[a]++
		if n > 0 {
			a = fmt.Sprintf("%s-%d", a, n)
		}
		return a
	}
	anchor("Taragen manual")

	part := parts[0].name
	chapter := 0
	for _, s := range h2s {
		a := anchor(s.title)
		switch s.title {
		case "Contents", "":
			continue
		case "Tutorials":
			anchors[a] = "/docs/tutorials"
			n := 0
			for _, t := range splitSub(s.lines) {
				ta := anchor(t.title)
				n++
				d := &doc{kind: "tutorials", slug: short(ta), title: t.title, num: n, lines: t.lines, shift: 2}
				anchors[ta] = d.path()
				collectAnchors(d, anchor, anchors)
				docs = append(docs, d)
			}
			continue
		case "Reference":
			anchors[a] = "/docs/reference"
			n := 0
			for _, t := range splitSub(s.lines) {
				ta := anchor(t.title)
				n++
				d := &doc{kind: "reference", slug: short(ta), title: t.title, num: n, lines: t.lines, shift: 2}
				anchors[ta] = d.path()
				collectAnchors(d, anchor, anchors)
				docs = append(docs, d)
			}
			continue
		case "Errors and troubleshooting":
			d := &doc{kind: "reference", slug: short(a), title: s.title, num: 99, lines: s.lines, shift: 1}
			anchors[a] = d.path()
			collectAnchors(d, anchor, anchors)
			docs = append(docs, d)
			continue
		}
		for _, p := range parts {
			if p.first == s.title {
				part = p.name
			}
		}
		chapter++
		d := &doc{kind: "manual", slug: short(a), title: s.title, num: chapter, part: part, lines: s.lines, shift: 1}
		anchors[a] = d.path()
		collectAnchors(d, anchor, anchors)
		docs = append(docs, d)
	}

	for _, kind := range []string{"manual", "tutorials", "reference"} {
		dir := filepath.Join(webDir, "docs", kind)
		old, _ := filepath.Glob(filepath.Join(dir, "*.md"))
		for _, f := range old {
			os.Remove(f)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatal(err)
		}
	}
	for _, d := range docs {
		if err := os.WriteFile(filepath.Join(webDir, "docs", d.kind, d.slug+".md"), []byte(render(d, anchors)), 0o644); err != nil {
			log.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(webDir, "manual.txt"), src, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d pages\n", len(docs))
}

func short(a string) string {
	if s, ok := shortSlugs[a]; ok {
		return s
	}
	return a
}

// splitSections cuts the manual at "## " headings outside code fences.
func splitSections(lines []string) []section {
	var out []section
	cur := section{}
	inFence := ""
	for _, l := range lines {
		inFence = fence(l, inFence)
		if inFence == "" && strings.HasPrefix(l, "## ") {
			out = append(out, cur)
			cur = section{title: strings.TrimSpace(l[3:]), level: 2}
			continue
		}
		cur.lines = append(cur.lines, l)
	}
	return append(out, cur)
}

// splitSub cuts a section's body at "### " headings, dropping any intro.
func splitSub(lines []string) []section {
	var out []section
	var cur *section
	inFence := ""
	for _, l := range lines {
		was := inFence
		inFence = fence(l, inFence)
		if was == "" && inFence == "" && strings.HasPrefix(l, "### ") {
			out = append(out, section{title: strings.TrimSpace(l[4:]), level: 3})
			cur = &out[len(out)-1]
			continue
		}
		if cur != nil {
			cur.lines = append(cur.lines, l)
		}
	}
	return out
}

// fence tracks ``` and ```` code fences: it returns the open fence after l.
func fence(l, open string) string {
	t := strings.TrimSpace(l)
	if open != "" {
		if strings.HasPrefix(t, open) && strings.Trim(t, "`") == "" {
			return ""
		}
		return open
	}
	if strings.HasPrefix(t, "```") {
		n := len(t) - len(strings.TrimLeft(t, "`"))
		return strings.Repeat("`", n)
	}
	return ""
}

// collectAnchors records the anchors of a doc's own headings, and its
// table of contents (its top-level headings once demoted).
func collectAnchors(d *doc, anchor func(string) string, anchors map[string]string) {
	inFence := ""
	for i, l := range d.lines {
		was := inFence
		inFence = fence(l, inFence)
		if was != "" || inFence != "" || !strings.HasPrefix(l, "#") {
			continue
		}
		level := len(l) - len(strings.TrimLeft(l, "#"))
		if level < 2 || l[level] != ' ' {
			continue
		}
		title := strings.TrimSpace(l[level:])
		a := anchor(title)
		anchors[a] = d.path() + "#" + a
		newLevel := level - d.shift
		if newLevel < 2 {
			newLevel = 2
		}
		d.lines[i] = fmt.Sprintf("<a id=\"%s\"></a>\n\n%s %s", a, strings.Repeat("#", newLevel), title)
		if newLevel == 2 {
			d.toc = append(d.toc, [2]string{a, title})
		}
	}
}

var linkRE = regexp.MustCompile(`\]\(#([a-z0-9-]+)\)`)

func render(d *doc, anchors map[string]string) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %q\n", d.title)
	if s, ok := shortTitles[d.title]; ok {
		fmt.Fprintf(&b, "short: %q\n", s)
	}
	fmt.Fprintf(&b, "kind: %s\n", d.kind)
	fmt.Fprintf(&b, "num: %d\n", d.num)
	fmt.Fprintf(&b, "order: %d\n", d.num)
	if d.part != "" {
		fmt.Fprintf(&b, "part: %q\n", d.part)
	}
	if as := aliases[d.path()]; len(as) > 0 {
		fmt.Fprintf(&b, "aliases: [%s]\n", strings.Join(as, ", "))
	}
	if len(d.toc) > 0 {
		b.WriteString("toc:\n")
		for _, t := range d.toc {
			fmt.Fprintf(&b, "  - [%q, %q]\n", t[0], t[1])
		}
	}

	body := strings.Join(d.lines, "\n")
	var uses []string
	body = linkRE.ReplaceAllStringFunc(body, func(m string) string {
		a := linkRE.FindStringSubmatch(m)[1]
		href, ok := anchors[a]
		if !ok {
			log.Printf("%s: no heading for #%s", d.path(), a)
			return m
		}
		page, frag, _ := strings.Cut(href, "#")
		if page == d.path() {
			return "](#" + frag + ")"
		}
		if strings.HasPrefix(page, "/docs/manual/") && !contains(uses, page) {
			uses = append(uses, page)
		}
		return "](" + href + ")"
	})
	if d.kind == "tutorials" && len(uses) > 0 {
		sort.Strings(uses)
		fmt.Fprintf(&b, "uses: [%s]\n", strings.Join(uses, ", "))
	}
	b.WriteString("---\n\n")

	body = tables(body)
	// Markdown runs through the template engine first, where [[ starts an
	// action, so a literal [[ has to be written as one.
	body = strings.ReplaceAll(body, "[[", `[[ "[[" ]]`)
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n")
	return b.String()
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if x == y {
			return true
		}
	}
	return false
}

// tables turns GitHub-style pipe tables outside code fences into HTML.
func tables(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	inFence := ""
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		was := inFence
		inFence = fence(l, inFence)
		if was == "" && inFence == "" && isRow(l) && i+1 < len(lines) && isDivider(lines[i+1]) {
			head := cells(l)
			var rows [][]string
			j := i + 2
			for ; j < len(lines) && isRow(lines[j]); j++ {
				rows = append(rows, cells(lines[j]))
			}
			var t strings.Builder
			t.WriteString("<div class=\"tw\"><table>\n<thead><tr>")
			for _, h := range head {
				t.WriteString("<th>" + inline(h) + "</th>")
			}
			t.WriteString("</tr></thead>\n<tbody>\n")
			for _, r := range rows {
				t.WriteString("<tr>")
				for k, c := range r {
					cls := ""
					if k > 0 {
						cls = ` class="prose"`
					}
					t.WriteString("<td" + cls + ">" + inline(c) + "</td>")
				}
				t.WriteString("</tr>\n")
			}
			t.WriteString("</tbody>\n</table></div>")
			out = append(out, t.String())
			i = j - 1
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func isRow(l string) bool {
	t := strings.TrimSpace(l)
	return strings.HasPrefix(t, "|") && strings.HasSuffix(t, "|") && len(t) > 1
}

func isDivider(l string) bool {
	return isRow(l) && strings.Trim(strings.TrimSpace(l), "|-: ") == ""
}

func cells(l string) []string {
	t := strings.TrimSpace(l)
	t = t[1 : len(t)-1]
	// Split on | outside backticks.
	var out []string
	var cur strings.Builder
	inCode := false
	for _, r := range t {
		switch {
		case r == '`':
			inCode = !inCode
			cur.WriteRune(r)
		case r == '|' && !inCode:
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	return append(out, strings.TrimSpace(cur.String()))
}

var (
	codeRE = regexp.MustCompile("`([^`]+)`")
	mdLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	boldRE = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	autoRE = regexp.MustCompile(`&lt;(https?://[^&]+)&gt;`)
)

// inline renders the Markdown that table cells use: code, links and bold.
func inline(s string) string {
	var codes []string
	s = codeRE.ReplaceAllStringFunc(s, func(m string) string {
		codes = append(codes, "<code>"+html.EscapeString(m[1:len(m)-1])+"</code>")
		return fmt.Sprintf("\x00%d\x00", len(codes)-1)
	})
	s = html.EscapeString(s)
	s = autoRE.ReplaceAllString(s, `<a href="$1">$1</a>`)
	s = boldRE.ReplaceAllString(s, "<b>$1</b>")
	s = mdLink.ReplaceAllString(s, `<a href="$2">$1</a>`)
	for i, c := range codes {
		s = strings.Replace(s, fmt.Sprintf("\x00%d\x00", i), c, 1)
	}
	return s
}

// slugify makes a GitHub-style heading anchor.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}
