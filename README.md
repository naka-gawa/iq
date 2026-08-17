# iq

[![Go Version](https://img.shields.io/badge/go-1.25.6-blue)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/naka-gawa/iq)](https://github.com/naka-gawa/iq/releases)

A lightweight, portable CLI for reading, writing, and transforming INI configuration files — with jq-style query syntax and full round-trip fidelity.

## What

`iq` brings the expressiveness of `jq` to INI files. You can query, mutate, and reformat any INI file without losing comments, blank lines, or key ordering. It reads from files or stdin and writes to stdout or back in-place.

```bash
iq '.database.host' config.ini
# → "localhost"

iq -i '.database.host = "prod.example.com"' config.ini
# updates file in-place, comments preserved
```

## Why

INI files are still everywhere: `systemd` units, `.gitconfig`, AWS credentials, PHP runtime config, legacy app settings. But the Unix toolbox has no native way to query or mutate them precisely — `sed`, `awk`, and `grep` work on text, not structure, and they silently destroy comments and formatting.

`iq` fills that gap. It treats INI files as structured data and exposes them through a query language developers already know, while guaranteeing the file comes back out exactly as it went in — minus only the specific change you asked for.

## How

### Install

```bash
git clone https://github.com/naka-gawa/iq.git
cd iq
make init    # install pinned tools via aqua
make build   # produces ./iq binary
```

To install system-wide:

```bash
sudo make install          # installs to /usr/bin by default
INSTALL_DIR=~/.local/bin make install
```

### Query a value

```bash
iq '.database.host' config.ini

# from stdin
cat config.ini | iq '.database.host'
curl -s https://example.com/config.ini | iq '.database.host'
```

### Query a section

```bash
iq '.database' config.ini
```

### Mutate in-place

```bash
# set a value
iq -i '.database.host = "prod.example.com"' config.ini

# inject an environment variable safely
iq -i '.credentials.api_key = strenv(API_KEY)' config.ini

# multiple mutations in one pass
iq -i '.section1.key = "A" | .section2.key = "B"' config.ini

# delete a key
iq -i 'del(.section.key)' config.ini

# delete an entire section
iq -i 'del(.section)' config.ini
```

### Format conversion

```bash
# output as JSON
iq -o json config.ini

# output as JSON with all values as strings (no type coercion)
iq --raw-strings -o json config.ini
```

### Merge multiple files (`eval-all`)

```bash
# deep-merge prod.ini over base.ini (later files win on conflict)
iq eval-all base.ini prod.ini

# merge to JSON
iq eval-all -o json base.ini prod.ini

# union conflicting values into arrays
iq eval-all --merge-append base.ini prod.ini

# fail (exit 1) if two files disagree on a key
iq eval-all --merge-strict base.ini prod.ini
```

`eval-all` requires two or more files and merges them section by section: keys in later files overwrite earlier ones, while keys unique to earlier files are preserved. Because the result is synthesized from multiple sources, comments from the inputs are not carried over. Output defaults to INI; use `-o json` to emit JSON.

### Advanced queries (jq pipes and filters)

```bash
# list keys in a section
iq '.database | keys' config.ini

# filter with a condition
iq '.Service.ExecStart | select(test("pre-start"))' service.service
```

### Interactive mode

```bash
# open a live query editor (type a jq expression, see results update in real time)
iq --interactive config.ini
iq -I config.ini

# interactive over piped INI data (keyboard read from /dev/tty)
cat config.ini | iq -I
```

Press Enter to print the final query to stdout; Esc or Ctrl+C exits without printing.

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--in-place` | `-i` | `false` | Write mutations back to the original file |
| `--interactive` | `-I` | `false` | Launch interactive query mode (TUI with live jq preview) |
| `--output` | `-o` | `ini` | Output format: `ini` or `json` |
| `--raw-strings` | — | `false` | Disable JSON type coercion (all values as strings) |
| `--profile` | — | `generic` | Dialect profile: `generic`, `systemd`, `gitconfig` (auto-detected from file extension when omitted) |
| `--ignore-case` | — | `false` | Case-insensitive key and section matching |

**`eval-all` subcommand flags** (in addition to `-o`, `--raw-strings`, `--profile`, `--ignore-case`):

| Flag | Default | Description |
|---|---|---|
| `--merge-overwrite` | `true` | Later files win on conflict |
| `--merge-append` | `false` | Union conflicting values into an array |
| `--merge-strict` | `false` | Error (exit 1) when files disagree on a key |

## Prerequisites / Environment

| | |
|---|---|
| **Language** | Go 1.25.6 |
| **CLI framework** | [cobra](https://github.com/spf13/cobra) v1.10.2 |
| **INI backend** | [gopkg.in/ini.v1](https://github.com/go-ini/ini) v1.67.1 |
| **jq evaluator** | [gojq](https://github.com/itchyny/gojq) v0.12.19 |
| **Color output** | [fatih/color](https://github.com/fatih/color) v1.19.0 |
| **TUI** | [bubbletea](https://github.com/charmbracelet/bubbletea) v2 |
| **OS** | Linux, macOS (single static binary, no CGO) |

Tool versions are pinned via [aqua](https://aquaproj.github.io/). Run `make init` to install them.

## Output Example

Given `config.ini`:

```ini
[database]
# primary host
host = localhost
port = 5432
```

```bash
$ iq '.database' config.ini
{
  "host": "localhost",
  "port": 5432
}

$ iq -i '.database.host = "prod.db.internal"' config.ini
$ cat config.ini
[database]
# primary host
host = prod.db.internal
port = 5432
```

Comments and structure are preserved exactly.

## Notes

**Round-trip fidelity** is a hard guarantee, not a best-effort feature. Comments, blank lines, and key ordering survive every read-write cycle. Any regression in this behavior is a bug.

**Exit codes** follow a stable contract for scripting:

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | General error (parse failure, bad path, permission denied) |
| `2` | Key or section not found |

**Dialect profiles** handle INI variants automatically. `systemd` is detected from `.service`, `.target`, `.socket`, `.mount`, `.timer`, `.path`, `.scope`, `.slice` extensions; `gitconfig` is detected from `.gitconfig` filenames and `.git/config` paths. Override with `--profile`.

**Interactive TUI** (`--interactive` / `-I`) opens a live query editor: type a jq-style expression, watch the matching result update in real time, and press Enter to print the final query to stdout. Esc / Ctrl+C exits without printing. It works on a file argument or on piped INI data (`cat config.ini | iq -I`), in which case the keyboard is read from `/dev/tty`.

**Color output** is TTY-aware and automatically disabled when piping or redirecting.

## License

[MIT](LICENSE) — Copyright 2026 Tomoaki Nakagawa

## Author

**naka-gawa** — [github.com/naka-gawa](https://github.com/naka-gawa)

Bugs and feature requests: [github.com/naka-gawa/iq/issues](https://github.com/naka-gawa/iq/issues)
