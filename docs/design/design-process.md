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

Adopt a proposal-driven design process: one Markdown proposal per design change in `docs/design/`, a spec describing the system as built, a small in-repo tool (`tools/design`) that validates and summarizes proposals, and CI that enforces the rules and keeps a generated status index.

## Background

taragen's design decisions lived only in code and commit messages. There was no record of which behavior is intended, what's planned, or why alternatives were rejected, and nothing for AI agents working in the repository to follow.
The [baseline](baseline.md) proposal and the [spec](../spec.md) capture the design as it stands; this proposal adds the process for changing it.

## Proposal

Add the process described in the [design README](README.md):

- `docs/spec.md` describes taragen as built, and is signed by whoever last reviewed it as a whole.
- `docs/design/` holds the process README, `TEMPLATE.md`, `SIGNERS`, one `<slug>.md` per proposal, and a generated `STATUS.md`.
- Proposal frontmatter is a strict YAML subset with the keys, statuses, and computed states the README defines.
- `tools/design` provides `list`, `ready`, `graph`, `status`, `new`, `propose`, `log`, `sign`, `close-issues`, `check`, and `test`, using only Go, git, and `gh`.
- `internal/proposaltest` tags Go test cases with proposals (`name [a, b]`) and skips cases for unimplemented proposals unless `DESIGN_PENDING=1`.
- Four workflows: `design-check.yml`, `test.yml`, `design-status.yml`, and `design-sign.yml`.
- `AGENTS.md` restates the rules for AI agents.

## Rationale

Keeping the tool in the repository, in Go, means a PR that changes a rule changes the checker in the same diff, and nobody installs anything beyond the toolchain taragen already needs.
A strict YAML subset instead of a YAML library keeps the parser small and makes `check` reject anything another parser could read differently.
Storing only facts and computing state from git avoids an "in progress" status that goes stale.
Generating `STATUS.md` only on main avoids merge conflicts between PRs that change statuses.
Milestones and a separate changelog were rejected: the dependency graph orders the work, and `log <slug>` reconstructs history from git.

## Compatibility

None for taragen users: no change to the CLI, site conventions, or output.
For contributors, design changes now need a proposal, as listed in the README, and test cases can be tagged.

## Security

The `/sign` workflow can push commits to PR branches. It acts only on PRs from branches in this repository, ignores comments from bots, and reads `SIGNERS` from main, never from the PR, so a PR can't add its own signer. It commits only the signed file.
The status workflow can write to `main` (only `STATUS.md`) and close issues.

## Done criteria

- [x] `go run ./tools/design check` passes on the repository.
- [x] Each `check` rule has a fixture under `internal/design/testdata/check/`, run by `go test ./internal/design`.
- [x] `list`, `ready`, `graph`, `status`, `new`, and `log` work on the repository.
- [x] `propose --dry-run` and `close-issues --dry-run` show what they'd do without changing anything.
- [x] `sign` refuses without a terminal unless `--login` and `--date` are given, and refuses signers not in `SIGNERS`.
- [x] Tagged cases for an unimplemented proposal skip by default and run with `DESIGN_PENDING=1`.
- [x] The four workflows and `AGENTS.md` are in place.

## Implementation

1. Write the spec and the baseline proposal.
2. Add the README, template, and signers list.
3. Build `tools/design` and `internal/design`, with a fixture per check rule.
4. Add `internal/proposaltest` and tag the existing tests.
5. Add the CI workflows and `AGENTS.md`.
6. Create the `Proposal` label on GitHub, and add signers.
