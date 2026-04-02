# INI Dialect Reference

INI has no formal specification. Each ecosystem has developed its own conventions.
`iq` ships named dialect profiles that pre-configure the parser for known variants.

## Profile Selection

```bash
# Explicit
iq --profile=systemd '.Service.ExecStart' unit.service

# Auto-detected (based on file extension and content heuristics)
iq '.Service.ExecStart' unit.service
```

Auto-detection can always be overridden with `--profile`.

---

## Generic (default)

Applied when no other profile is detected.

| Feature | Behavior |
|---|---|
| Key-value delimiter | `=` and `:` both accepted; original preserved on write |
| Comment characters | `#` and `;` |
| Inline comments | Supported |
| Duplicate keys | Aggregated into a JSON array; serialized back as repeated keys |
| Global properties | Keys before the first section stored at root (path: `.key`) |
| Case sensitivity | Case-sensitive (Linux default) |
| Nested sections | Dot notation `[a.b]` recognized as hierarchical path |

---

## systemd

**Detection:** file extension is `.service`, `.target`, `.socket`, `.mount`, `.timer`, `.path`, `.scope`, `.slice`

Reference: [systemd.syntax(7)](https://www.freedesktop.org/software/systemd/man/latest/systemd.syntax.html)

| Feature | Behavior |
|---|---|
| Comment characters | Both `#` and `;` are valid; must appear at the start of the line. Inline comments after a value are **not** supported — they are treated as part of the value. |
| Duplicate keys | Aggregated into an ordered array. Keys listed in the table below support multiple values. |
| Line continuation | A line ending with `\` is joined to the next line; the backslash is replaced by a single space. The visual line break is preserved on write. |
| Empty value semantics | `ExecStart=` (no value) resets the entire list of prior assignments. This is used in drop-in files to clear the base unit's value before re-setting it. |
| Case sensitivity | Case-sensitive |
| Duplicate sections | Not allowed in a single file. Use `.d/` drop-in directories for extension. |

**Keys that support multiple values (duplicate key behavior):**
`ExecStart`, `ExecStartPre`, `ExecStartPost`, `ExecReload`, `ExecStop`, `ExecStopPost`

Note: `ExecStart` allows multiple values only for `Type=oneshot` services.

### Example: Multiple values

```ini
[Service]
ExecStartPre=/usr/bin/setup.sh
ExecStart=/usr/bin/myapp --config /etc/myapp.conf
ExecStart=/usr/bin/myapp --secondary
ExecStopPost=/usr/bin/cleanup.sh
```

```bash
iq '.Service.ExecStart' unit.service
# → ["/usr/bin/myapp --config /etc/myapp.conf", "/usr/bin/myapp --secondary"]

iq '.Service.ExecStart[0]' unit.service
# → "/usr/bin/myapp --config /etc/myapp.conf"
```

### Example: Line continuation

```ini
[Service]
ExecStart=/usr/bin/myapp \
    --config /etc/myapp.conf \
    --verbose
```

```bash
iq '.Service.ExecStart' unit.service
# → "/usr/bin/myapp --config /etc/myapp.conf --verbose"
```

The write-back preserves the visual line breaks.

### Example: Empty-value reset (drop-in pattern)

```ini
# /etc/systemd/system/myapp.service.d/override.conf
[Service]
ExecStart=
ExecStart=/usr/bin/myapp --config /etc/myapp-prod.conf
```

`iq` must preserve the bare `ExecStart=` line as-is and not collapse it. When the query engine sees an empty value in systemd profile, it stores it as an empty string element in the array, not as a deletion.

---

## gitconfig

**Detection:** file is named `.gitconfig` or `.git/config`; or contains `[core]` with `repositoryformatversion`

Reference: [git-config(1)](https://git-scm.com/docs/git-config)

| Feature | Behavior |
|---|---|
| Subsection syntax | `[section "subsection"]` parsed as a two-level path |
| Case sensitivity | Section names and variable names: case-insensitive (normalized to lowercase). Subsection names: **case-sensitive**. |
| Comment characters | `#` and `;` |
| Value continuation | Values can span multiple lines; continuation lines must start with a tab character |
| Boolean shorthands | A key with no `=` value is treated as `true` |
| Variable name characters | Alphanumeric and `-` only. Underscores (`_`) are not valid. Must start with a letter. |

**Subsection name rules:**
- Any character except NUL (`\0`) and newline is allowed in the quoted subsection name.
- Escape sequences: `\"` for a literal double-quote, `\\` for a literal backslash.
- All other backslash sequences drop the backslash (e.g., `\t` → `t`).

**Case folding in `iq` queries:**
When querying a gitconfig file, `iq` normalizes section and variable names to lowercase before matching, but matches subsection names case-sensitively. So `.Remote.Origin.url` and `.remote.origin.url` both resolve to `[remote "origin"]` → `url`, but `.remote.Origin.url` does not.

### Example: Subsection access

```gitconfig
[remote "origin"]
    url = git@github.com:example/repo.git
    fetch = +refs/heads/*:refs/remotes/origin/*

[branch "main"]
    remote = origin
    merge = refs/heads/main
```

```bash
iq '.remote.origin.url' .git/config
# → "git@github.com:example/repo.git"

iq '.branch.main.remote' .git/config
# → "origin"
```

### Example: Subsection name with special characters

```gitconfig
[url "git@github.com:example/repo.git"]
    insteadOf = https://github.com/example/repo.git
```

```bash
iq '.url.["git@github.com:example/repo.git"].insteadOf' .git/config
# → "https://github.com/example/repo.git"
```

Use bracket notation when the subsection name contains characters that would conflict with dot-notation parsing.

---

## AWS Credentials / Config

**Detection:** file path matches `~/.aws/credentials` or `~/.aws/config`; or contains `[default]` with `aws_access_key_id`

Reference: [AWS CLI – Configuration and credential file settings](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)

| Feature | Behavior |
|---|---|
| Profile sections | See normalization rules below |
| Case sensitivity | Key names are case-insensitive |
| Comment characters | `#` and `;` |
| Duplicate keys | Last value wins |

**Profile section naming and normalization:**

The credentials file and config file use different section naming conventions. `iq` normalizes both to the bare profile name so that the same query works regardless of which file is being read:

| File | Section header | `iq` path |
|---|---|---|
| `~/.aws/credentials` | `[default]` | `.default` |
| `~/.aws/credentials` | `[staging]` | `.staging` |
| `~/.aws/config` | `[default]` | `.default` |
| `~/.aws/config` | `[profile staging]` | `.staging` |

The `profile ` prefix (with trailing space) in the config file is a syntactic marker. `iq` strips it during parsing so the profile name itself is the key. `[profile default]` is not a valid header; the default profile is always `[default]`.

**Valid keys by file:**

`~/.aws/credentials`: `aws_access_key_id`, `aws_secret_access_key`, `aws_session_token`

`~/.aws/config`: `region`, `output`, `role_arn`, `source_profile`, `sso_start_url`, `sso_region`, `sso_account_id`, `endpoint_url`, `credential_process`, and others.

### Example: credentials file

```ini
[default]
region = us-east-1

[staging]
region = ap-northeast-1
```

```bash
iq '.staging.region' ~/.aws/credentials
# → "ap-northeast-1"
```

### Example: config file with `[profile ...]`

```ini
[default]
region = us-east-1

[profile staging]
region = ap-northeast-1
output = json
```

```bash
# [profile staging] is normalized to .staging
iq '.staging.region' ~/.aws/config
# → "ap-northeast-1"
```

### Example: safe secret injection

```bash
# Inject a value from environment without it appearing in shell history
iq -i '.staging.aws_secret_access_key = strenv(AWS_SECRET)' ~/.aws/credentials
```

---

## Windows INI

**Detection:** file uses CRLF line endings and/or BOM; or `--profile=windows` is specified explicitly

Reference: [GetPrivateProfileString (Windows API)](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-getprivateprofilestring)

| Feature | Behavior |
|---|---|
| Line endings | CRLF preserved on write |
| BOM | UTF-8 BOM preserved if present |
| Case sensitivity | Case-insensitive — section names and key names are normalized to lowercase |
| Comment characters | `;` at the start of a line. `#` is not a standard Windows INI comment character. |
| Inline comments | Not supported — a `;` within a value line is treated as part of the value, not a comment delimiter |
| Duplicate keys | Last value wins (standard `GetPrivateProfileString` behavior) |

### Example

```ini
[Database]
Host=db.example.com
Port=5432

[Cache]
Host=cache.example.com
; this section is optional
```

```bash
iq '.database.host' app.ini    # section name is case-insensitive
# → "db.example.com"

iq '.Database.Host' app.ini    # equivalent query
# → "db.example.com"
```

**Note on `[Strings]` sections:** The `[Strings]` section is a convention of Windows **INF** files (used for driver installation), not generic INI files. INF files are a specialized INI variant with their own formal specification ([INF Strings Section](https://learn.microsoft.com/en-us/windows-hardware/drivers/install/inf-strings-section)). `iq`'s Windows profile targets generic INI files; INF files are out of scope.
