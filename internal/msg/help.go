package msg

// ─── Guard help text ────────────────────────────────────────────────

// GuardHelp returns the help text for guarded commands (rm/mv/chmod).
func GuardHelp(cmdName string) string {
	return `cmdguard — command protection tool

Usage:
  ` + cmdName + ` [options] [arguments...]

Options:
  --check       Verify that cmdguard protection is active
  --dry-run     Preview rule matching without executing
  --verbose     Show detailed execution info (matched rule, backup path, actual command)
  --version     Show version info (including underlying command version)
  --help        Show this help (including underlying command help)
  --bypass=<id> Force execution on a protected path (non-interactive)

Protection levels:
  reject           ` + LevelIcons[LevelReject] + `   ` + LevelActions[LevelReject] + `
  confirm_double   ` + LevelIcons[LevelConfirmDbl] + `   ` + LevelActions[LevelConfirmDbl] + `
  confirm          ` + LevelIcons[LevelConfirm] + `   ` + LevelActions[LevelConfirm] + `
  warn             ` + LevelIcons[LevelWarn] + `   ` + LevelActions[LevelWarn] + `

More info: cmdguard help
`
}

// ─── Main help text ─────────────────────────────────────────────────

var MainHelp = `cmdguard — command protection tool

Usage:
  cmdguard <command> [options]

Commands:
  rm <args...>       Run in rm mode (alias: alias rm='cmdguard rm')
                       --check verify guard is active
                       --dry-run preview matching, no execution
                       --verbose show detailed execution info
                       --version show version (including underlying command version)
                       --help show help (including underlying command help)
  mv <args...>       Run in mv mode
  chmod <args...>    Run in chmod mode
  init [--force]     Initialize environment (create dirs, config, wrapper scripts)
                       --force overwrite existing files (old files backed up to ~/.cmdguard/backup/)
                       --dry-run preview without executing
  list [options]     List operation logs
  undo [options]     Restore an operation
                       --dry-run preview files to restore, don't actually restore
  vault clean        Clean expired vault backups
  vault list         List all vault backups
  config             View effective configuration
                       --default  show built-in defaults
                       --raw      show raw config.toml content
                       --bin-dir  print the wrapper directory (machine-readable)
  path               Show cmdguard directory structure
  help, --help       Show this help
  version, --version Show version info

list options:
  --recent N    Last N entries (default 20)
  --since D     Since duration (e.g. "2h", "7d")
  --cmd C       Filter by command (rm/mv/chmod)
  --path P      Filter by path keyword
  --json        Output as JSON

undo options:
  --id ID       Restore by exact ID
  --interactive Interactive selection (default)

vault options:
  --dry-run     List expired backups without deleting (used with "clean")
  --json        Output as JSON (used with "list")

Protection levels:
  reject           ` + LevelIcons[LevelReject] + `   ` + LevelActions[LevelReject] + `
  confirm_double   ` + LevelIcons[LevelConfirmDbl] + `   ` + LevelActions[LevelConfirmDbl] + `
  confirm          ` + LevelIcons[LevelConfirm] + `   ` + LevelActions[LevelConfirm] + `
  warn             ` + LevelIcons[LevelWarn] + `   ` + LevelActions[LevelWarn] + `

Config file: ~/.cmdguard/config.toml
Log dir:     ~/.cmdguard/log/
Vault dir:   ~/.cmdguard/vault/

Environment:
  CMDGUARD_CONFIG_DIR    Custom config directory (default ~/.cmdguard)
  CMDGUARD_NONINTERACTIVE  Skip confirm wait, go straight to rejection
                           (for AI agents / automation; does NOT bypass)
`
