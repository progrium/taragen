# Design process

Changes to taragen's design go through proposals: one Markdown file per change in this directory, with frontmatter that `cmd/design` validates and summarizes.
The [spec](../spec.md) describes taragen as built. Proposals say what changes and why; the spec says what exists.
The order of work comes from the dependency graph between proposals, not from milestones.

This README is how taragen applies the workflow. It was adopted by the [design-process](design-process.md) proposal, and changes to it go through proposals too.

## Principles

- **Facts are stored, state is computed.** Frontmatter records only facts (status, dependencies, branches, issue number). Whether a proposal is ready, in progress, or stalled is derived from those facts and from git.
- **Proposals are permanent and named by slug.** A slug never changes and is the only way a proposal is referenced: in `depends`, test tags, links, commits, and PRs.
- **Bodies describe intent, not progress.** Progress lives in status, `branch`, Done criteria checkboxes, and dated Updates notes.
- **Decisions stay findable.** Declined and superseded proposals stay here with their reasons.
- **A person stands behind each design.** A proposal or the spec is signed by a named person who directed and reviewed it. Automation and AI agents can draft, but never sign.
- **CI enforces the rules.** `check` runs on every PR, and the status index is regenerated on main after each merge.

## Files

| File | Holds | Who edits it |
| --- | --- | --- |
| [`../spec.md`](../spec.md) | taragen as built: architecture, invariants, decisions | The PR that finishes a proposal |
| `README.md` (this file) | The workflow as taragen applies it | A proposal that changes the process |
| [`TEMPLATE.md`](TEMPLATE.md) | The sections a new proposal starts from | A proposal that changes the process |
| `<slug>.md` | One proposal: what changes, why, and how it's proven | Its authors, under the rules below |
| `STATUS.md` | Generated table and dependency graph | Only the status job on main |
| [`SIGNERS`](SIGNERS) | Who may sign, as `<login> <Full Name>` per line | Only the person each line names |
| [`../../AGENTS.md`](../../AGENTS.md) | The rules AI agents follow | A proposal that changes the process |

## When a proposal is needed

Write one before changing any of taragen's public surfaces:

- **CLI:** commands, aliases, flags, defaults (port `8088`, output dir `_public`), and exit behavior.
- **Site conventions:** which files are pages, the `.` and `_` prefix rules, special files (`_layout`, `_globals`, `_defaults`, partials), and how source paths map to URLs and output files.
- **Page data:** keys with built-in meaning (`layout`, `date`, `draft`, `contentType`, `preamble`, ...), draft rules, and ordering.
- **JSX runtime:** built-in globals (`page`, `data`, `partial`, `pages`), accepted child and attribute values, and how elements serialize to HTML.
- **Templates and Markdown:** delimiters, helper functions, frontmatter rules, and Markdown extensions and options.
- **Serve mode:** HTTP behavior and the live reload protocol.
- **New major components:** a new page format, an asset pipeline, plugins, or a new dependency that changes how sites are built.
- **Distribution:** supported platforms and install channels.
- **This process:** this README, the template, `cmd/design`, the design CI jobs, and `AGENTS.md`.

Bug fixes, workarounds, performance work that doesn't change output, internal refactors, and changes to the example site in `web/` don't need one; put their reasoning in code comments and commit messages.
A bug fix that changes documented behavior (for example, copying `_assets/` in builds) does need one. When in doubt, draft one.

## Proposals

A proposal is `docs/design/<slug>.md`: frontmatter, then the sections in [TEMPLATE.md](TEMPLATE.md).

### Slugs

Lowercase words of letters and digits joined by dashes, starting with a letter, unique, and never changed. Proposals are referred to only by slug, never by issue number.

### Frontmatter

```yaml
---
issue: 42
slug: example-feature
title: Example feature
status: accepted
created: 2026-11-03
authors: [progrium]
depends: [other-feature, base-change]
areas: [cli, jsx]
branch: [example-feature]
superseded-by:
---
```

| Field | Required | Meaning |
| --- | --- | --- |
| `issue` | no | The GitHub issue number, once one is filed |
| `slug` | always | Its permanent name, and its file's |
| `title` | always | Also the issue title, after `proposal: ` |
| `status` | always | One of the statuses below |
| `created` | always | `YYYY-MM-DD` |
| `authors` | always | GitHub logins |
| `depends` | always, may be `[]` | Proposals, by slug, that must be implemented first |
| `areas` | no | Free-form tags for filtering, such as `cli`, `jsx`, `serve` |
| `branch` | once work starts; `[main]` once implemented | The git branches where its work lives |
| `superseded-by` | when superseded | The replacing proposal's slug |

