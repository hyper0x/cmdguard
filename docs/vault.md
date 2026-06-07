# Vault & undo

cmdguard maintains its own vault — it does NOT rely on the OS trash.

## How it works

1. For `confirm` / `confirm_double` / `warn` operations, target files
   are copied to the vault **before** the underlying command runs.
2. Backup directory: `~/.cmdguard/vault/<timestamp>_<id>/`
3. The backup is a full copy of the original file plus metadata.
4. After the original command succeeds, the vault copy is the source
   of truth for `cmdguard undo`.
5. `reject`ed operations never write to the vault (nothing happened),
   but they DO get a log entry.

## Why not the OS trash?

- macOS Trash cannot handle `mv` overwrite scenarios
- Multiple deletions of same-named files get confused in Trash
- Trash has no metadata linking it to the audit log
- Cross-platform consistency: same vault mechanism on Linux/macOS/Windows

## Auto-purge

- `retention_days`: how long backups are kept (default 7)
- `auto_purge`: purge expired backups on every guarded command
  (default `true`)
- Logs are **kept forever** (small files, valuable for audit)

Manual purge:

```bash
cmdguard vault clean              # purge expired
cmdguard vault clean --dry-run    # preview
```

## Undo

`cmdguard undo` restores files from the vault:

```bash
# By ID (short prefix is enough)
cmdguard list                          # find the ID
cmdguard undo --id abc12345

# Interactive picker
cmdguard undo --interactive

# Preview
cmdguard undo --id abc12345 --dry-run

# Pipeline (list → undo)
cmdguard list --json | cmdguard undo
```

### Per-command restore behaviour

| Original op | Restore behaviour |
|:------------|:------------------|
| `rm file`     | Copy the file from vault back to its original path |
| `mv src dst`  | Restore the destination from vault (the source is NOT moved back automatically) |
| `chmod file`  | Restore the original permissions |

## Audit log

Every operation that goes through cmdguard is logged to
`~/.cmdguard/log/` — including rejected, allowed, bypassed, undo
itself, and vault-clean events.

- Format: JSON Lines, one file per day (`YYYY-MM-DD.jsonl`)
- Retention: **permanent** (independent of vault retention)
- Fields: id, timestamp, command, action, targets, matched rule,
  message, bypass identifier

```json
{
  "id": "abc123456789",
  "timestamp": "2026-06-06T12:00:00+08:00",
  "command": "rm",
  "action": "confirm",
  "targets": "/Users/x/Documents/old.txt",
  "rule": "/Users/x/Documents/**",
  "message": "path matches protection rule '/Users/x/Documents/**'",
  "bypass": "mac-studio/qwenpaw/ai_research/cleanup-cache"
}
```

The `bypass` field attributes the operation to a specific agent and task.
