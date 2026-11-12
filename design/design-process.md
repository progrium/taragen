---
slug: design-process
title: Proposal-driven design process
status: implemented
created: 2026-11-12
authors: [progrium]
depends: [baseline]
areas: [process]
branch: [main]
---

# Proposal: Proposal-driven design process

## Abstract

Adopt a proposal-driven design process: one Markdown proposal per design change in `design/`, a spec describing the system as built, a small in-repo tool (`tools/design`) that validates and summarizes proposals, and a CI check that enforces the rules.

## Background

taragen's design decisions lived only in code and commit messages. There was no record of which behavior is intended, what's planned, or why alternatives were rejected, and nothing for AI agents working in the repository to follow.
The [baseline](baseline.md) proposal and the [spec](SPEC.md) capture the design as it stands; this proposal adds the process for changing it.

## Proposal

Add the process described in the [design README](README.md):

- `design/SPEC.md` describes taragen as built, and is signed by whoever last reviewed it as a whole.
- `design/` holds the process README, `TEMPLATE.md`, `SIGNERS`, and one `<slug>.md` per proposal.
- Proposal frontmatter is a strict YAML subset with the keys, statuses, and computed states the README defines.
- `tools/design` provides `list`, `ready`, `graph`, `new`, `log`, `sign`, and `check`, using only Go and git.
- `design-check.yml` runs `check` on every PR.
- `AGENTS.md` restates the rules for AI agents.

## Rationale

Keeping the tool in the repository, in Go, means a PR that changes a rule changes the checker in the same diff, and nobody installs anything beyond the toolchain taragen already needs.
A strict YAML subset instead of a YAML library keeps the parser small and makes `check` reject anything another parser could read differently.
Storing only facts and computing state from git avoids an "in progress" status that goes stale.
The full workflow also has a generated status index, issue-sync commands, signing by PR comment, and proposal-tagged tests. They were left out to keep the tooling in proportion to taragen, and `list`, `ready`, and `graph` show the same state on demand. Each can be added with its own proposal.
Milestones and a separate changelog were rejected: the dependency graph orders the work, and `log <slug>` reconstructs history from git.

## Compatibility

None for taragen users: no change to the CLI, site conventions, or output.
For contributors, design changes now need a proposal, as listed in the README.

## Security

None: the check job only reads the repository, and `sign` runs only at a person's terminal.

## Done criteria

- [x] `go run ./tools/design check` passes on the repository.
- [x] The `check` rules are covered by the table test in `internal/design/check_test.go`.
- [x] `list`, `ready`, `graph`, `new`, and `log` work on the repository.
- [x] `sign` refuses without a terminal, and refuses people not in `SIGNERS`.
- [x] `design-check.yml` and `AGENTS.md` are in place.

## Implementation

1. Write the spec and the baseline proposal.
2. Add the README, template, and signers list.
3. Build `tools/design` and `internal/design`, with tests for the check rules.
4. Add the check workflow and `AGENTS.md`.
5. Add signers.
