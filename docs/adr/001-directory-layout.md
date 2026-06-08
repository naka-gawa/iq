# ADR 001: Directory Layout — When to Use `internal/` and Why `pkg/` Is Prohibited

**Status**: Accepted  
**Date**: 2026-06-08

## Context

Two directory conventions appear frequently in Go codebases and are often applied without deliberate reasoning:

- **`internal/`** — enforced by the Go compiler: packages under `internal/` cannot be imported by code outside the parent module. Often added preemptively to "hide" packages.
- **`pkg/`** — popularised by `golang-standards/project-layout`, a community-maintained repository. Not an official Go recommendation.

Both conventions carry costs when misapplied. This ADR records the policy for `iq`.

## Decision

### Default: flat layout at the module root

For a small-to-medium Go project, placing packages directly under the module root (`iq/parser`, `iq/query`, …) keeps import paths short and reduces cognitive overhead. There is no structural reason to nest packages inside a prefix directory unless that prefix carries explicit meaning enforced by the toolchain.

### Use `internal/` only when there is a concrete reason

`internal/` is appropriate when a package is an implementation detail that must not be importable by external modules. The bar is:

> "If an external module imported this package, would that create a coupling we are unwilling to support?"

If the answer is yes, `internal/` is the right choice. If the answer is "I'm not sure" or "maybe someday", it is not. Moving a package into `internal/` later costs nothing; moving it out requires renaming every import path.

Reasons that do **not** justify `internal/`:

- "It feels private" — visibility without compiler enforcement is just convention; use unexported identifiers instead
- "We might want to hide it in the future" — apply constraints when they are needed, not in advance
- "Everyone else does it" — cargo-culting a pattern does not make it correct for this codebase

### Why the current `iq` packages are in `internal/`

`iq` is a single-binary CLI tool. It is not intended to be an importable library. Every package under `internal/` is an implementation detail of the binary with no external-import contract. This satisfies the concrete-reason bar above: the answer to "should external modules import `iq/parser`?" is an unambiguous no.

This placement is justified **because `iq` is a CLI tool**, not because `internal/` is the default for all new packages. Future packages added to `iq` should go through the same reasoning — they do not inherit `internal/` placement automatically.

### `pkg/` is prohibited

`pkg/` adds path depth (`iq/pkg/foo` vs `iq/foo`) without conveying meaning beyond "this is a package", which is self-evident. It is not recommended by the Go team. A CI check in `.github/workflows/pr.yaml` enforces its absence on every pull request.

## Consequences

- When adding a new package, apply the decision criterion above explicitly. Do not default to `internal/`.
- `pkg/` is blocked by CI.
- If `iq` ever gains a public library API, affected packages should be moved out of `internal/` at that time. This ADR should be revised to reflect the updated scope.

## Decision Criterion (quick reference)

| Situation | Placement |
|-----------|-----------|
| Package is a `iq` binary implementation detail; external import would be undesirable | `internal/<name>/` |
| Package is intended to be imported by external modules | module root (`iq/<name>/`) |
| `pkg/` prefix considered | Not allowed |
