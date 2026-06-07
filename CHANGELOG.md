# Changelog

## v0.6.0 (2026-06-07)

### Default retention_days changed (30 → 7)

- `retention_days` default reduced from 30 to 7, covering weekend
  scenarios while keeping storage manageable. Synced across all docs,
  example config, test assertions, and generated `init` output.

### Config loading semantics clarified

- **`[protect]` section present** → config file is the source of truth
  for protection rules (no default merging). Users who write `[protect]`
  are actively managing their policy and should be fully respected.
- **No `[protect]` section** → protection rules fall back to built-in
  defaults (safe baseline for users who only configure vault/guard).
- **Vault/Guard field-level merge** — only fields explicitly written in
  the config file override defaults. Other fields keep their defaults.
  Documented in `docs/configuration.md` and `docs/configuration.zh.md`.

### Bug fix: `cmdguard list --json` missing bypass field

- `list --json` previously used hand-written string concatenation that
  omitted the `bypass` field. Now uses `json.Marshal(entries)` for
  correct serialization.

### Unit tests for non-interactive paths

- Added `github.com/creack/pty` dependency for pseudo-terminal tests.
- 8 new tests in `internal/subcmd/subcmd_test.go` covering:
  `isTerminal()` (pipe & TTY), `readLineWithTimeout()` (timeout fires,
  normal input, disabled), `emitNonTTYRejection()` (bypass guidance,
  log creation), `emitNonTTYRejectionTimeout()` (timeout value in
  output).
- All 7 packages pass `go vet ./... && go test ./...`.

### Agent-aware non-interactive handling

- `--bypass=<host>/<platform>/<agent>/<task>` flag for AI agents to
  explicitly confirm operations on protected paths from non-TTY contexts
- Agent env `CMDGUARD_NONINTERACTIVE=1` skips the interactive wait;
  rejection is instant (0 delay). Does NOT grant permission — `--bypass`
  is still required. See `internal/config/env.go` for the contract.
- `bypass` field in audit log entries; `cmdguard list` displays
  `[bypass:...]` tags for traceability
- `--bypass` identifier validation: 4 segments, `[a-zA-Z0-9._-]+` only,
  no angle brackets, no placeholder words (`host`, `platform`, `agent`,
  `task`, `foo`, `todo`, `changeme`, ...)

### Configurable confirmation timeouts (`[guard]` section)

- `confirm_timeout` (default 5s) and `confirm_double_timeout` (default
  10s per step) added to config. Set to 0 to disable timeout entirely.
- stdin read timeout implemented via goroutine + channel + `time.After`.
  Timeout fires → falls back to non-interactive rejection with bypass
  guidance. This solves the "pseudo-TTY no human" problem without
  platform-specific hacks.

### Path existence checks

- `rm` / `chmod` on a non-existent target: cmdguard prints
  `no such file or directory` and exits 1 immediately — no config
  lookup, no log entry, no confirm prompt.
- `mv` to a non-existent destination: cmdguard skips protection and
  executes the underlying `mv` directly (creating a new file needs no
  backup). Existing destinations still trigger the normal protection
  flow.

### Message extraction & i18n

- All user-facing messages extracted to `internal/msg/` package,
  organized by category: `guard.go`, `init.go`, `help.go`, `ops.go`,
  `errors.go`
- All code comments translated to English
- Example config renamed to `example/config.example.toml` (follows the
  `*.example.*` convention used by `.env.example` and the broader
  ecosystem); contents fully English
- TTY prompts improved: `press y to proceed, N or Enter to cancel,
  Ctrl+C to abort (timeout 5s)`
- Invalid bypass and non-TTY rejection guidance show the original
  command with formatted placeholders

### Centralized env var constants

- `internal/config/env.go` defines all env vars cmdguard reads.
  Each constant has a doc comment explaining purpose, use case, and
  safety implications.
- `os.Getenv("CMDGUARD_CONFIG_DIR")` → `os.Getenv(EnvConfigDir)`
- Config file read-only warning banner in `internal/config/config.go`

### Documentation overhaul

- README.md in English (new), README.zh.md for Chinese
- docs/commands.md (English), docs/commands.zh.md (Chinese) — includes
  `--bypass` reference with examples
- docs/configuration.md (English), docs/configuration.zh.md (Chinese) —
  adds `[guard]` section, environment variables reference
- docs/vault.md (English), docs/vault.zh.md (Chinese) — includes
  `bypass` field in the sample log entry

## v0.5.0 (2026-06-06)

- All operations are now logged (previously `allow` was silent)
- Default config: `~/Documents/archive/**` → `~/Documents/**`
- Added `/private/**` to reject (macOS real path)
- Added `~/Desktop/**` to confirm
- Removed all short flags (`-v`, `-h`, `-f`)
- Added `Makefile` (`make build` / `make install` with version injection)

## v0.4.0 (2026-06-06)

- Added `--verbose` flag (detailed execution info)
- Fixed cmdguard flags leaking through to the underlying command
- Release builds no longer embed commit hash in version output
- Split docs: `docs/commands.md`, `docs/configuration.md`, `docs/vault.md`
- Completed doc coverage: `CMDGUARD_CONFIG_DIR`, `--check`, `--verbose`,
  `init --dry-run`, `undo --interactive`, `undo --dry-run`

## v0.3.0 (2026-06-06)

- Added `--check` flag (verify alias is active)
- Added `--dry-run` flag (rm/mv/chmod/init/undo/vault clean)
- Added `--version`/`--help` for guarded commands (shows underlying
  command info too)
- Fixed `findRealCommand` to skip the wrapper bin directory, preventing
  infinite recursion
- Added GitHub Actions: CI (vet + test), Release (cross-compile + publish)

## v0.2.0 (2026-06-06)

- Added `cmdguard init` subcommand (`--force` backs up old files as zip)
- Added `cmdguard list --json` output
- Fixed pipeline mode (table and JSON input)
- `CMGGUARD_CONFIG_DIR` renamed to `CMDGUARD_CONFIG_DIR`
- Improved docs: design rationale, human & agent protection

## v0.1.0 (2026-06-06)

- Initial release
- Four protection levels: reject, confirm_double, confirm, warn
- Wrapped commands: rm, mv, chmod
- Automatic vault backup with undo support
- Audit log (JSON, one file per day)
- TOML config with glob path patterns