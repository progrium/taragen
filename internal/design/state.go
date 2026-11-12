package design

import (
	"fmt"
	"strings"
	"time"
)

// StaleAfter is how long a proposal branch can go without a commit before
// the proposal counts as stalled.
const StaleAfter = 7 * 24 * time.Hour

// Computed states. These are derived, never stored in frontmatter.
const (
	StateReady      = "ready"
	StateInProgress = "in progress"
	StateStalled    = "stalled"
)

// terminal statuses are decided; their branches don't make them in progress
// or stalled.
var terminal = map[string]bool{"implemented": true, "declined": true, "superseded": true}

// State computes a proposal's state at now: ready, in progress, stalled, or
// "" when none applies. Without git, only ready can be computed.
func (r *Repo) State(p *Proposal, now time.Time) string {
	if p.Status == "accepted" && len(p.Branch) == 0 && r.depsImplemented(p) {
		return StateReady
	}
	if terminal[p.Status] || len(p.Branch) == 0 || r.Git == nil {
		return ""
	}
	for _, b := range p.Branch {
		if t, ok := r.Git.BranchTime(b); ok && now.Sub(t) <= StaleAfter {
			return StateInProgress
		}
	}
	return StateStalled
}

func (r *Repo) depsImplemented(p *Proposal) bool {
	for _, d := range p.Depends {
		dep := r.BySlug[d]
		if dep == nil || !dep.Implemented() {
			return false
		}
	}
	return true
}

// Ready lists proposals that can be picked up now.
func (r *Repo) Ready() []*Proposal {
	var out []*Proposal
	for _, p := range r.Proposals {
		if r.State(p, time.Now()) == StateReady {
			out = append(out, p)
		}
	}
	return out
}

// Graph renders the dependency graph as Mermaid. An edge points from a
// dependency to the proposal that needs it.
func (r *Repo) Graph(now time.Time) string {
	var b strings.Builder
	b.WriteString("graph TD\n")
	id := func(slug string) string { return "p_" + strings.ReplaceAll(slug, "-", "_") }
	for _, p := range r.Proposals {
		label := p.Status
		if s := r.State(p, now); s != "" {
			label += ", " + s
		}
		fmt.Fprintf(&b, "  %s[\"%s<br/><i>%s</i>\"]\n", id(p.Slug), p.Slug, label)
	}
	for _, p := range r.Proposals {
		for _, d := range p.Depends {
			if r.BySlug[d] != nil {
				fmt.Fprintf(&b, "  %s --> %s\n", id(d), id(p.Slug))
			}
		}
	}
	return b.String()
}