The syntax is a strict subset of YAML: one `key: value` per line, scalars or flow lists (`[a, b]`), no blank lines, comments, nesting, quoting, or multi-line values.
A value can't contain `: ` or ` #`, start with a YAML indicator (``-?:,[]{}#&*!|>'"%@` ``), or be a word YAML reads as a boolean or null (`yes`, `no`, `true`, `null`, ...).
`check` rejects anything a full YAML parser could read differently.

### Sections

1. **Title and signature.** `# Proposal: <title>`, then the signature line once a person has signed.
2. **Abstract.** A short summary; it becomes the issue body.
3. **Background.** The problem, and what exists today.
4. **Proposal.** A precise statement of the change. What changes, never what's built so far.
5. **Rationale.** Alternatives and trade-offs, including rejected ones.
6. **Compatibility.** Effects on the public surfaces listed above.
7. **Security.** What newly becomes reachable, and how that's covered. "None" with a reason is fine.
8. **Done criteria.** A checklist of the demo commands and test cases, by name, that prove it.
9. **Implementation.** The steps, in order.
10. **Open issues.** Unanswered questions; deleted when there are none.
11. **Updates** or **Declined.** Added once decided: dated notes, or the reason for declining.

Write one line per paragraph, so diffs stay line-oriented. A proposal whose parts finish at different times is split into one proposal per part, linked by `depends`.

### Done criteria

Each top-level item is a checkbox: `- [ ]` not done, `- [x]` done, or `- [~]` waived with the reason after the item (`- [~] Passes on Windows. Waived: CI has no Windows runner.`).
Items can have indented notes or sub-items. Checking or waiving an item is the one edit allowed to an accepted proposal's body.

### Statuses

| Status | Meaning |
| --- | --- |
| `draft` | Being written. No issue needed. |
| `proposed` | Filed as an issue for feedback. Optional. |
| `accepted` | Decided and ready, but nobody is working on it now. Optional. |
| `implemented` | On main, every Done criteria item checked or waived, and the spec describes it. |
| `declined` | Decided against; has a Declined section with the reason. |
| `hold` | Paused, with the reason on the issue or in an Updates note. |
| `superseded` | Replaced by the proposal in `superseded-by`. |

A draft can move to any other status, and a proposed proposal to any but `draft`. A proposal can go from `draft` straight to `implemented` when a person directed the work and it's done. Status changes ride in the PR that makes them.

Computed states, never stored:

- **ready:** accepted, no branch, and every dependency implemented.
- **in progress:** not yet decided (not implemented, declined, or superseded), with a branch that had a commit in the last 7 days.
- **stalled:** not yet decided, with branches that haven't, or that no longer exist.

Computed states read `refs/remotes/origin/*` and local branches, so run `git fetch origin` before trusting them.

### Signatures

A doc is signed with a line under its title: `Nov 11, 2026 · Full Name, h.d.a.`
**h.d.a.** means *human-directed, AI-assisted*: work substantially directed by the named person, who reviewed it closely enough to defend it and takes responsibility for it, however much AI wrote. A line without `, h.d.a.` is also accepted, for work written by hand.
The spec is signed the same way by whoever last reviewed it as a whole.

Only a deliberate act by the named person adds a signature: a `/sign <slug>` or `/sign spec` comment on a PR, `go run ./cmd/design sign <slug>` at a terminal, or typing the line by hand.
Re-sign after significant changes. Only people in [SIGNERS](SIGNERS) can sign, and each person adds their own line there. Agents never sign, run `sign`, post `/sign`, or edit `SIGNERS`. An unsigned proposal is unlikely to be accepted.

## Flow

1. **Draft.** `go run ./cmd/design new <slug> --title "..."` copies the template with frontmatter filled in. A spike for a draft is fine; record its branch in `branch`.
2. **Optionally propose.** `go run ./cmd/design propose <slug>` files the issue, labeled `Proposal`, and sets `status: proposed` in one commit. Discussion happens on the issue; the doc is revised in commits.
3. **Sign.** The person who directed the work reviews the whole doc and signs it.
4. **Optionally accept.** If it's decided but nobody is starting it yet, set `status: accepted`.
5. **Build on a feature branch** named after the slug, listed in `branch`. Each PR updates Done criteria, `depends` if a dependency turns up, and dated Updates notes.
6. **Finish.** The PR that completes the proposal updates the spec, checks off or waives every Done criteria item, and sets `status: implemented` and `branch: [main]` in its last commit, so a squash merge carries both.
7. **After merge.** The status job regenerates `STATUS.md` and closes the proposal's issue.

`implemented` requires exactly `branch: [main]`, and `main` is allowed only on an implemented proposal.

### After acceptance

An accepted proposal's body isn't edited except to check or waive Done criteria. A refinement gets a dated Updates note; a change in design gets a new proposal that supersedes the old one or is added to its `depends`.
The spec never describes unbuilt features, and proposals aren't updated to describe the current system.

### Declining and superseding

