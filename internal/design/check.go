package design

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Diag is one problem found by Check.
type Diag struct {
	Path string
	Line int
	Msg  string
	Warn bool
}

func (d Diag) String() string {
	kind := "error"
	if d.Warn {
		kind = "warning"
	}
	loc := d.Path
	if d.Line > 0 {
		loc += ":" + strconv.Itoa(d.Line)
	}
	return fmt.Sprintf("%s: %s: %s", kind, loc, d.Msg)
}

// SkipPrefixes are paths check never scans for test tags or links: vendored
// code, and the checker's own fixtures, which are broken on purpose.
var SkipPrefixes = []string{"vendor/"}

func skipped(file string) bool {
	for _, p := range SkipPrefixes {
		if strings.HasPrefix(file, p) {
			return true
		}
	}
	return strings.HasPrefix(file, "testdata/") || strings.Contains(file, "/testdata/")
}

var (
	slugRe    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	numRefRe  = regexp.MustCompile(`^#?[0-9]+$`)
	dateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	issueRe   = regexp.MustCompile(`^[1-9][0-9]*$`)
	declinedH = "Declined"
)

// GrandfatheredSlugs may start with a digit without an issue.
var GrandfatheredSlugs = map[string]bool{}

type checker struct {
	r      *Repo
	diags  []Diag
	linked map[string]bool
}

func (c *checker) errorf(p string, line int, format string, args ...any) {
	c.diags = append(c.diags, Diag{Path: p, Line: line, Msg: fmt.Sprintf(format, args...)})
}

func (c *checker) warnf(p string, line int, format string, args ...any) {
	c.diags = append(c.diags, Diag{Path: p, Line: line, Msg: fmt.Sprintf(format, args...), Warn: true})
}

