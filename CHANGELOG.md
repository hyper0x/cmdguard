# Changelog

## v0.7.0 (2026-06-19)

A reliability and correctness release. No new user-facing features
beyond two convenience subcommands; the bulk of the work is fixing
subtle bugs and tightening invariants found during a five-round
release sweep.

### New subcommands

#### `cmdguard vault list [--json]`

Lists every backup currently in the vault: ID, command, age,
expiration, and the original target path. Complements `vault clean`
(which already showed expired backups via `--dry-run`) by giving
users a full inventory.

```text
$ cmdguard vault list
[cmdguard] 2 backup(s) (vault size: 11.2 KB)
ID            Command  Created           Expires           Target
abc123def456  rm       2026-06-19 09:01  in 6d 23h         /etc/important.conf
fed987cba210  mv       2026-06-19 10:14  in 6d 23h         /opt/data/old.bin
```

`--json` emits a machine-readable form for tooling.

#### `cmdguard path`

Prints cmdguard's directory layout — config file, log dir, vault
dir, bin dir — with file counts and sizes. Replaces the implicit
"go look in `~/.cmdguard`" knowledge users used to need:

```text
$ cmdguard path
[cmdguard] cmdguard paths

Config dir:  /home/alice/.cmdguard
Config file: /home/alice/.cmdguard/config.toml (327 B, 24 lines)

Log dir:     /home/alice/.cmdguard/log
  - 2026-06-19.log (4.1 KB)
  - 2026-06-18.log (1.8 KB)
  (2 file(s))

Vault dir:   /home/alice/.cmdguard/vault (11.2 KB, 2 backup(s))
Bin dir:     /home/alice/.cmdguard/bin
  - rm, mv, chmod (3 file(s))
```

The log section caps display at 5 newest files to keep output
bounded on long-running installs.

### `cmdguard config` enhancements

- `--default` shows the built-in defaults verbatim (what you get if
  you never wrote `~/.cmdguard/config.toml`).
- `--raw` dumps the raw `config.toml` content with a header showing
  the path. Equivalent to `cat ~/.cmdguard/config.toml` but keeps
  the user inside the cmdguard CLI when checking what's actually
  on disk vs. what's effective.
- `--bin-dir` prints just the wrapper-script directory as a bare
  path. Designed for shell composition: the integration guide now
  recommends `export PATH="$(cmdguard config --bin-dir):$PATH"`
  instead of asking users to hardcode `~/.cmdguard/bin`.

The three flags are mutually exclusive with each other and with
the bare `cmdguard config` (which still shows the merged effective
view).

### Critical fix: wrapper recursion (P0)

`PATH=$HOME/.cmdguard/bin:...` was the documented integration. With
v0.6.0 binaries, the wrapper at `~/.cmdguard/bin/rm` would invoke
`cmdguard guard rm <args>`, which used a basename search to find
the "real" `rm`. If `cmdguard` itself was earlier in PATH than
`/bin`, that basename search returned the wrapper again, and the
process forked into infinite recursion until OS limits killed it.

Fix: the wrapper template now contains a sentinel comment
`# cmdguard:wrapper:v1` on its first line. `findRealCommand` reads
the first 256 bytes of each candidate and skips any file containing
the sentinel. Detection is purely textual — no SameFile chains, no
hardcoded paths — so it works regardless of where cmdguard or the
wrappers live.

Existing v0.6.0 deployments can be repaired by running
`cmdguard init --force` (recreates wrappers with sentinel) or
manually appending the sentinel line to each wrapper script. The
sentinel is a comment, so older v0.6.0 binaries treat upgraded
wrappers as harmless.

Test coverage: `TestIsCmdguardWrapper` (8 sub-tests) exercises
sentinel-present, sentinel-absent, broken-shebang, large-file,
no-read-permission, and not-a-regular-file cases.

### Other fixes

- **`mv ... existing-dir/` lost the destination's existing files.**
  When the destination directory already contained files that
  matched basenames of the source files, `mv` overwrote them with
  no warning and no backup. Now backed up before the rename, with
  log entries pointing at the clobbered originals.
- **Vault same-basename collisions.** Backups are now keyed by full
  original path (recorded in `manifest.json`), not just basename.
  Restoring `/etc/foo.conf` no longer accidentally overwrites
  `/opt/foo.conf` or vice versa. Legacy basename-keyed backups
  (pre-manifest) still restore correctly via fallback.
- **`matchPath` `/**` boundary.** `/etc/**` previously matched
  `/etcd/foo` because of a missing path-separator check. Fuzz tests
  added to lock the invariant.
- **Audit log durability.** `f.Sync()` after every audit-log write,
  so a crash or power loss between operations doesn't lose entries
  that the user has already seen confirmation for.
- **Vault file permissions preserved.** Backup files retained the
  default 0600 permissions; on restore, that didn't always match
  the original file's mode. Now stored alongside the file in the
  manifest and applied on restore.
- **Log entry IDs.** Switched from time-based to `crypto/rand`
  generation. Theoretical-only collision risk, but cheap to fix.
- **Double-formatted error messages.** A handful of error paths
  ran `Sprintf` once in the message constant and once at the call
  site, producing strings like `"failed: %v: <error>"`. Cleaned up
  by introducing `errExit` / `errPrint` / `warn` helpers that own
  the `[cmdguard]` tag and `error:` / `warning:` prefix; templates
  no longer embed those affixes.
- **`cmdguard list --since 7days`** now errors out with
  `invalid --since value "7days" (use formats like 30m, 2h, 7d)`.
  Previously it parsed to zero and silently returned the entire
  unfiltered log.
- **Unknown flags rejected.** `list`, `config`, `init`, `path`,
  `vault clean`, and `vault list` now exit 1 with
  `unknown flag "..."` on any unrecognised argument. Catches
  typos like `--recnet 5` (intended `--recent 5`) that previously
  silently fell through to defaults. The guarded commands
  (`rm`/`mv`/`chmod`) deliberately keep their permissive
  passthrough — they need to forward `-rf`, `-R`, etc. to the
  real binary.
- **`cmdguard vault` without a subcommand** now prints a usage
  block and exits 1, instead of running `clean`. A destructive
  verb shouldn't be the silent default.
- **`undo` error messages route to stderr.** Six call sites
  (no-args usage, ID-not-found, expired, rejected, backup-not-found,
  per-file restore failure) moved from `fmt.Println` to
  `fmt.Fprintln(os.Stderr, ...)`. Success-path messages
  (`restored N file(s)`, dry-run header) stay on stdout.

### Internal cleanup

- Centralised the protected-command list in `internal/subcmd/consts.go`
  via a `GuardedCommands()` function. Single source of truth; was
  previously duplicated across `init`, `guard`, and the wrapper
  template.
- `findRealCommand` rewritten: `filepath.SplitList(PATH)` instead of
  manual splitting, early-skip for cmdguard's own bin dir, memoised
  `os.Stat` of self.
- Extracted `shouldFallThroughForMissing`, `autoPurgeIfEnabled`,
  and `appendLog` helpers from `RunGuard` to reduce a 200-line
  function to digestible pieces.
- Removed dead code (unused `formatTargets`, etc.) and tests
  with hardcoded `/Users/<name>` paths.
- Fixed a file-handle leak in `backupFiles` on the rare error
  path where `io.Copy` failed mid-stream.
- `placeholderTokens` for `--bypass` validation tightened: previous
  list missed a few common stand-ins (`changeme`, `todo`).

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