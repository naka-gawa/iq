# Development Guide

## Prerequisites

- Go 1.22 or later
- `golangci-lint` (install via `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`)
- GNU Make (optional, for convenience targets)

## Getting Started

```bash
git clone https://github.com/your-org/iq.git
cd iq
go mod tidy
go build ./cmd/iq
```

The built binary is placed at `./iq`.

## Common Commands

```bash
# Build
go build ./cmd/iq

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Lint
golangci-lint run

# Run a single test
go test ./internal/parser -run TestGlobalProperties

# Build for multiple platforms (example)
GOOS=linux GOARCH=amd64 go build -o iq-linux-amd64 ./cmd/iq
```

The following `make` targets are available:

```makefile
make build        # go build ./cmd/iq → ./iq
make test         # go test ./...
make lint         # golangci-lint run
make release-dry  # goreleaser release --snapshot --clean (no publish)
```

## Project Layout

```
iq/
├── cmd/iq/           # CLI entry point (main.go)
├── internal/         # All business logic (unexported)
├── testdata/         # INI fixture files used by tests
├── docs/             # Project documentation
├── go.mod
├── go.sum
└── .golangci.yml     # Linter configuration
```

## Adding a New Dialect Profile

1. Add a new `Profile` constant in `internal/dialect/profiles.go`
2. Add detection heuristics in `internal/dialect/detect.go`
3. Add parser options in `internal/dialect/options.go`
4. Add fixture files in `testdata/<dialect>/`
5. Add test cases in `internal/dialect/detect_test.go`

## Debugging

Set `IQ_DEBUG=1` to enable verbose logging to stderr. This prints the parsed AST structure, the pointer table after query resolution, and the gojq expression being evaluated.

```bash
IQ_DEBUG=1 iq '.section.key' file.ini
```

> Note: `IQ_DEBUG` is a design intent; the exact output format will be defined during implementation.

## Release Process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions. The workflow triggers on `vMAJOR.MINOR.PATCH` tags and publishes multi-platform binaries to GitHub Releases.

```bash
# Tag and push to trigger the release workflow
git tag v0.1.0
git push origin v0.1.0
```

To test the release build locally without publishing:

```bash
make release-dry
# equivalent to: goreleaser release --snapshot --clean
```

The GoReleaser configuration lives at `.goreleaser.yaml` in the repository root. It builds for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64`.
