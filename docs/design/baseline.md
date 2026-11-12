---
slug: baseline
title: Baseline design
status: implemented
created: 2026-11-12
authors: [progrium]
depends: []
areas: [cli, site, jsx, templates, markdown, serve]
branch: [main]
---

# Proposal: Baseline design

## Abstract

Record taragen's design as it stood when the proposal process was adopted (commit `0f08e28`), so later proposals have a fixed starting point instead of a proposal backfilled for every past decision.

## Background

taragen was built over about 60 commits without design docs. Its decisions live in the code: the single-binary JSX runtime built on esbuild and goja, filesystem conventions instead of a config file, one data map per page, nested directory layouts, and serve mode with live reload.
The [spec](../spec.md) was written from the code as it stood and describes all of that, including the rough edges.

## Proposal

The spec as first committed is taragen's baseline design. Everything it describes counts as decided, and changes to it go through proposals from here on.
Test cases that prove existing behavior are tagged `[baseline]`.

## Rationale

Writing one proposal per past decision would take long and add little: the reasoning that matters is already in the spec's Key design decisions section.
Leaving existing behavior untracked would make it unclear which behavior is intended and which is accidental. A single implemented baseline settles that, and the spec's Known limitations section marks what's accidental.

## Compatibility

None. This records existing behavior and changes no code.

## Security

None: no code changes, so nothing new is reachable.

## Done criteria

- [x] The spec describes the CLI, site conventions, formats, rendering flow, build and serve modes, and known limitations, as of commit `0f08e28`.
- [x] The existing JSX cases in `jsx_test.go` are tagged `[baseline]` and pass with `go test ./...`.
- [x] `go run ./cmd/design test baseline` runs the tagged cases.

## Implementation

1. Write the spec from the code.
2. Tag the existing JSX test cases with `[baseline]`.

## Updates

2026-11-12: The spec was written by reading the code and building the example site in `web/`, which is how the `_assets/` build gap in Known limitations was found.
