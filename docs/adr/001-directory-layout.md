# ADR 001: Directory Layout — When to Use `internal/` and Why `pkg/` Is Prohibited

**Status**: Accepted  
**Date**: 2026-06-08

## Context

Two directory conventions appear frequently in Go codebases and are often applied without deliberate reasoning:

- **`internal/`** — enforced by the Go compiler: packages under `internal/` cannot be imported by code outside the parent module. Often added preemptively to "hide" packages.
- **`pkg/`** — popularised by `golang-standards/project-layout`, a community-maintained repository. Not an official Go recommendation.

Both conventions carry costs when misapplied. This ADR records the policy for `iq`.

## Decision

### `internal/` serves two distinct roles — distinguish them

`internal/` can be used for two different reasons, and conflating them leads to confusion:

**Role 1 — structural grouping**: keeps implementation packages out of the module root, making the top-level directory easier to navigate. This is a legitimate reason on its own: `internal/` is the only Go-toolchain-recognised grouping prefix that does not require semantic justification for each package it contains. Alternatives like `src/` (a GOPATH-era relic, non-standard in modules) or `utils/` (semantically vacuous, same problem as `pkg/`) are worse choices.

**Role 2 — compiler-enforced access control**: prevents any external module from importing the packages. This is the intended Go language feature, and it is a meaningful constraint — but only when there is something to distinguish. If every package in the module is under `internal/`, the constraint applies uniformly and carries no signal about _which_ packages are especially sensitive.

The correct question before adding a new package to `internal/` is:

> "Am I using `internal/` for structural grouping, compiler enforcement, or both — and is either reason applicable here?"

### Why the current `iq` packages are in `internal/`

Both roles apply:

1. **Structural grouping**: `iq` has seven implementation packages. Placing them all at the module root would clutter the top-level directory alongside `cmd/`, `docs/`, `testdata/`, and `main.go`. `internal/` provides a clean separation without inventing a meaningless prefix name.

2. **Compiler enforcement**: `iq` is a single-binary CLI tool. It is not intended to be used as an importable library. The packages under `internal/` are implementation details of the binary with no external-import contract. Compiler enforcement reinforces this, even if the realistic probability of an accidental external import is low.

### Why not `pkg/`, `src/`, or `utils/`

| Option | Problem |
|--------|---------|
| `pkg/` | No official Go backing; adds path depth without meaning |
| `src/` | Belongs to the GOPATH workspace layout, not Go modules; confuses tooling expectations |
| `utils/` | Semantically empty; says nothing about the packages it contains |

`internal/` is the only prefix that is recognised by the Go toolchain and carries an enforced constraint. For structural grouping, it is therefore the best available option even when compiler enforcement is not the primary motivation.

### Default: flat layout at the module root

For projects with only a small number of packages, flat layout at the module root is often the right choice. There is no reason to introduce `internal/` just because a package "feels private". The default should be flat; `internal/` should be introduced when either structural grouping or compiler enforcement (or both) provides concrete benefit.

### `pkg/` is prohibited

`pkg/` adds path depth without conveying meaning. It is not recommended by the Go team. A CI check in `.github/workflows/pr.yaml` enforces its absence on every pull request.

## Consequences

- When adding a new package, ask which role(s) of `internal/` apply. If neither does, use flat layout.
- `pkg/` is blocked by CI.
- If `iq` ever gains a public library API, affected packages should be moved out of `internal/` at that time. This ADR should be revised to reflect the updated scope.

## Decision Criterion (quick reference)

| Situation | Placement |
|-----------|-----------|
| Small number of packages; root stays uncluttered | Module root (`iq/<name>/`) |
| Many packages; `internal/` keeps root navigable | `internal/<name>/` |
| Package must not be importable by external modules | `internal/<name>/` |
| `pkg/` prefix considered | Not allowed |
| `src/` or `utils/` prefix considered | Not allowed |