To decline, set `status: declined` and add a dated `## Declined` section saying why, linking the discussion, and naming what would change the answer. To reconsider, write a new proposal that answers it. A draft nobody needs can be deleted.
To supersede, set `status: superseded` and `superseded-by: <new-slug>`.

### Issues

An issue is optional and exists only for feedback; the file is the source of truth.
`propose` asks before filing (it acts as your `gh` account), refuses if an open issue already has the title, and takes `--dry-run`, `--yes`, and `--issue N` to adopt an existing issue.
After each merge to main, `close-issues` closes the open issue of each implemented proposal as completed, and of each declined or superseded one as not planned, with a comment linking the doc. It leaves closed issues alone.

## Tests tied to proposals

Go test cases are tagged with the proposals they prove by ending the subtest name with their slugs in brackets, written exactly `[a, b]`:

```go
proposaltest.Run(t, "renders fragments [baseline]", func(t *testing.T) { ... })
```

- A tagged case whose proposal isn't implemented is skipped, unless `DESIGN_PENDING=1` is set. It starts running in the commit that marks the proposal implemented.
- Untagged cases always run.
- `go run ./cmd/design test <slug>` runs one proposal's tagged cases whatever its status (and, since Go can't filter them out, every top-level test that has no subtests).
- `check` validates the tags in `_test.go` files, on lines that call `.Run(` or set a `name:` field: the format, and that each slug exists.

Demos are written as tagged test cases, so there is no separate demo registry.

## Tooling

`cmd/design` implements the workflow with only the Go toolchain, git, and the `gh` CLI for the commands that touch issues. Run `go run ./cmd/design` for usage.

| Command | Does |
| --- | --- |
| `list [--status S] [--area A]` | Table of slug, issue, status, computed state, and title |
| `ready` | Accepted proposals with no branch and every dependency implemented |
| `graph` | The dependency graph, as Mermaid |
| `status` | Rewrites `STATUS.md`; run on main only |
| `new <slug> [--title T] [--author LOGIN]` | Starts a draft from the template |
| `propose <slug> [--yes] [--dry-run] [--issue N]` | Files its issue and makes it proposed |
| `log <slug>` | Its status and branch changes, with their commits |
| `sign <slug or spec>` | Signs a doc you've reviewed; interactive, asks you to type the slug |
| `close-issues [--dry-run]` | Closes issues of implemented, declined, and superseded proposals |
| `check` | Validates proposals, test tags, signatures, and links |
| `test <slug>` | Runs one proposal's tagged test cases |

`STATUS.md` is generated, never hand-edited, and never committed in PRs, so two PRs that change statuses never conflict over it. Use `list` and `ready` for the current state anywhere.

### What check enforces

`check` fails on: unparseable frontmatter, unknown keys, missing required fields, unknown statuses, bad dates; a file not named for its slug, duplicate slugs or issue numbers, a slug starting with a digit without an issue; references by number or to unknown slugs in `depends`, `superseded-by`, or test tags; test tags not written `[a, b]`; links to files under `docs/design/` that don't exist, from any file; dependency cycles; an implemented proposal depending on an unimplemented one; an accepted proposal depending on a declined one; `proposed` without `issue`; `declined` without a Declined section; `superseded` without `superseded-by`; `implemented` without `branch: [main]`, or `main` on anything else; Done criteria items that aren't checkboxes, or open items on an implemented proposal; and signature lines in neither form or naming someone not in `SIGNERS`.

It warns when a proposed or accepted proposal has every Done criteria item checked or waived, and when a branch in `branch` can't be found.

It skips `vendor/` and `testdata/` directories when scanning for tags and links. Each rule has a fixture under `internal/design/testdata/check/`.

## CI

| Workflow | Runs |
| --- | --- |
| `design-check.yml` | `check` on every PR (and by hand), with every branch tip fetched; a new push cancels the previous run. |
| `test.yml` | `go test ./...` on PRs, except from `claude/` branches, which get only the design check. Run it by hand on any branch. |
| `design-status.yml` | On pushes to main that touch `docs/design/` or the tool, daily, and by hand: `status`, then `close-issues`, committing `STATUS.md` as a bot if it changed. |
| `design-sign.yml` | On a PR comment whose first line is `/sign <slug>` or `/sign spec`: signs as the commenter if they're in main's `SIGNERS`, for same-repository PRs only, and replies with the result. |

Commits pushed by CI with the default token don't trigger other workflows, so a PR head after a `/sign` commit has no check results until the next push. Keep that in mind before making the check a required status.
A PR that says "closes #N" closes the issue on merge before `close-issues` runs; that's harmless, but the doc-link comment is lost.

## Local choices

- Stalled after **7 days** without a commit.
- Branches starting with **`claude/`** are agent branches and skip the full test suite.
- The signature abbreviation is **h.d.a.**
- `close-issues` doesn't sync labels from status.
