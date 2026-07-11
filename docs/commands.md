# Command reference

## `cmdguard rm/mv/chmod <args...>`

Run a command through cmdguard. Invoked automatically via alias or PATH
hijack; can also be called directly.

```bash
# Direct
cmdguard rm -rf ~/Downloads/temp

# Via alias
alias rm='cmdguard rm'
rm -rf ~/Downloads/temp
```

### Special flags

| Flag | Description |
|:----|:----|
| `--check` | Verify the cmdguard wrapper is active (does not execute the real command) |
| `--dry-run` | Preview the rule-matching result, do not execute |
| `--verbose` | Print detailed execution info (matched rule, backup path, actual command) |
| `--version` | Show cmdguard version; also attempt to show the underlying command's version (GNU tools work, BSD silently skipped) |
| `--help` | Show cmdguard help (with protection levels); also attempt to show the underlying command's help |
| `--bypass=<id>` | Force execution of a guarded path (see [`--bypass`](#--bypass) below) |

### Examples

```bash
# Verify the alias is active
rm --check
# [cmdguard] guard active - rm is running through cmdguard

# Show version
rm --version
# cmdguard 0.6.0
# rm (GNU coreutils) 9.2 ...

# Preview rule matching
rm --dry-run -rf ~/Downloads/temp

# Detailed execution info
rm --verbose -rf ~/Downloads/temp

# Help
rm --help
```

---

## `--bypass`

**Purpose:** Let an AI agent (or any caller) explicitly declare: "I
have reviewed this operation and confirm it is safe - proceed."

**Format:** `--bypass=<platform>/<agent>/<task>` - exactly 3
slash-separated segments:

| Segment  | Meaning                       | Example                |
|----------|-------------------------------|------------------------|
| platform | agent platform                | `qwenpaw`, `cursor`, `claude-code` |
| agent    | agent id or `manual`          | `ai_research`, `coding` |
| task     | brief task slug               | `cleanup-tmp-dirs`     |

**Validation rules:**

- Exactly 3 segments
- Each segment matches `[a-zA-Z0-9._-]+`, no empty segments
- Total length ≥ 10 characters
- No segment may be a template placeholder (`platform`, `agent`,
  `task`, `xxx`, `foo`, `todo`, `changeme`, ...)
- No angle brackets `<>` or curly braces `{}` (to defeat verbatim
  template copies)

**Examples:**

```bash
# ✅ Valid
rm /tmp/cache --bypass=qwenpaw/ai_research/cleanup-cache
mv old.txt /Users/x/Documents/new.txt --bypass=cursor/default/refactor
rm ~/Downloads/old.zip --bypass=manual/haolin/cleanup-downloads

# ❌ Rejected (template copied verbatim)
rm /tmp/cache --bypass='<platform>/<agent>/<task>'
rm /tmp/cache --bypass=platform/agent/task

# ❌ Rejected (placeholder words / wrong format)
rm /tmp/cache --bypass=xxx/yyy/zzz
rm /tmp/cache --bypass=abc/def        # too few segments
```

**How it works:** When a command targets a **guarded** path and no
`--bypass` is provided, cmdguard rejects the operation immediately
and prints guidance showing the expected `--bypass` format. With a
valid `--bypass`, cmdguard backs up the target files to the vault,
logs the operation with the full bypass identifier, then executes.

**Backup failure is always fatal.** If the vault backup fails, the
operation is aborted (exit 1) regardless of whether the caller is an
agent or a human. The undo safety net is broken; proceeding without
rollback capability is irresponsible.

**Audit:** every bypass is recorded. `cmdguard list` shows
`[bypass:qwenpaw/ai_research/cleanup-cache]` so the audit trail
attributes the operation back to a specific agent/task.

---

## `cmdguard init [--force] [--dry-run]`

Initialize the environment. Idempotent and safe to re-run.

| Flag | Description |
|:----|:----|
| `--force` | Overwrite existing config and wrapper scripts (old files zipped to `~/.cmdguard/backup/init-<timestamp>.zip`) |
| `--dry-run` | Preview the operations, do not execute |

```bash
cmdguard init                    # first-time init
cmdguard init --force            # force overwrite
cmdguard init --dry-run          # preview
cmdguard init --force --dry-run  # preview with force
```

---

## `cmdguard list [options]`

List audit log entries.

| Flag | Description |
|:----|:----|
| `--recent N` | Last N entries (default 20) |
| `--since D` | Since duration (e.g. `"2h"`, `"7d"`) |
| `--cmd C` | Filter by command (`rm`/`mv`/`chmod`) |
| `--path P` | Filter by path keyword |
| `--json` | JSON output (pipeline-friendly) |

```bash
cmdguard list                    # last 20
cmdguard list --recent 50        # last 50
cmdguard list --since 7d         # last 7 days
cmdguard list --cmd rm           # only rm
cmdguard list --path Documents   # by path keyword
cmdguard list --json             # JSON
```

---

## `cmdguard undo [options]`

Restore an operation.

| Flag | Description |
|:----|:----|
| `--id ID` | Restore by ID (short prefix supported) |
| `--interactive` | Interactive picker |
| `--dry-run` | Preview the files to restore, do not actually restore |

```bash
cmdguard undo --id abc123              # by ID
cmdguard undo --interactive            # picker
cmdguard undo --id abc123 --dry-run    # preview
cmdguard list --json | cmdguard undo   # pipeline
```

---

## `cmdguard vault list [--json]`

List all backups currently stored in the vault.

| Flag | Description |
|:----|:----|
| `--json` | Emit a JSON array (machine-readable) |

```bash
cmdguard vault list
cmdguard vault list --json
```

The default text output shows ID, command, creation time, expiration,
and the original target path. Sorted newest first.

---

## `cmdguard vault clean [--dry-run]`

Purge expired vault backups.

| Flag | Description |
|:----|:----|
| `--dry-run` | List what would be deleted, do not actually delete |

```bash
cmdguard vault clean              # purge
cmdguard vault clean --dry-run    # preview
```

Bare `cmdguard vault` (no subcommand) prints a usage block and
exits 1 - `clean` is destructive enough that it should never be the
silent default.

---

## `cmdguard config [--default | --raw | --bin-dir]`

Print configuration views. The flags are mutually exclusive.

| Flag | Description |
|:----|:----|
| _(none)_ | Effective merged config (defaults + your overrides) |
| `--default` | Built-in defaults verbatim |
| `--raw` | Raw `config.toml` contents from disk |
| `--bin-dir` | Bare path to the wrapper-script directory |

```bash
cmdguard config              # what's actually being used
cmdguard config --default    # what you'd get with no config.toml
cmdguard config --raw        # what's literally on disk
cmdguard config --bin-dir    # /home/alice/.cmdguard/bin
```

`--bin-dir` is designed for shell composition:

```bash
export PATH="$(cmdguard config --bin-dir):$PATH"
```

---

## `cmdguard path`

Print cmdguard's directory layout - config file, log dir, vault
dir, bin dir - with file counts and sizes.

```bash
cmdguard path
```

The log section caps display at the 5 newest files to keep output
bounded on long-running installs. A summary line shows the total
file count.

---

## `cmdguard help` / `cmdguard version`

Self-explanatory.