// Check validates every proposal, test tag, signature, and link to the
// design directory. Diagnostics are sorted by path and line.
func Check(r *Repo) []Diag {
	c := &checker{r: r, linked: map[string]bool{}}
	c.diags = append(c.diags, r.SignerErr...)

	slugs := map[string]string{}
	issues := map[int]string{}
	for _, p := range r.Proposals {
		c.frontmatter(p)
		c.naming(p, slugs, issues)
		c.references(p)
		c.status(p)
		c.done(p)
		c.signature(p.Path, p.Doc)
	}
	if r.Spec != nil {
		c.signature(SpecPath, r.Spec)
	}
	c.graph()
	if err := c.files(); err != nil {
		c.errorf(".", 0, "listing files: %v", err)
	}

	sort.SliceStable(c.diags, func(i, j int) bool {
		a, b := c.diags[i], c.diags[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Line < b.Line
	})
	return c.diags
}

// HasErrors reports whether any diagnostic is an error.
func HasErrors(diags []Diag) bool {
	for _, d := range diags {
		if !d.Warn {
			return true
		}
	}
	return false
}

func (c *checker) frontmatter(p *Proposal) {
	for _, e := range p.FMErrs {
		c.errorf(p.Path, e.Line, "%s", e.Msg)
	}
	if p.FM == nil {
		return
	}
	for _, f := range p.FM.Fields {
		if hint, ok := RemovedKeys[f.Key]; ok {
			c.errorf(p.Path, f.Line, "removed key %q: %s", f.Key, hint)
			continue
		}
		isList, known := Keys[f.Key]
		if !known {
			c.errorf(p.Path, f.Line, "unknown key %q", f.Key)
			continue
		}
		if isList && !f.IsList && f.Scalar != "" {
			c.errorf(p.Path, f.Line, "%s must be a list, like [a, b]", f.Key)
		}
		if !isList && f.IsList {
			c.errorf(p.Path, f.Line, "%s must be a single value, not a list", f.Key)
		}
	}
	badLine := map[string]bool{}
	for _, e := range p.FMErrs {
		badLine[e.Key] = true
	}
	for _, k := range RequiredKeys {
		f := p.FM.Get(k)
		switch {
		case f == nil && badLine[k]:
			// already reported
		case f == nil:
			c.errorf(p.Path, 1, "missing required key %q", k)
		case k == "depends":
			// may be []
		case (f.IsList && len(f.List) == 0) || (!f.IsList && f.Scalar == ""):
			c.errorf(p.Path, f.Line, "required key %q is empty", k)
		}
	}
	if f := p.FM.Get("status"); f != nil && f.Scalar != "" && !validStatus(f.Scalar) {
		c.errorf(p.Path, f.Line, "unknown status %q (want one of %s)", f.Scalar, strings.Join(Statuses, ", "))
	}
	if f := p.FM.Get("created"); f != nil && f.Scalar != "" {
		if _, err := time.Parse("2006-01-02", f.Scalar); err != nil || !dateRe.MatchString(f.Scalar) {
			c.errorf(p.Path, f.Line, "created %q is not a YYYY-MM-DD date", f.Scalar)
		}
	}
	if f := p.FM.Get("issue"); f != nil && f.Scalar != "" && !issueRe.MatchString(f.Scalar) {
		c.errorf(p.Path, f.Line, "issue %q is not a number", f.Scalar)
	}
}

func (c *checker) line(p *Proposal, key string) int {
	if f := p.FM.Get(key); f != nil {
		return f.Line
	}
	return 1
}

func (c *checker) naming(p *Proposal, slugs map[string]string, issues map[int]string) {
	if p.Slug == "" {
		return
	}
	if !slugRe.MatchString(p.Slug) {
		c.errorf(p.Path, c.line(p, "slug"), "slug %q must be lowercase letters and digits joined by dashes", p.Slug)
	} else if p.Slug[0] >= '0' && p.Slug[0] <= '9' && p.Issue == 0 && !GrandfatheredSlugs[p.Slug] {
		c.errorf(p.Path, c.line(p, "slug"), "slug %q must start with a letter", p.Slug)
	}
	if p.FileSlug() != p.Slug {
		c.errorf(p.Path, c.line(p, "slug"), "file must be named %s.md for slug %q", p.Slug, p.Slug)
	}
	if other, dup := slugs[p.Slug]; dup {
		c.errorf(p.Path, c.line(p, "slug"), "duplicate slug %q (also in %s)", p.Slug, other)
	} else {
		slugs[p.Slug] = p.Path
	}
	if p.Issue != 0 {
		if other, dup := issues[p.Issue]; dup {
			c.errorf(p.Path, c.line(p, "issue"), "duplicate issue %d (also in %s)", p.Issue, other)
		} else {
			issues[p.Issue] = p.Path
		}
	}
}

// ref checks one reference to a proposal by slug.
func (c *checker) ref(file string, line int, what, ref string) {
	if numRefRe.MatchString(ref) {
		c.errorf(file, line, "%s refers to %q by number; refer to proposals by slug", what, ref)
		return
	}
	if c.r.BySlug[ref] == nil {
		c.errorf(file, line, "%s refers to unknown proposal %q", what, ref)
	}
}

func (c *checker) references(p *Proposal) {
	for _, d := range p.Depends {
		c.ref(p.Path, c.line(p, "depends"), "depends", d)
		if d == p.Slug {
			c.errorf(p.Path, c.line(p, "depends"), "proposal depends on itself")
		}
	}
	if p.SupersededBy != "" {
		c.ref(p.Path, c.line(p, "superseded-by"), "superseded-by", p.SupersededBy)
	}
}

func (c *checker) status(p *Proposal) {
	switch p.Status {
	case "proposed":
		if p.Issue == 0 {
			c.errorf(p.Path, c.line(p, "status"), "proposed proposal has no issue")
		}
	case "declined":
		if p.Doc.Section(declinedH) == nil {
			c.errorf(p.Path, c.line(p, "status"), "declined proposal has no \"## Declined\" section")
		}
	case "superseded":
		if p.SupersededBy == "" {
			c.errorf(p.Path, c.line(p, "status"), "superseded proposal has no superseded-by")
		}
	}
	hasMain := false
	for _, b := range p.Branch {
		if b == "main" {
			hasMain = true
		}
	}
	if p.Implemented() {
		if len(p.Branch) != 1 || p.Branch[0] != "main" {
			c.errorf(p.Path, c.line(p, "branch"), "implemented proposal must have branch: [main]")
		}
	} else if hasMain {
		c.errorf(p.Path, c.line(p, "branch"), "branch lists main, but the proposal isn't implemented")
	}
	if !p.Implemented() && c.r.Git != nil {
		for _, b := range p.Branch {
			if _, ok := c.r.Git.BranchTime(b); !ok {
				c.warnf(p.Path, c.line(p, "branch"), "branch %q not found (fetch every branch tip to check)", b)
			}
		}
	}
}

func (c *checker) done(p *Proposal) {
	items, ok := p.Doc.DoneItems()
	if !ok {
		return
	}
	open := 0
	for _, it := range items {
		switch it.Mark {
		case "":
			c.errorf(p.Path, it.Line, "Done criteria item is not a checkbox (- [ ], - [x], or - [~])")
		case " ":
			open++
			if p.Implemented() {
				c.errorf(p.Path, it.Line, "implemented proposal has an open Done criteria item")
			}
		}
	}
	if (p.Status == "proposed" || p.Status == "accepted") && len(items) > 0 && open == 0 {
		c.warnf(p.Path, c.line(p, "status"), "every Done criteria item is checked or waived; is it implemented?")
	}
}

func (c *checker) signature(file string, d *Doc) {
	if d.Signature == "" {
		return
	}
	sig, err := ParseSignature(d.Signature)
	if err != nil {
		c.errorf(file, d.SignatureLine, "%v", err)
		return
	}
	if _, ok := c.r.SignerByName(sig.Name); !ok {
		c.errorf(file, d.SignatureLine, "signed by %q, who is not in %s", sig.Name, SignersPath)
	}
}

func (c *checker) graph() {
	// cycles
	const (
		unvisited = iota
		visiting
		done
	)
	mark := map[string]int{}
	var stack []string
	reported := map[string]bool{}
	var visit func(p *Proposal)
	visit = func(p *Proposal) {
		mark[p.Slug] = visiting
		stack = append(stack, p.Slug)
		for _, d := range p.Depends {
			dep := c.r.BySlug[d]
			if dep == nil || d == p.Slug {
				continue
			}
			switch mark[d] {
			case visiting:
				i := len(stack) - 1
				for stack[i] != d {
					i--
				}
				cycle := append(append([]string{}, stack[i:]...), d)
				if !reported[d] {
					reported[d] = true
					c.errorf(dep.Path, c.line(dep, "depends"), "dependency cycle: %s", strings.Join(cycle, " -> "))
				}
			case unvisited:
				visit(dep)
			}
		}
		stack = stack[:len(stack)-1]
		mark[p.Slug] = done
	}
	for _, p := range c.r.Proposals {
		if p.Slug != "" && c.r.BySlug[p.Slug] == p && mark[p.Slug] == unvisited {
			visit(p)
		}
	}

	for _, p := range c.r.Proposals {
		for _, d := range p.Depends {
			dep := c.r.BySlug[d]
			if dep == nil {
				continue
			}
			if p.Implemented() && !dep.Implemented() {
				c.errorf(p.Path, c.line(p, "depends"), "implemented proposal depends on %q, which is %s", d, dep.Status)
			}
			if p.Status == "accepted" && dep.Status == "declined" {
				c.errorf(p.Path, c.line(p, "depends"), "accepted proposal depends on %q, which was declined", d)
			}
		}
	}
}

var (
	// a test name ending in " [a, b]" tags the case with proposals; names
	// are looked for on lines that run a subtest or set a table's name field
	tagLineRe = regexp.MustCompile(`\.Run\(|\bname:\s`)
	tagCandRe = regexp.MustCompile(`\s\[([a-z0-9 ,#-]+)\]["` + "`" + `]`)
	tagRe     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*(, [a-z0-9]+(-[a-z0-9]+)*)*$`)
	// literal paths into the design directory, from any file
	designPathRe = regexp.MustCompile(regexp.QuoteMeta(DesignDir) + `/[A-Za-z0-9_./-]*[A-Za-z0-9_-]\.md`)
	// relative Markdown links
	mdLinkRe = regexp.MustCompile(`\]\(([^)\s#]+)(#[^)\s]*)?\)`)
)

// generated files may not exist yet in a PR checkout
var generated = map[string]bool{IndexPath: true}

func (c *checker) files() error {
	files, err := TrackedFiles(c.r.Root)
	if err != nil {
		return err
	}
	for _, f := range files {
		if skipped(f) {
			continue
		}
		isTest := strings.HasSuffix(f, "_test.go")
		isMD := strings.HasSuffix(f, ".md")
		b, err := os.ReadFile(filepath.Join(c.r.Root, f))
		if err != nil || strings.ContainsRune(string(b[:min(len(b), 8000)]), 0) {
			continue
		}
		for i, line := range strings.Split(string(b), "\n") {
			n := i + 1
			if isTest && tagLineRe.MatchString(line) {
				for _, m := range tagCandRe.FindAllStringSubmatch(line, -1) {
					c.tag(f, n, m[1])
				}
			}
			for _, m := range designPathRe.FindAllString(line, -1) {
				c.link(f, n, path.Clean(m))
			}
			if isMD {
				for _, m := range mdLinkRe.FindAllStringSubmatch(line, -1) {
					target := m[1]
					if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "/") {
						continue
					}
					c.link(f, n, path.Clean(path.Join(path.Dir(f), target)))
				}
			}
		}
	}
	return nil
}

func (c *checker) tag(file string, line int, inner string) {
	if !tagRe.MatchString(inner) {
		c.errorf(file, line, "test tag [%s] must be written exactly as [a, b]", inner)
		return
	}
	for _, s := range strings.Split(inner, ", ") {
		c.ref(file, line, "test tag", s)
	}
}

func (c *checker) link(file string, line int, target string) {
	if !strings.HasPrefix(target, DesignDir+"/") || generated[target] {
		return
	}
	key := fmt.Sprintf("%s:%d:%s", file, line, target)
	if c.linked[key] {
		return
	}
	c.linked[key] = true
	if _, err := os.Stat(filepath.Join(c.r.Root, target)); err != nil {
		c.errorf(file, line, "link to %s, which doesn't exist", target)
	}
}
