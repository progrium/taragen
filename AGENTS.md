# Agent guidance

Rules for AI agents working in this repository. They restate [the design process](docs/design/README.md); where the two differ, the README wins.

## Design changes

- Before changing anything a proposal covers (see "When a proposal is needed" in the README), find its proposal with `go run ./cmd/design list`.
- If there's none, draft one with `go run ./cmd/design new <slug>` and stop, unless the person directing the work says to build it. Then the PR adds or updates the proposal as it goes.
- Refer to proposals by slug everywhere: `depends`, links, commits, PRs, and test tags. Never by issue number.
- Never write progress into a proposal's body. Progress goes in `status`, `branch`, Done criteria checkboxes, and dated Updates notes.
- Never edit an accepted proposal's body except to check or waive Done criteria items.
- The spec (`docs/spec.md`) describes only what's built. Update it in the PR that finishes a proposal, never before.

## Signatures

- Never add, copy, re-date, or remove a signature line.
- Never run `go run ./cmd/design sign`, post a `/sign` comment, or edit `docs/design/SIGNERS`.
- When a signed doc changes enough to need re-signing, say so in the PR description.

## Tests

- Tag new test cases with the proposals they prove: end the subtest name with `[slug]` or `[a, b]`, written exactly like that, and run them with `proposaltest.Run`.
- Never skip, disable, or untag a test to get CI green.

## Before opening a PR

- Run `go run ./cmd/design check` and `go test ./...`.
- Never run `go run ./cmd/design status`, and never commit `docs/design/STATUS.md`. Only the status job on main writes it.
- Agent branches start with `claude/`. Their PRs get only the design check in CI, so run the tests yourself.
