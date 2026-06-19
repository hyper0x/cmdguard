<div align="center">

# cmdguard

**Command protection tool** — guards `rm`, `mv`, `chmod` with confirmation, automatic backup, and undo.

[![Go Report Card](https://goreportcard.com/badge/github.com/hyper0x/cmdguard)](https://goreportcard.com/report/github.com/hyper0x/cmdguard)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[简体中文](README.zh.md)

</div>

---

## Overview

cmdguard wraps dangerous commands (`rm`, `mv`, `chmod`) to prevent accidental data loss.

- 🚫 **Four protection levels** — reject, confirm_double, confirm, warn
- 💾 **Automatic backup** — files are copied to a vault before destructive ops
- ↩️ **Undo** — restore deleted or overwritten files via `cmdguard undo`
- 📋 **Audit log** — every operation is recorded permanently; searchable & filterable
- 🤖 **Agent-aware** — explicit handling for AI agents and automation (see below)
- ⚙️ **TOML config** — glob path patterns, per-command overrides

---

## Design

### Protecting humans

Add aliases to `~/.zshrc` or `~/.bashrc`:

```bash
alias rm='cmdguard rm'
alias mv='cmdguard mv'
alias chmod='cmdguard chmod'
```

### Protecting AI agents

Put cmdguard's wrapper directory at the front of `PATH` so the agent's
`rm`/`mv`/`chmod` lookups hit cmdguard's wrappers:

```bash
export PATH="$(cmdguard config --bin-dir):$PATH"
export CMDGUARD_NONINTERACTIVE=1   # skip the 5s/10s confirm wait
```

`cmdguard config --bin-dir` prints the wrapper directory as a bare
path (default `~/.cmdguard/bin`, or wherever your install puts it),
so the export above stays correct even if you customise the layout.

Both methods can coexist.

### What cmdguard does NOT do

cmdguard cannot intercept calls that bypass `PATH` lookup, e.g. `/bin/rm /etc/passwd`.
The model is "protect + audit + recoverable", not "absolute lockdown".

---

## AI agent usage

When an AI agent invokes a guarded command and the target is a protected
path, cmdguard refuses to hang on an interactive prompt. The agent must:

1. **Set the env var** to declare itself non-interactive:
   ```bash
   export CMDGUARD_NONINTERACTIVE=1
   ```
   This skips the 5s/10s wait at confirm prompts. It does **NOT** grant
   permission — the operation is still rejected.

2. **If the operation is genuinely safe**, retry with a `--bypass` identifier:
   ```bash
   rm /path/to/file --bypass=<host>/<platform>/<agent>/<task>
   ```
   The identifier must have exactly 4 segments:

   | segment  | meaning                       | example                |
   |----------|-------------------------------|------------------------|
   | host     | machine hostname / alias      | `mac-studio`           |
   | platform | agent platform                | `qwenpaw`, `cursor`    |
   | agent    | agent id                      | `ai_research`          |
   | task     | brief task slug               | `cleanup-tmp-dirs`     |

   Allowed characters: `[a-zA-Z0-9._-]`. Empty segments, angle brackets,
   and template placeholder words (`host`, `agent`, `task`, `xxx`, `foo`,
   `todo`, ...) are rejected.

Every bypass is recorded in the audit log with the full identifier, so
the audit trail attributes every protected-path operation back to a
specific agent / task.

See [docs/commands.md](docs/commands.md#--bypass) for full details.

---

## Quick start

### Install

```bash
# Option 1 — pre-built binary (recommended)
# Download from the Releases page for your platform.

# Option 2 — build from source
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
make install   # installs to $GOBIN with version info baked in
```

### Initialize

```bash
cmdguard init
```

Creates `~/.cmdguard/{config.toml, bin/, log/, vault/}` and prints the
integration guide. Idempotent — re-running is safe. Use `--force` to
overwrite (old files are zipped to `~/.cmdguard/backup/`).

---

## Commands at a glance

| Command | Description |
|:----|:----|
| `rm/mv/chmod <args...>` | Run a command through cmdguard |
| `init [--force] [--dry-run]` | Initialize the environment |
| `list [options]` | List audit log entries |
| `undo [options]` | Restore an operation from vault |
| `vault list [--json]` | List all vault backups |
| `vault clean [--dry-run]` | Purge expired vault backups |
| `config [--default \| --raw \| --bin-dir]` | Print configuration views |
| `path` | Show the cmdguard directory layout |
| `help` / `version` | Self-explanatory |

Full reference: [docs/commands.md](docs/commands.md)

---

## Configuration

Location: `~/.cmdguard/config.toml` (overridable via `CMDGUARD_CONFIG_DIR`).

```toml
[protect]
reject         = ["/etc/**", "/private/**", "~/.ssh/**"]
confirm_double = ["~/.config/**"]
confirm        = ["~/Documents/**", "~/Desktop/**"]
warn           = ["~/Downloads/**"]

[vault]
retention_days = 7
auto_purge     = true

[guard]
confirm_timeout        = 5    # seconds; 'confirm' prompt
confirm_double_timeout = 10   # seconds per step; 'confirm_double' prompt
```

> **The config file is read-only from cmdguard's perspective.** cmdguard
> never modifies it at runtime; only `cmdguard init --force` may overwrite
> it (with a backup zip).

Full reference: [docs/configuration.md](docs/configuration.md)

---

## Vault & undo

- Backup directory: `~/.cmdguard/vault/<timestamp>_<id>/`
- Retention: 30 days by default, configurable
- Restore: `cmdguard undo [--id <id>] [--interactive] [--dry-run]`
- Logs: permanent, JSON, one file per day

Details: [docs/vault.md](docs/vault.md)

---

## Build from source

```bash
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
make build    # build into ./dist with version from git tag
make install  # install to $GOBIN
```

---

## FAQ

**Can cmdguard be bypassed?**
Yes. `/bin/rm /etc/passwd` skips the PATH lookup entirely. cmdguard is
protection + audit + recovery, not an absolute lockdown.

**What's the difference between `confirm` and `confirm_double`?**
`confirm` requires a single `y`. `confirm_double` requires `y` first and
then typing the full word `yes` — designed to defeat fatigue errors on
high-risk paths.

**What if my config file gets overwritten?**
`cmdguard init` never overwrites without `--force`. With `--force`, the
old file is preserved at `~/.cmdguard/backup/init-<timestamp>.zip`.

**Where do agents put `CMDGUARD_NONINTERACTIVE`?**
In the agent's shell init or wherever the agent's environment is set up.
Once exported, every cmdguard call in that environment skips the wait
and goes straight to the bypass-or-reject path.

**How do I uninstall?**
```bash
# Remove the alias / PATH line from your shell rc, then:
trash ~/.cmdguard         # or rm -rf if you don't have trash
rm $(which cmdguard)
```

---

## License

[MIT](LICENSE)
