# ADR 001: Directory Layout — `internal/` Usage and `pkg/` Prohibition

**Status**: Accepted  
**Date**: 2026-06-08

## Context

`iq` is a single-binary CLI tool. It is not intended to be used as an importable Go library by external consumers. All core logic lives in purpose-named packages under `internal/`:

| Package | Responsibility |
|---------|---------------|
| `internal/errors` | Sentinel errors and rich error types; drives exit code mapping |
| `internal/parser` | Loads INI files into `*ini.File` AST; the only package importing `gopkg.in/ini.v1` |
| `internal/query` | Translates AST to map and executes jq expressions via `gojq` |
| `internal/dialect` | Dialect detection and profile definitions |
| `internal/mutation` | Assignment and deletion mutations on the AST |
| `internal/serializer` | Serializes AST to INI or JSON output |
| `internal/e2e` | End-to-end test suite |

Two alternative layouts were considered during initial design:

1. **Flat layout** (`iq/parser`, `iq/query`, …) — packages importable by external modules
2. **`pkg/` layout** (`iq/pkg/parser`, `iq/pkg/query`, …) — a community convention with no official Go backing

## Decision

All implementation packages are placed under `internal/`. The `pkg/` directory is not used and is prohibited.

### Why `internal/`

The Go compiler enforces that packages under `internal/` cannot be imported by code outside the parent module. For a CLI tool this is the correct constraint: implementation details are not a public contract and must not be accidentally stabilised by external importers.

Placing packages under `internal/` also makes the intent unambiguous to contributors. A package outside `internal/` implies "this is intended for external use" — a signal that would be misleading here.

### Why not `pkg/`

`pkg/` originates from the internal source layout of the Go standard library and was popularised by `golang-standards/project-layout`, a community-maintained repository that is **not** an official Go recommendation. The Go team has never endorsed `pkg/` as a convention.

Beyond its unofficial status, `pkg/` adds path depth (`iq/pkg/foo` vs `iq/foo`) without conveying meaning — it says only "this is a package", which is self-evident. Responsibility-named directories (`internal/parser/`, `internal/query/`) communicate far more than a structural prefix.

## Consequences

- New packages must be added under `internal/<responsibility>/`.
- A CI step in `.github/workflows/pr.yaml` enforces the absence of a `pkg/` directory on every pull request.
- The sentinel errors in `internal/errors` are not reachable via `errors.Is` by external code. This is intentional: consumers of the `iq` binary observe exit codes, not Go error values.

## Library Extraction (resolves issue #59)

Exposing `iq` as an importable library is **out of scope for v1**. If this goal is adopted in the future, the affected packages should be moved out of `internal/` at that time, and this ADR should be revised to reflect the updated decision. There is no value in pre-emptively restructuring now — moving code out of `internal/` costs an import-path refactor across all callers, while moving code in costs nothing.
