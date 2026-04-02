# Testing Strategy

## Principles

- Tests use real INI fixture files from `testdata/`. No mocking of the parser layer.
- Every serialization path has a golden file test: parse → mutate → serialize must reproduce expected output byte-for-byte.
- Comment preservation is tested explicitly — it is a core correctness guarantee, not an afterthought.

## Test Types

### Unit Tests

Located alongside source files as `*_test.go`.
Cover individual functions in isolation (e.g., path tokenizer, type coercion logic).

### Integration Tests (`internal/*`)

Test a full parse → query or parse → mutate → serialize cycle using fixture files.
Each dialect has its own subdirectory under `testdata/`.

### Golden File Tests

For serialization round-trips: the expected output is stored as a `.golden` file.
Run with `-update` flag to regenerate golden files when output intentionally changes.

```bash
go test ./internal/serializer -update
```

The `-update` flag is implemented as a package-level variable in each test file that uses golden files:

```go
var update = flag.Bool("update", false, "regenerate golden files")

func TestSerialize_RoundTrip(t *testing.T) {
    // ...
    if *update {
        os.WriteFile(goldenPath, got, 0644)
    }
    want, _ := os.ReadFile(goldenPath)
    if !bytes.Equal(got, want) {
        t.Errorf("output differs from golden file %s", goldenPath)
    }
}
```

### CLI End-to-End Tests

Located in `internal/e2e/`. `TestMain` compiles the binary into a temp directory before any tests run, and cleans it up after:

```go
func TestMain(m *testing.M) {
    bin, err := buildBinary()  // go build -o tmpdir/iq ./cmd/iq
    if err != nil { log.Fatal(err) }
    binaryPath = bin
    code := m.Run()
    os.Remove(bin)
    os.Exit(code)
}
```

Each test case invokes the binary via `exec.Command`, then asserts on stdout content, stderr content, and exit code.

> Note: The exact helper structure will be finalized once the CLI entry point is stable.

## Fixture Layout

```
testdata/
├── generic/
│   ├── basic.ini
│   ├── comments.ini
│   ├── global_properties.ini
│   ├── duplicate_keys.ini
│   └── special_chars.ini
├── systemd/
│   ├── multivalue_execstart.service
│   ├── line_continuation.service
│   └── ...
├── gitconfig/
│   ├── subsections.gitconfig
│   ├── case_folding.gitconfig
│   └── ...
├── aws/
│   ├── credentials.ini
│   └── config.ini
└── windows/
    ├── basic.ini
    └── crlf.ini
```

## Coverage Requirements

> Provisional targets — revisit after the first working implementation.

| Package | Target |
|---|---|
| `internal/parser` | ≥ 90% |
| `internal/query` | ≥ 85% |
| `internal/mutation` | ≥ 90% |
| `internal/serializer` | ≥ 90% |
| `internal/dialect` | ≥ 80% |

## Test Naming Convention

```go
// Table-driven test
func TestParser_GlobalProperties(t *testing.T) {
    cases := []struct {
        name  string
        input string
        want  map[string]string
    }{
        {"key before first section", "key=val\n[s]\n", map[string]string{"key": "val"}},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) { ... })
    }
}
```

Pattern: `Test<Type>_<Scenario>` for exported types, `test<function>_<scenario>` for helpers.

## Critical Test Scenarios

The following behaviors must have explicit test coverage because they are core correctness guarantees:

- [ ] Comments survive a read → write round-trip with no mutation
- [ ] Comments survive a read → mutate-one-key → write round-trip
- [ ] Global properties (before first section) are preserved on write
- [ ] Duplicate keys round-trip as repeated keys (not as a YAML array)
- [ ] `--in-place` write is atomic (temp file + rename)
- [ ] Exit code `2` is returned when a queried key does not exist
- [ ] ANSI color codes are absent when stdout is not a TTY
