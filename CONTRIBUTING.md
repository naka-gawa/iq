# Contributing

## Before You Start

- Check existing Issues and PRs to avoid duplicate work.
- For significant changes, open an Issue first to align on approach before writing code.

## Development Setup

See [docs/dev/DEVELOPMENT.md](docs/dev/DEVELOPMENT.md).

## Directory Layout

See [docs/adr/001-directory-layout.md](docs/adr/001-directory-layout.md) for the full rationale.

- New packages go under `internal/` for structural grouping (keeping root clean) or to prevent external import; otherwise place them at the module root
- `pkg/` is prohibited — a CI check enforces this on every PR
- Do not move packages out of `internal/` without a corresponding ADR update

## Workflow

1. Fork the repository and create a branch from `main`.
2. Branch naming: `feat/<short-description>`, `fix/<short-description>`, `docs/<short-description>`.
3. Write tests for your changes (see [docs/dev/TESTING.md](docs/dev/TESTING.md)).
4. Run `golangci-lint run` and `go test ./...` locally before opening a PR.
5. Open a PR against `main`. Fill in the PR template.

## PR Requirements

- [ ] Tests pass (`go test ./...`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] New behavior has test coverage
- [ ] Comment preservation is not broken (run the critical test scenarios from TESTING.md)
- [ ] If a new dialect is added, a fixture file and detection test exist
- [ ] No `pkg/` directory introduced (CI enforces this)

## Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <short summary (50 chars or less)>

<optional body — explain WHY, not what>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`

Examples:
```
feat: add --profile=windows flag for CRLF-aware parsing
fix: preserve inline comments when mutating systemd unit files
docs: document AWS [profile ...] normalization behavior
```

## Reporting Bugs

Use the bug report template at `.github/ISSUE_TEMPLATE/bug_report.md`.

Include:
- The INI file content (or a minimal reproducing example)
- The command you ran
- Expected output vs actual output
- OS and `iq --version` output

## Adding a New Dialect

See the "Adding a New Dialect Profile" section in [docs/dev/DEVELOPMENT.md](docs/dev/DEVELOPMENT.md).
New dialects require:
1. A profile definition
2. Detection heuristics
3. At least two fixture files
4. Documentation in [docs/spec/DIALECTS.md](docs/spec/DIALECTS.md)
