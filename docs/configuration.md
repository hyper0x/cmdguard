# Configuration

Path: `~/.cmdguard/config.toml`

> **Environment variable:** `CMDGUARD_CONFIG_DIR` overrides the config
> directory (containing `config.toml`, `log/`, `vault/`, `bin/`).
> All cmdguard paths resolve under this directory when set.
>
> **Read-only contract:** the config file is **read-only** from cmdguard's
> perspective. cmdguard never modifies, rewrites, or auto-cleans it at
> runtime. Only `cmdguard init --force` may overwrite it (the previous
> file is preserved in `~/.cmdguard/backup/init-<timestamp>.zip`). If a
> path listed in the config no longer exists, cmdguard does NOT silently
> prune it — your safety policy must reflect exactly what you wrote.

## Basic structure

```toml
[protect]
reject = [
  "/etc/**",
  "~/.ssh/**",
]
confirm_double = [
  "~/.config/**",
]
confirm = [
  "~/Documents/**",
  "~/Desktop/**",
]
warn = [
  "~/Downloads/**",
]

# Per-command overrides
[protect.command.rm]
reject = [
  "~/.config/**",
]

# mv protects the destination (last argument). Reject mv into
# ~/Downloads/** to prevent accidental overwrites.
[protect.command.mv]
reject = [
  "~/Downloads/**",
]

# chmod changes permissions. Reject for app config directories
# (global rules set ~/.config/** to confirm_double).
[protect.command.chmod]
reject = [
  "~/.config/**",
]

[vault]
retention_days = 7
auto_purge = true

[guard]
confirm_timeout = 5          # seconds; 'confirm' prompt
confirm_double_timeout = 10  # seconds per step; 'confirm_double' prompt
```

> ⚠️ **Source of truth:** When your config file defines a `[protect]`
> section, the file's rules are used **as-is** — built-in defaults are
> NOT merged on top. If you want no reject rules, write `reject = []`.
>
> If the file has **no** `[protect]` section at all, built-in defaults
> apply (so a minimal file that only sets `[vault]` still gets the
> default protect rules).
>
> `[vault]` and `[guard]` use **field-level merge**: only the keys you
> explicitly write override the defaults; omitted keys keep their
> built-in default values.

## Path patterns

Glob-style patterns are supported:

| Pattern | Meaning | Example |
|:----|:----|:----|
| `**` | matches any depth | `/etc/**` matches everything under `/etc/` |
| `*` | matches any chars in a single segment | `*.key` matches all `.key` files |
| `?` | matches a single char | `file.???` matches `file.txt`, not `file.html` |
| `~` | expanded to the user's home dir | `~/.ssh/**` → `/Users/xxx/.ssh/**` |

## Default config

The config generated on first run / `cmdguard init` contains:

- **reject:** system directories (`/bin/`, `/etc/`, `/usr/`, `/private/`, ...), key files (`*.key`, `*.pem`, ...), critical home configs (`~/.ssh/`, `~/.gnupg/`, `~/.aws/`)
- **confirm_double:** home app data (`~/.config/`, `~/.local/share/`)
- **confirm:** Documents (`~/Documents/`), Desktop (`~/Desktop/`)
- **warn:** Downloads (`~/Downloads/`)

## Protection levels

| Level | Icon | Behaviour | When to use |
|:----|:---:|:----|:--------|
| `reject` | 🚫 | Refused outright, not executed | System dirs, key files, non-recoverable configs |
| `confirm_double` | 🔒 | Warn + double confirmation (type `yes`) → backup → execute | App data dirs — cleanable but high-risk |
| `confirm` | ❓ | Warn + single confirmation (press `y`) → backup → execute | Documents and other occasional-cleanup paths |
| `warn` | ⚠️ | Warn + backup → execute | Downloads and other temp dirs |

**Rule order:** `reject → confirm_double → confirm → warn`. First match wins.

## `[guard]` — interactive timeouts

```toml
[guard]
confirm_timeout = 5          # default 5s
confirm_double_timeout = 10  # default 10s per step
```

| Key | Default | Meaning |
|:---|:-----:|:----|
| `confirm_timeout` | `5` | Seconds to wait for input at the `confirm` prompt |
| `confirm_double_timeout` | `10` | Seconds to wait **per step** at `confirm_double` (one step for pressing `y`, one step for typing `yes`) |

**Special value:**
- `0` = disable timeout, wait forever (until the user presses a key or
  hits Ctrl+C). Only sensible on personal machines where no automation
  will ever hit a confirm path; otherwise a forgotten agent invocation
  may hang indefinitely.

**What happens when the timeout fires:** cmdguard falls back to the
non-interactive rejection path, prints the same `--bypass` guidance as
`CMDGUARD_NONINTERACTIVE=1` / non-TTY stdin, and logs `timeout waiting
for confirmation (Xs)`.

**Why a timeout exists:** `isTerminal()` can be fooled by agent sandboxes
that allocate a pseudo-TTY for the subprocess. Without a timeout, the
process would hang forever at the `Are you sure?` prompt in those
"absent human" scenarios. The timeout is the safety net.

## Environment variables

| Variable | Purpose |
|:--------|:----|
| `CMDGUARD_CONFIG_DIR` | Overrides the config directory (which contains `config.toml`, `log/`, `vault/`, `bin/`). Default: `~/.cmdguard` |
| `CMDGUARD_NONINTERACTIVE` | Any non-empty value: skip the interactive wait at `confirm`/`confirm_double` prompts and go straight to the reject+bypass path. **Does NOT grant permission** — `--bypass=<id>` is still required to proceed |

`CMDGUARD_NONINTERACTIVE` is intended for AI agents and CI. Once set,
rejection is instant (0 latency), no 5/10s wait. See
[docs/commands.md](commands.md#--bypass) for the bypass identifier format.
