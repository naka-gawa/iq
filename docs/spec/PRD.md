# Product Requirements Document: iq

**Version:** 0.1 (Draft)
**Status:** Draft

---

## 1. Overview

**iq** (INI Query) is a lightweight, dependency-free command-line tool for reading, writing, and transforming INI configuration files. It adopts the query syntax and pipeline philosophy of **jq** and **yq**, bringing modern developer ergonomics to a file format that has resisted them for decades.

```bash
# Read a value
iq '.database.host' config.ini

# Update in-place
iq -i '.database.host = "prod.example.com"' config.ini

# Inject a secret from the environment
iq -i '.credentials.api_key = strenv(API_KEY)' config.ini

# Convert to JSON for downstream tooling
iq -o json config.ini | jq '.database'
```

---

## 2. Problem Statement

INI files remain the configuration format of choice for critical infrastructure:

- **systemd** unit files (`.service`, `.target`, `.socket`)
- **Git** configuration (`.gitconfig`, `.git/config`)
- **AWS** credentials and config files (`~/.aws/credentials`)
- **PHP** runtime (`php.ini`)
- Countless legacy and modern applications

Despite their ubiquity, the tooling for manipulating INI files programmatically has not kept pace with modern developer expectations. Engineers in CI/CD pipelines are forced to choose between:

| Approach | Problem |
|---|---|
| `sed` / `awk` | Brittle. Breaks on comments, quoted values, and section reordering |
| **djui/iq** | Abandoned (last release 2018). Read-only. No mutation support |
| **mikefarah/yq** | Coerces INI to YAML AST, destroying comments, mangling booleans (`yes`/`no`), and silently corrupting values |
| **crudini** | Requires Python. Imperative `--set`/`--get` flags. No pipeline-style queries or filters |

The result: engineers either write fragile shell one-liners or reach for full scripting languages for tasks that should be single commands.

---

## 3. Target Users

| Persona | Use Case |
|---|---|
| **DevOps / SRE Engineer** | Mutating config files in CI/CD pipelines, Ansible playbooks, and Terraform provisioners |
| **Infrastructure Engineer** | Editing systemd unit files and gitconfig programmatically |
| **Developer** | Extracting and transforming values from application config files in scripts |

All three share one requirement: **the tool must not corrupt the file it touches**.

---

## 4. Design Principles

These principles govern every feature and implementation decision.

1. **Fidelity first.** Comments, blank lines, key ordering, and dialect-specific syntax are preserved on every read-write round trip. A tool that silently destroys comments will not be trusted in production.
2. **jq syntax is the API.** No new query language. Operators, paths, pipes, and filters follow jq conventions. Users with existing jq muscle memory gain zero learning curve.
3. **Pipeline-native.** Reads from stdin, writes to stdout. Composable with `grep`, `jq`, `curl`, and every other Unix tool.
4. **Single static binary.** No runtime dependencies. Runs in minimal Docker containers and CI runners without any setup step.
5. **Deterministic exit codes.** Automation depends on exit codes, not parsing output text.

---

## 5. Functional Requirements

### 5.1 Reading (Query)

**Basic value extraction**

```bash
iq '.section.key' file.ini
```

Extracts the exact string value of `key` inside `[section]`.

**Section extraction**

```bash
iq '.section' file.ini
```

Returns all key-value pairs of the section as a JSON object.

**Special characters in keys/sections**

```bash
iq '.["section name"]["key-with-dashes"]' file.ini
```

Bracket notation handles keys and section names that contain spaces, dots, or dashes.

**Chaining and filtering**

```bash
iq '.section | keys' file.ini
iq '.Service.ExecStart | select(test("pre-start"))' service.service
```

Supports jq pipe operators and filter expressions. When a key has multiple values (duplicate keys), the result is a JSON array that can be iterated with standard jq operators.

**Reading from stdin**

```bash
cat file.ini | iq '.section.key'
curl -s https://example.com/config.ini | iq '.database.host'
```

`-` or omitting a filename both signal stdin input.

---

### 5.2 Writing (Mutation)

**In-place update**

```bash
iq -i '.section.key = "new-value"' file.ini
```

The `-i` / `--in-place` flag writes back to the original file. Everything outside the mutated node — other keys, comments, blank lines, section order — is preserved exactly.

**Auto-create missing sections and keys**

If the target path does not exist, `iq` creates the section and key rather than erroring. This makes mutation idempotent by default.

**Multi-mutation in a single pass**

```bash
iq -i '.section1.key = "A" | .section2.key = "B"' file.ini
```

A single invocation handles multiple updates atomically.

**Safe environment variable injection**

```bash
iq -i '.credentials.secret = strenv(DB_PASSWORD)' config.ini
```

`strenv(VAR)` reads the value from the environment at runtime. The secret never appears in the shell command, process list, or shell history.

**Deletion**

```bash
iq -i 'del(.section.key)' file.ini
iq -i 'del(.section)' file.ini
```

Removes a key or an entire section.

---

### 5.3 Merging and Composition

**Multi-file merge**

```bash
iq eval-all '. as $item ireduce ({}; . * $item)' base.ini prod.ini
```

Merges `prod.ini` over `base.ini`. Keys in `prod.ini` overwrite their counterparts in `base.ini`; keys that exist only in `base.ini` are preserved.

**Merge conflict policies** (controlled by flag)

| Flag | Behavior |
|---|---|
| `--merge-overwrite` (default) | Values in later files win |
| `--merge-append` | Duplicate keys become an array (union) |
| `--merge-strict` | Error on any conflicting key |

---

### 5.4 Format Conversion

**INI to JSON**

```bash
iq -o json file.ini
```

Outputs the entire file as a JSON object. Sections become top-level keys; key-value pairs become nested objects.

