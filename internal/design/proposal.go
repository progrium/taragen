package design

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Paths, relative to the repository root.
const (
	DesignDir    = "design"
	SpecPath     = DesignDir + "/SPEC.md"
	SignersPath  = DesignDir + "/SIGNERS"
	TemplateName = "TEMPLATE.md"
)

// nonProposals are files in the design directory that aren't proposals.
var nonProposals = map[string]bool{
	"README.md":  true,
	"SPEC.md":    true,
	TemplateName: true,
}

// Statuses, in the order they are documented.
var Statuses = []string{"draft", "proposed", "accepted", "implemented", "declined", "hold", "superseded"}

func validStatus(s string) bool {
	for _, st := range Statuses {
		if st == s {
			return true
		}
	}
	return false
}

// Keys lists every known frontmatter key, and whether it holds a list.
var Keys = map[string]bool{
	"issue":         false,
	"slug":          false,
	"title":         false,
	"status":        false,
	"created":       false,
	"authors":       true,
	"depends":       true,
	"areas":         true,
	"branch":        true,
	"superseded-by": false,
}

// KeyOrder is the order keys are written in by new.
var KeyOrder = []string{"issue", "slug", "title", "status", "created", "authors", "depends", "areas", "branch", "superseded-by"}

// RemovedKeys are keys that used to exist and are now rejected, with a hint.
var RemovedKeys = map[string]string{}

// RequiredKeys must be present on every proposal.
var RequiredKeys = []string{"slug", "title", "status", "created", "authors", "depends"}

// Proposal is one parsed proposal file.
type Proposal struct {
	Path string // relative to the repository root, slash-separated
	Src  string

	FM       *Frontmatter
	FMErrs   []FrontmatterError
	Body     string
	BodyLine int

	Slug, Title, Status, Created, SupersededBy string
	Issue                                      int
	Authors, Depends, Areas, Branch            []string

	Doc *Doc
}

// FileSlug is the slug implied by the file name.
func (p *Proposal) FileSlug() string {
	return strings.TrimSuffix(filepath.Base(p.Path), ".md")
}

func (p *Proposal) Implemented() bool { return p.Status == "implemented" }

// Signer is one line of the signers list.
type Signer struct {
	Login, Name string
	Line        int
}

// Repo is a loaded design directory.
type Repo struct {
	Root      string
	Proposals []*Proposal // sorted by created, then issue, then slug
	BySlug    map[string]*Proposal
	Signers   []Signer
	SignerErr []Diag
	Spec      *Doc // nil if there is no spec

	// Git answers questions about branches. It is nil outside a git checkout.
	Git Git
}

// Load reads the design directory, signers list, and spec under root.
// Problems with individual files are reported later by Check; Load fails
// only when the design directory can't be read at all.
func Load(root string) (*Repo, error) {
	r := &Repo{Root: root, BySlug: map[string]*Proposal{}}
	entries, err := os.ReadDir(filepath.Join(root, DesignDir))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || nonProposals[name] {
			continue
		}
		rel := DesignDir + "/" + name
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return nil, err
		}
		p := parseProposal(rel, string(b))
		r.Proposals = append(r.Proposals, p)
		if p.Slug != "" && r.BySlug[p.Slug] == nil {
			r.BySlug[p.Slug] = p
		}
	}
	sortProposals(r.Proposals)

	if err := r.loadSigners(); err != nil {
		return nil, err
	}
	if b, err := os.ReadFile(filepath.Join(root, SpecPath)); err == nil {
		r.Spec = ParseDoc(string(b), 1)
	}
	if isGitRoot(root) {
		r.Git = execGit{dir: root}
	}
	return r, nil
}

func sortProposals(ps []*Proposal) {
	sort.SliceStable(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		if a.Created != b.Created {
			return a.Created < b.Created
		}
		if a.Issue != b.Issue {
			// proposals without an issue sort after those with one
			if a.Issue == 0 || b.Issue == 0 {
				return b.Issue == 0
			}
			return a.Issue < b.Issue
		}
		return a.FileSlug() < b.FileSlug()
	})
}

func parseProposal(rel, src string) *Proposal {
	p := &Proposal{Path: rel, Src: src}
	p.FM, p.Body, p.BodyLine, p.FMErrs = ParseFrontmatter(src)
	scalar := func(k string) string {
		if f := p.FM.Get(k); f != nil && !f.IsList {
			return f.Scalar
		}
		return ""
	}
	list := func(k string) []string {
		if f := p.FM.Get(k); f != nil && f.IsList {
			return f.List
		}
		return nil
	}
	p.Slug = scalar("slug")
	p.Title = scalar("title")
	p.Status = scalar("status")
	p.Created = scalar("created")
	p.SupersededBy = scalar("superseded-by")
	if n, err := strconv.Atoi(scalar("issue")); err == nil {
		p.Issue = n
	}
	p.Authors = list("authors")
	p.Depends = list("depends")
	p.Areas = list("areas")
	p.Branch = list("branch")
	p.Doc = ParseDoc(p.Body, p.BodyLine)
	return p
}

var signerLineRe = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9-]*) +(\S.*\S|\S)$`)

func (r *Repo) loadSigners() error {
	f, err := os.Open(filepath.Join(r.Root, SignersPath))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := signerLineRe.FindStringSubmatch(line)
		if m == nil {
			r.SignerErr = append(r.SignerErr, Diag{Path: SignersPath, Line: n, Msg: fmt.Sprintf("expected \"<login> <Full Name>\", got %q", line)})
			continue
		}
		r.Signers = append(r.Signers, Signer{Login: m[1], Name: m[2], Line: n})
	}
	return sc.Err()
}

// SignerByLogin finds a signer by account login, ignoring case.
func (r *Repo) SignerByLogin(login string) (Signer, bool) {
	for _, s := range r.Signers {
		if strings.EqualFold(s.Login, login) {
			return s, true
		}
	}
	return Signer{}, false
}

// SignerByName finds a signer by full name.
func (r *Repo) SignerByName(name string) (Signer, bool) {
	for _, s := range r.Signers {
		if s.Name == name {
			return s, true
		}
	}
	return Signer{}, false
}
