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
> prune it - your safety policy must reflect exactly what you wrote.

## Basic structure

```toml
[protect]
reject = [
  "/etc/**",
  "~/.ssh/**",
]
guarded = [
  "~/.config/**",
  "~/Documents/**",
  "~/Desktop/**",
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
# (global rules set ~/.config/** to guarded).
[protect.command.chmod]
reject = [
  "~/.config/**",
]

[vault]
retention_days = 7
auto_purge = true
```

> ⚠️ **Source of truth:** When your config file defines a `[protect]`
> section, the file's rules are used **as-is** - built-in defaults are
> NOT merged on top. If you want no reject rules, write `reject = []`.
>
> If the file has **no** `[protect]` section at all, built-in defaults
> apply (so a minimal file that only sets `[vault]` still gets the
> default protect rules).
>
> `[vault]` uses **field-level merge**: only the keys you explicitly
> write override the defaults; omitted keys keep their built-in default
> values.

## Path patterns

Glob-style patterns are supported:

| Pattern | Meaning | Example |
|:----|:----|:----|
| `**` | matches any depth | `/etc/**` matches everything under `/etc/` |
| `*` | matches any chars in a single segment | `*.key` matches all `.key` files |
| `?` | matches a single char | `file.???` matches `file.txt`, not `file.html` |
| `~` | expanded to the user's home dir | `~/.ssh/**` -> `/Users/xxx/.ssh/**` |

## Default config

The config generated on first run / `cmdguard init` contains:

- **reject:** system directories (`/bin/`, `/etc/`, `/usr/`, `/private/`, ...), key files (`*.key`, `*.pem`, ...), critical home configs (`~/.ssh/`, `~/.gnupg/`)
- **guarded:** home app data (`~/.config/`, `~/.local/share/`), Documents (`~/Documents/`), Desktop (`~/Desktop/`)
- **allow:** everything else (by default - paths not matching any rule)

## Protection levels

| Level | Icon | Behaviour | When to use |
|:----|:---:|:----|:--------|
| `reject` | 🚫 | Always refused, even with `--bypass`. Logged. | System dirs, key files, non-recoverable configs |
| `guarded` | 🔒 | Without `--bypass`: rejected + logged. With valid `--bypass`: backup -> log -> execute. | App data dirs, documents - cleanable but high-risk |
| `allow` | ✅ | Executed directly. Logged. | Everything not matching a rule |

**Rule order:** `reject -> guarded`. First match wins. Paths not matching any rule are allowed by default.

## Environment variables

| Variable | Purpose |
|:--------|:----|
| `CMDGUARD_CONFIG_DIR` | Overrides the config directory (which contains `config.toml`, `log/`, `vault/`, `bin/`). Default: `~/.cmdguard` |

cmdguard no longer uses any behavior-controlling environment variables.
The old `CMDGUARD_AGENT_MODE` and `CMDGUARD_NONINTERACTIVE` variables
are silently ignored - wrappers no longer set them, and the code no
longer reads them. If you have existing wrappers that export these
variables, they will still work (the variables are simply ignored).
Run `cmdguard init --force` to regenerate clean wrappers.

## Migration from pre-v0.14 config

If you have an existing `config.toml` with the old `[guard]` section
or old protection levels, here's what changed:

| Old | New | Notes |
|:----|:----|:------|
| `confirm_double` | `guarded` | Rename the key in `[protect]` |
| `confirm` | `guarded` | Merge into the same `guarded` list |
| `warn` | (removed) | Remove these paths or move to `guarded` if you want backup |
| `[guard]` section | (removed) | The entire section is ignored; no interactive settings exist |
| `CMDGUARD_AGENT_MODE` | (removed) | No longer needed; non-interactive is the only mode |
| `CMDGUARD_NONINTERACTIVE` | (removed) | Same as above |

The TOML decoder silently ignores unknown keys, so your old config file
will not cause errors - but the old keys have no effect. `cmdguard init
--force` will write a clean config with the new structure.
