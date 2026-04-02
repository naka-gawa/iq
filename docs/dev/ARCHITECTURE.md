# Architecture

## Overview

`iq` is structured as a pipeline: **parse → query/mutate → serialize**.
The INI AST is the single source of truth throughout; no intermediate representation is persisted to disk.

```
┌─────────────┐     ┌───────────────┐     ┌──────────────────┐     ┌─────────────┐
│  Input      │────▶│  INI Parser   │────▶│  Query Engine    │────▶│  Serializer │
│  (file/     │     │  (ini.v1 AST) │     │  (gojq + AST     │     │  (ini.v1    │
│   stdin)    │     │               │     │   pointer map)   │     │   write)    │
└─────────────┘     └───────────────┘     └──────────────────┘     └─────────────┘
                           │                       │
                    ┌──────┴──────┐         ┌──────┴──────┐
                    │  Dialect    │         │  Mutation   │
                    │  Detector   │         │  Applier    │
                    └─────────────┘         └─────────────┘
```

## Components

### Parser (`internal/parser`)

Accepts a file path or `io.Reader` plus a `dialect.Profile`, applies the profile's `ini.LoadOptions`, and returns a `*ini.File` AST. All parse errors are wrapped as `ErrFileParseFailed`. This is the only package that imports `gopkg.in/ini.v1` directly; all other packages receive the AST through this interface.

### Dialect Detector (`internal/dialect`)

Examines a file path and, optionally, its first few bytes to select the appropriate `Profile`. Detection priority: explicit `--profile` flag → file extension → content heuristics. Returns the generic profile when no match is found. Profiles carry `ini.LoadOptions` (e.g., `AllowShadows` for systemd duplicate keys) and behavioral flags consumed by the query and mutation layers.

### Query Engine (`internal/query`)

Converts the `*ini.File` AST into a `map[string]any` for gojq evaluation, and simultaneously builds a pointer table mapping each JSON path string to its `*ini.Key` or `*ini.Section` node in the live AST. Executes jq expressions via `github.com/itchyny/gojq`. On read-only queries, results are serialized to stdout. On mutation queries, the matched paths are returned to the Mutation Applier as a list of `(pointer, newValue)` pairs.

### Mutation Applier (`internal/mutation`)

Receives `(pointer, newValue)` pairs from the Query Engine and applies them to the live `*ini.File` AST using `ini.v1`'s setter and delete APIs. Never re-parses the file; operates entirely on the in-memory AST so that comments and ordering are untouched. Handles auto-creation of missing sections and keys when the target path does not exist.

### Serializer (`internal/serializer`)

Writes the mutated `*ini.File` AST to a temporary file in the same directory as the target, then calls `os.Rename` for an atomic replace. Copies the original file's permissions (`os.Stat` → `os.Chmod`) before writing. Used only when `--in-place` is active; otherwise the AST is rendered directly to stdout.

### CLI (`cmd/iq`)

The top-level entry point. Parses flags (`--in-place`, `--output`, `--profile`, `--ignore-case`, `--raw-strings`) and dispatches to the appropriate path: single-file query/mutate, `eval-all` subcommand, or interactive TUI mode. Maps internal sentinel errors to exit codes (`ErrKeyNotFound` → 2, all others → 1) and ensures all error messages go to stderr, never stdout.

## Data Flow: Read

```
file.ini ──▶ Parser ──▶ ini.File AST ──▶ toMap() ──▶ gojq eval ──▶ stdout
```

## Data Flow: Mutate (--in-place)

```
file.ini ──▶ Parser ──▶ ini.File AST ──▶ toMap() + pointer table
                                                │
                              gojq eval ────────┘
                                                │
                              pointer table lookup ──▶ AST mutation ──▶ atomic write
```

## Key Design Decisions

| Decision | Rationale |
|---|---|
| Use `gopkg.in/ini.v1` as the AST layer | Only Go INI library with explicit comment and order preservation guarantees |
| Use `gojq` for expression evaluation | Reuses the jq language without maintaining a parser; pure Go, no CGo |
| Dual-layer (AST + map) | Keeps jq filtering power while delegating serialization to a battle-tested library |
| Atomic write (temp + rename) | Prevents partial writes from leaving a corrupted config file |
| Comments as first-class AST nodes | Satisfies the fidelity principle; never destroyed by mutation |

## Package Layout

```
iq/
├── cmd/
│   └── iq/           # main entry point
├── internal/
│   ├── parser/       # INI parsing and dialect application
│   ├── dialect/      # profile detection and configuration
│   ├── query/        # AST→map translation and gojq execution
│   ├── mutation/     # AST mutation applier
│   ├── serializer/   # atomic file write
│   └── tui/          # interactive mode (bubbletea)
├── testdata/         # INI fixture files per dialect
└── docs/
```