**Type coercion** (applied by default on JSON output)

| INI value | JSON output |
|---|---|
| `42` | `42` (integer) |
| `3.14` | `3.14` (float) |
| `true` / `false` | `true` / `false` (boolean) |
| anything else | `"string"` |

`--raw-strings` disables all type coercion; every value is output as a JSON string.

---

### 5.5 Dialect Profiles

INI has no formal specification. Different ecosystems have developed incompatible conventions. `iq` ships named profiles that pre-configure the parser for known dialects.

**`--profile=systemd`**

- **Duplicate keys → array.** `ExecStart=` appearing multiple times is read as an ordered array. Writing an array back to file expands it to repeated keys.
- **Line continuation.** Lines ending with `\` are joined into a single logical value. The visual line break is preserved on write.
- **Comment delimiter.** Only `;` is treated as a comment (not `#`).

**`--profile=gitconfig`**

- **Subsection syntax.** `[section "subsection"]` is parsed as a two-level path. The query `.remote.origin.url` resolves to `[remote "origin"]` → `url`, and writes back using the original bracket syntax.
- **Case folding.** Section names and variable names are lowercased for comparison. Subsection names are case-sensitive.

**Auto-detection**

`iq` inspects the file extension and shebang/header to select the appropriate profile automatically:

| Extension / Pattern | Profile |
|---|---|
| `.service`, `.target`, `.socket`, `.mount`, `.timer` | `systemd` |
| `.gitconfig`, `.git/config` | `gitconfig` |
| Everything else | generic |

Auto-detection can be overridden with an explicit `--profile` flag.

---

### 5.6 INI Parsing Capabilities

These parsing features apply regardless of profile.

| Capability | Behavior |
|---|---|
| **Global properties** | Keys before the first section header are stored at the root level (internally `""` / `__default__`). They are not assigned to a synthetic section on write. |
| **Duplicate keys** | Automatically aggregated into a JSON array. Serialized back to repeated keys. |
| **Case sensitivity** | Configurable via `--ignore-case`. Default: case-sensitive on Linux, case-insensitive on Windows. |
| **Comment preservation** | `#` and `;` line comments and inline comments are attached to their adjacent node in the AST and survive mutation. |
| **Delimiter variants** | Both `=` and `:` are recognized as key-value delimiters. The original delimiter is preserved on write. |
| **Nested sections** | Dot-notation `[section.subsection]` and double-bracket `[[subsection]]` are parsed as hierarchical paths. |
| **Multi-line values** | Values split across lines with `\` continuation are supported. |
| **Whitespace** | Leading/trailing whitespace around delimiters is normalized; blank lines between sections are preserved. |

---

## 6. Developer Experience

### 6.1 Interactive Mode

```bash
iq --interactive file.ini
iq -I file.ini
```

Launches a TUI (terminal UI) where the user types a jq-style query and sees the extracted result update in real time. Pressing Enter copies the final query to stdout for use in scripts. Implemented with `charmbracelet/bubbletea`.

This mode is intended for query authoring — it eliminates the trial-and-error loop of running a command, reading the result, adjusting, and running again.

### 6.2 Syntax Highlighting

When writing to a TTY, sections, keys, values, and comments are rendered in distinct colors. When stdout is a pipe or file redirect, color output is suppressed automatically (no ANSI escape code pollution in downstream pipelines).

### 6.3 Error Output

All error messages are written to **stderr**. Stdout carries only the data result. This ensures that error messages never contaminate a pipeline or get written into a config file.

### 6.4 Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | General error (invalid syntax, file not found, permission denied) |
| `2` | Path not found (the queried key or section does not exist in the file) |

Exit code `2` is distinct so that callers can differentiate "the key is missing" from "the tool crashed."

---

## 7. Recommended Implementation Stack

| Component | Library | Rationale |
|---|---|---|
| INI parser | `gopkg.in/ini.v1` | Explicitly preserves key/section order and round-trip comment fidelity. Native parent-child AST for nested sections. |
| jq evaluator | `github.com/itchyny/gojq` | Pure-Go jq implementation. Reuses the jq expression language without reinventing it. |
| TUI framework | `github.com/charmbracelet/bubbletea` | Industry-standard Go TUI library; enables the interactive mode. |
| Color output | `github.com/fatih/color` | TTY-aware; automatically disables when stdout is not a terminal. |

The execution model for mutation:

1. Parse INI file into a `gopkg.in/ini.v1` AST.
2. Translate the AST to an in-memory map for jq evaluation.
3. Run the jq expression via `gojq` to identify target nodes.
4. Map results back to pointers in the `ini.v1` AST.
5. Apply mutations via the library's `SetValue` / delete APIs.
6. Serialize the modified AST back to the file system atomically (write to a temp file, then rename).

This dual-layer approach keeps jq's full filtering power while delegating all file serialization to a library purpose-built for INI round-trip fidelity.

---

## 8. Out of Scope (v1)

- INI ↔ YAML bidirectional conversion
- Full Windows INI compliance (CRLF, BOM, registry-format INI)
- Plugin or extension system
- Schema validation against a user-defined spec
- Network-fetched config files

---

## 9. Competitive Positioning Summary

| | **iq** | djui/iq | mikefarah/yq | crudini |
|---|---|---|---|---|
| Language | Go | Go + Ruby | Go | Python |
| Distribution | Single binary | Source / mixed | Single binary | OS package |
| Query syntax | jq-style | Dot notation | jq-style | Imperative flags |
| Mutation | Yes | No | Yes (with bugs) | Yes |
| Comment preservation | Yes (first-class) | N/A | Partial | Good |
| Nested sections | Yes | No | Partial | Partial |
| Dialect profiles | Yes | No | No | No |
| Active maintenance | Yes | No (abandoned 2018) | Yes | Slow |
