<div align="center">

# cmdguard

**Command protection tool** - guards `rm`, `mv`, `chmod` with path rules, automatic backup, and undo.

[![Go Report Card](https://goreportcard.com/badge/github.com/hyper0x/cmdguard)](https://goreportcard.com/report/github.com/hyper0x/cmdguard)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[简体中文](README.zh.md)

</div>

---

## Overview

cmdguard wraps dangerous commands (`rm`, `mv`, `chmod`) to prevent accidental data loss.

- 🚫 **Three protection levels** - reject, guarded, allow
- 💾 **Automatic backup** - files are copied to a vault before destructive ops
- ↩️ **Undo** - restore deleted or overwritten files via `cmdguard undo`
- 📋 **Audit log** - every operation is recorded permanently; searchable & filterable
- 🔑 **Bypass with audit** - guarded paths can be executed with a `--bypass` identifier that is permanently logged
- ⚙️ **TOML config** - glob path patterns, per-command overrides

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
```

`cmdguard config --bin-dir` prints the wrapper directory as a bare
path (default `~/.cmdguard/bin`, or wherever your install puts it),
so the export above stays correct even if you customise the layout.

Both methods can coexist.

### What cmdguard does NOT do

cmdguard cannot intercept calls that bypass `PATH` lookup, e.g. `/bin/rm /etc/passwd`.
The model is "protect + audit + recoverable", not "absolute lockdown".

---

## Bypass identifiers

When a command targets a **guarded** path, cmdguard rejects it unless the
caller provides a `--bypass` identifier:

```bash
rm /path/to/file --bypass=<platform>/<agent>/<task>
```

The identifier must have exactly 3 segments:

| segment  | meaning                  | example                |
|----------|--------------------------|------------------------|
| platform | agent platform           | `qwenpaw`, `cursor`    |
| agent    | agent id or `manual`     | `ai_research`          |
| task     | brief task slug          | `cleanup-tmp-dirs`     |

Allowed characters: `[a-zA-Z0-9._-]`. Empty segments, angle brackets,
and template placeholder words (`agent`, `task`, `xxx`, `foo`, `todo`,
...) are rejected.

Every bypass is recorded in the audit log with the full identifier, so
the audit trail attributes every protected-path operation back to a
specific agent / task.

See [docs/commands.md](docs/commands.md#--bypass) for full details.

---

## Quick start

### Install

```bash
# Option 1 - pre-built binary (recommended)
# Download from the Releases page for your platform.

# Option 2 - build from source
git clone https://github.com/hyper0x/cmdguard.git
cd cmdguard
make install   # installs to $GOBIN with version info baked in
```

### Initialize

```bash
cmdguard init
```

Creates `~/.cmdguard/{config.toml, bin/, log/, vault/}` and prints the
integration guide. Idempotent - re-running is safe. Use `--force` to
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
reject  = ["/etc/**", "/private/**", "~/.ssh/**"]
guarded = ["~/.config/**", "~/Documents/**", "~/Desktop/**"]

[vault]
retention_days = 7
auto_purge     = true
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

**What's the difference between `reject` and `guarded`?**
`reject` always blocks, even with `--bypass`. `guarded` blocks without
`--bypass`, but allows execution with a valid `--bypass` identifier
(after backup).

**What if my config file gets overwritten?**
`cmdguard init` never overwrites without `--force`. With `--force`, the
old file is preserved at `~/.cmdguard/backup/init-<timestamp>.zip`.

**How do I uninstall?**
```bash
# Remove the alias / PATH line from your shell rc, then:
trash ~/.cmdguard         # or rm -rf if you don't have trash
rm $(which cmdguard)
```

---

## License

[MIT](LICENSE)
