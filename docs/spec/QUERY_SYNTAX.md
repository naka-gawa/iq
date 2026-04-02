# Query Syntax

`iq` uses jq-compatible path expressions. This document defines exactly which jq operators are supported, which are out of scope, and where `iq`'s behavior intentionally diverges from standard jq.

---

## Supported Operators

### Path Access

| Syntax | Description | Example |
|---|---|---|
| `.section.key` | Dot-notation path | `.database.host` |
| `.["key"]` | Bracket notation (for keys with spaces, dashes, dots) | `.["my section"]["host-name"]` |
| `.section` | Whole-section access (returns JSON object) | `.database` |
| `.` | Root (returns entire file as JSON) | `.` |

### Mutation

| Syntax | Description | Example |
|---|---|---|
| `.section.key = "value"` | Set a string value | `.database.host = "localhost"` |
| `.section.key = 42` | Set a numeric value | `.server.port = 8080` |
| `del(.section.key)` | Delete a key | `del(.cache.ttl)` |
| `del(.section)` | Delete an entire section | `del(.legacy)` |

### Pipes and Composition

| Syntax | Description | Example |
|---|---|---|
| `expr1 \| expr2` | Pipe output of one expression to next | `.service \| keys` |
| `expr1 \| expr2 \| expr3` | Multi-stage pipeline | `.service.execstart \| .[]` |

### Array Operators (for duplicate keys)

| Syntax | Description | Example |
|---|---|---|
| `.section.key[]` | Iterate over array values | `.Service.ExecStart[]` |
| `.section.key[0]` | Index into array | `.Service.ExecStart[0]` |
| `select(expr)` | Filter array items | `.Service.ExecStart[] \| select(test("pre"))` |

### String Operators

| Syntax | Description | Example |
|---|---|---|
| `test("pattern")` | Regex match test (returns bool) | `.section.key \| test("^http")` |
| `ltrimstr("prefix")` | Remove prefix | `.section.key \| ltrimstr("https://")` |
| `rtrimstr("suffix")` | Remove suffix | `.section.key \| rtrimstr(".conf")` |
| `ascii_downcase` | Lowercase | `.section.key \| ascii_downcase` |
| `ascii_upcase` | Uppercase | `.section.key \| ascii_upcase` |

### Introspection

| Syntax | Description | Example |
|---|---|---|
| `keys` | Array of key names in a section | `.database \| keys` |
| `has("key")` | Test key existence (returns bool) | `.database \| has("port")` |
| `length` | Number of keys in a section | `.database \| length` |
| `type` | Type of value (`"string"`, `"array"`, `"object"`) | `.Service.ExecStart \| type` |

### `iq`-Specific Extensions

| Syntax | Description | Example |
|---|---|---|
| `strenv(VAR)` | Read value from environment variable | `.credentials.secret = strenv(DB_PASS)` |
| `env.VAR` | Alternative env var access | `.credentials.secret = env.DB_PASS` |

---

## Divergences from Standard jq

These are intentional differences, not bugs.

| Feature | Standard jq | `iq` behavior | Reason |
|---|---|---|---|
| Type system | Typed (null, bool, number, string, array, object) | All INI values are strings at rest; type coercion on JSON output only | INI has no native types |
| `null` | First-class value | No `null`; missing key returns exit code 2 | Avoids silent failures in pipelines |
| Recursive descent `..` | Descends into all nested objects | Not supported in v1 | INI nesting is at most two levels deep; full recursion adds complexity without practical benefit |
| `@base64`, `@uri`, `@csv` format strings | Supported | Not supported in v1 | Low priority for INI use cases |
| `reduce`, `foreach` | Supported | Supported via `ireduce` in `eval-all` only | Only needed in multi-file merge context |
| `$__loc__` | Debug location | Not supported | Not applicable |
| `path(expr)` | Returns path of matching nodes | Not supported in v1 | Rarely needed for INI's flat structure |
| `input`, `inputs` | Read additional inputs | `eval-all` subcommand used instead | Clearer UX for multi-file operations |

---

## Unsupported jq Features (v1)

The following jq features are **not supported** in v1. Using them returns exit code 1 with an informative error.

- `@format` string interpolation (`@base64d`, `@uri`, `@html`, `@csv`, `@tsv`, `@json`, `@text`, `@sh`)
- `$__loc__`
- `limit(n; expr)`, `first(expr)`, `last(expr)`
- `input` / `inputs` (use `eval-all` instead)
- SQL-style operators (`INDEX`, `IN`, `GROUP_BY`)
- `try ... catch`
- Variable binding (`expr as $var | ...`)
- Recursive descent `..`
- `path(expr)`

---

## Exit Code Behavior for Expressions

| Condition | Exit code |
|---|---|
| Expression matches and returns a value | `0` |
| Expression is valid but key/section does not exist | `2` |
| Expression syntax is invalid | `1` |
| File cannot be read | `1` |

---

## Examples

```bash
# Extract a single value
iq '.database.host' config.ini

# Extract with bracket notation
iq '.["my app"]["log-level"]' config.ini

# Check if a key exists (exit code 0 or 2)
iq 'has("port")' <<< "$(iq '.database' config.ini)"

# Filter duplicate keys
iq '.Service.ExecStart[] | select(test("--primary"))' unit.service

# Multiple mutations in one pass
iq -i '.database.host = "prod.db" | .database.port = "5432"' config.ini

# Inject secret from environment
iq -i '.api.token = strenv(API_TOKEN)' config.ini

# Convert to JSON
iq -o json config.ini

# Convert to JSON with raw strings (no type coercion)
iq -o json --raw-strings config.ini
```
