package msg

// ─── Init command output ────────────────────────────────────────────

const (
	// InitDryRunHeader is shown at the start of init --dry-run.
	InitDryRunHeader = TagCmdguard + " will perform the following operations:"

	// InitDryRunCreateDir is the dry-run line for creating a directory.
	InitDryRunCreateDir = "  ✓ Create directory %s"

	// InitDryRunCreateFile is the dry-run line for creating a file.
	InitDryRunCreateFile = "  ✓ Create config file %s"

	// InitDryRunExists is the dry-run line for an existing file (with overwrite info).
	InitDryRunExists = "  • %s already exists (%s)"

	// InitDryRunCreateScript is the dry-run line for creating a wrapper script.
	InitDryRunCreateScript = "  ✓ Create wrapper script %s"

	// InitDryRunExistsScript is the dry-run line for an existing wrapper script.
	InitDryRunExistsScript = "  • Wrapper script already exists %s (%s)"

	// InitDryRunBackup is the dry-run line for backup.
	InitDryRunBackup = "  📦 Old files will be backed up to backup/init-<timestamp>.zip"

	// InitOverwriteLabel is used in dry-run when --force is set.
	InitOverwriteLabel = "will be overwritten"

	// InitSkipLabel is used in dry-run when --force is not set.
	InitSkipLabel = "will be skipped"

	// InitDirExists is shown when a directory already exists.
	InitDirExists = "• Directory already exists %s"

	// InitDirCreated is shown when a directory is created.
	InitDirCreated = "✓ Created directory %s"

	// InitConfigCreated is shown when config file is created.
	InitConfigCreated = "✓ Created config file %s"

	// InitConfigExists is shown when config file already exists.
	InitConfigExists = "• Config file already exists %s"

	// InitConfigOverwritten is shown when config file is overwritten with --force.
	InitConfigOverwritten = "✓ Overwritten config file %s"

	// InitScriptCreated is shown when a wrapper script is created.
	InitScriptCreated = "✓ Created wrapper script %s"

	// InitScriptExists is shown when a wrapper script already exists.
	InitScriptExists = "• Wrapper script already exists %s"

	// InitScriptOverwritten is shown when a wrapper script is overwritten.
	InitScriptOverwritten = "✓ Overwritten wrapper script %s"

	// InitBackupCreated is shown when old files are backed up.
	InitBackupCreated = "📦 Old files backed up to %s"

	// InitIntegrationGuide is the integration guide shown after init.
	InitIntegrationGuide = TagCmdguard + ` Initialization complete.

To start using cmdguard, add one of the following to your shell config file
(~/.zshrc, ~/.bashrc, ~/.bash_profile, etc.):

Option 1 — Aliases (recommended for humans):
  alias rm='cmdguard rm'
  alias mv='cmdguard mv'
  alias chmod='cmdguard chmod'

Option 2 — PATH hijack (recommended for AI agents):
  export PATH="` + "`cmdguard env-path`" + `:$PATH"
  export CMDGUARD_NONINTERACTIVE=1   # skip the 5s/10s confirm wait

After editing, run 'source ~/.zshrc' (or restart your terminal) to apply changes.

Notes for AI agents:
  - CMDGUARD_NONINTERACTIVE=1 skips the interactive wait but does NOT
    grant permission. Protected paths still need --bypass=<id>.
  - --bypass identifier format: <host>/<platform>/<agent>/<task>
    Example: --bypass=mac-studio/qwenpaw/ai_research/cleanup-tmp-dirs

Test with: rm --check
`
)

// OverwriteLabel returns the label for dry-run based on force flag.
func OverwriteLabel(force bool) string {
	if force {
		return InitOverwriteLabel
	}
	return InitSkipLabel
}
