# Agent guidance

Rules for AI agents working in this repository. They restate [the design process](design/README.md); where the two differ, the README wins.

## Design changes

- Before changing anything a proposal covers (see "When a proposal is needed" in the README), find its proposal with `go run ./tools/design list`.
- If there's none, draft one with `go run ./tools/design new <slug>` and stop, unless the person directing the work says to build it. Then the PR adds or updates the proposal as it goes.
- Refer to proposals by slug everywhere: `depends`, links, commits, and PRs. Never by issue number.
- Never write progress into a proposal's body. Progress goes in `status`, `branch`, Done criteria checkboxes, and dated Updates notes.
- Never edit an accepted proposal's body except to check or waive Done criteria items.
- The spec (`design/SPEC.md`) describes only what's built. Update it in the PR that finishes a proposal, never before.

## Signatures

- Never add, copy, re-date, or remove a signature line.
- Never run `go run ./tools/design sign` or edit `design/SIGNERS`.
- When a signed doc changes enough to need re-signing, say so in the PR description.

## Before opening a PR

- Run `go run ./tools/design check` and `go test ./...`.
- Never skip or disable a test to get CI green.
