// Package proposaltest ties Go test cases to design proposals.
//
// A case is tagged by ending its name with the slugs of the proposals it
// proves, in brackets: "renders fragments [baseline]". Tags are written
// "[a, b]" exactly, since tooling matches them as text. A tagged case whose
// proposals aren't all implemented is skipped, unless DESIGN_PENDING=1 is
// set (as `go run ./cmd/design test <slug>` does). Untagged cases always run.
package proposaltest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/progrium/taragen/internal/design"
)

var tagRe = regexp.MustCompile(` \[([a-z0-9]+(?:-[a-z0-9]+)*(?:, [a-z0-9]+(?:-[a-z0-9]+)*)*)\]$`)

// Tags returns the proposal slugs a test name is tagged with.
func Tags(name string) []string {
	m := tagRe.FindStringSubmatch(name)
	if m == nil {
		return nil
	}
	return strings.Split(m[1], ", ")
}

// Run runs fn as a subtest named name, skipping it if name is tagged with a
// proposal that isn't implemented yet.
func Run(t *testing.T, name string, fn func(t *testing.T)) bool {
	t.Helper()
	return t.Run(name, func(t *testing.T) {
		t.Helper()
		Gate(t, Tags(name)...)
		fn(t)
	})
}

// Gate skips t unless every proposal in slugs is implemented, or
// DESIGN_PENDING=1 is set. An unknown slug fails the test.
func Gate(t *testing.T, slugs ...string) {
	t.Helper()
	if len(slugs) == 0 {
		return
	}
	repo, err := load()
	if err != nil {
		t.Fatalf("proposaltest: %v", err)
	}
	for _, s := range slugs {
		p := repo.BySlug[s]
		if p == nil {
			t.Fatalf("proposaltest: test is tagged with unknown proposal %q", s)
		}
		if !p.Implemented() && os.Getenv("DESIGN_PENDING") != "1" {
			t.Skipf("proposal %s is %s; set DESIGN_PENDING=1 or run `go run ./cmd/design test %s`", s, p.Status, s)
		}
	}
}

var (
	once    sync.Once
	repo    *design.Repo
	loadErr error
)

// load finds the repository root by walking up from the test's working
// directory to the nearest go.mod, and loads its proposals.
func load() (*design.Repo, error) {
	once.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			loadErr = err
			return
		}
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				loadErr = os.ErrNotExist
				return
			}
			dir = parent
		}
		repo, loadErr = design.Load(dir)
	})
	return repo, loadErr
}
