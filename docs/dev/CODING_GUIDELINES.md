# Coding Guidelines

## General

- Follow standard Go conventions: [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- Run `gofmt` and `golangci-lint` before every commit. CI enforces both.
- All exported symbols must have a doc comment.

## Package Structure

Each package has a single, clearly stated responsibility. Packages do not import siblings at the same level (no circular deps). `cmd/iq` may only import `internal/*`; it contains no business logic.

| Package | Responsibility |
|---|---|
| `cmd/iq` | CLI entry point: flag parsing, subcommand dispatch, exit code mapping |
| `internal/parser` | Load an INI file into a `*ini.File` AST with dialect options applied |
| `internal/dialect` | Detect the dialect profile from file path/content; define `Profile` and `LoadOptions` |
| `internal/query` | Translate `*ini.File` → `map[string]any`, build pointer table, execute gojq expressions |
| `internal/mutation` | Apply `(pointer, newValue)` pairs to the live AST; handle auto-create |
| `internal/serializer` | Atomically write the AST back to disk; render to stdout for non-in-place use |
| `internal/errors` | Sentinel error values shared across packages |
| `internal/tui` | Interactive query mode (bubbletea); imports `internal/query` only |

## Naming

| Context | Convention | Example |
|---|---|---|
| Exported types | PascalCase | `ParsedFile`, `DialectProfile` |
| Unexported vars | camelCase | `defaultSection` |
| Error variables | `Err` prefix | `ErrKeyNotFound`, `ErrPathInvalid` |
| Test files | `_test.go` suffix | `parser_test.go` |
| Source files | `snake_case.go` | `detect_dialect.go` |
| Fixture files | `snake_case` | `systemd_multivalue.ini` |
| Constants (iota) | PascalCase | `ProfileGeneric`, `ProfileSystemd` |
| Boolean flags | `is`/`has` prefix | `isInPlace`, `hasProfile` |

Do not use `SCREAMING_SNAKE_CASE` for constants. Prefer typed `iota` enums over bare `int` or `string` constants for profiles and output formats.

## Error Handling

- Always wrap errors with context using `fmt.Errorf("doing X: %w", err)`.
- Errors that a user might see are defined as sentinel values in `internal/errors/errors.go`.
- Exit code `2` (path not found) must originate from `ErrKeyNotFound`; never from a generic error.
- Do not use `panic` outside of `main`; return errors up the call stack.

Sentinel errors are defined in `internal/errors/errors.go`:

```go
var (
    ErrKeyNotFound     = errors.New("key not found")
    ErrPathInvalid     = errors.New("invalid path expression")
    ErrFileParseFailed = errors.New("failed to parse INI file")
    ErrDialectDetect   = errors.New("failed to detect dialect")
)
```

These are the only errors that may produce a specific exit code. All other errors map to exit code 1.

## Logging and Output

- **stdout**: data output only (extracted values, JSON, etc.)
- **stderr**: all errors, warnings, and diagnostic messages
- Never write to stdout in error paths. This is a hard rule — violating it corrupts pipelines.
- Color output: use `github.com/fatih/color` exclusively; never write raw ANSI codes.

## Testing

See [TESTING.md](TESTING.md) for the full strategy. <!-- same directory --> Summary:
- Table-driven tests using `t.Run`.
- Golden file tests for serialization round-trips.
- No mocking of the INI parser — use real `testdata/` fixture files.

## Dependencies

- Prefer the standard library. Add an external dependency only when the alternative is significant complexity.
- All dependencies must be pinned in `go.sum`.
- Avoid dependencies that require CGo.

## Concurrency

v1 is single-threaded. No goroutines are introduced. `eval-all` processes files sequentially in the order they are provided on the command line. This keeps the implementation simple and the output deterministic.

If concurrency is introduced in a future version, the rule is: `*ini.File` AST objects are never shared between goroutines. Each goroutine owns its own parsed AST; merging happens on plain `map[string]any` values after gojq evaluation.
