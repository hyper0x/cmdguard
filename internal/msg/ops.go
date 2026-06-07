package msg

// ─── Error message templates ────────────────────────────────────────

const (
	// ErrConfigParse is the error template when parsing config file fails.
	ErrConfigParse = "failed to parse config file %s: %w"

	// ErrConfigLoad is the error template when loading config fails.
	ErrConfigLoad = "failed to load config: %v"

	// ErrLogDir is the error template when creating log directory fails.
	ErrLogDir = "failed to create log directory: %w"

	// ErrLogLoad is the warning template when loading log fails.
	ErrLogLoad = TagCmdguard + " warning: failed to load log: %v"

	// ErrLogSerialize is the error template when serializing log entry fails.
	ErrLogSerialize = "failed to serialize log entry: %w"

	// ErrLogWrite is the error template when writing to log file fails.
	ErrLogWrite = "failed to write to log file: %w"

	// ErrLogLine is the error template when writing a log line fails.
	ErrLogLine = "failed to write log line: %w"

	// ErrCmdNotFound is the error template when a system command is not found.
	ErrCmdNotFound = "command %s not found"

	// ErrMkdir is the error template when creating a directory fails.
	ErrMkdir = "failed to create directory %s: %v"

	// ErrWriteFile is the error template when writing a file fails.
	ErrWriteFile = "failed to write file %s: %v"

	// ErrVaultNew is the error template when creating vault fails.
	ErrVaultNew = "failed to create vault: %v"

	// ErrVaultBackup is the warning template when backing up a file fails.
	ErrVaultBackup = "warning: backup of %s failed: %v"

	// ErrListExpired is the error template when listing expired backups fails.
	ErrListExpired = "failed to list expired backups: %v"

	// ErrPurgeExpired is the error template when purging expired backups fails.
	ErrPurgeExpired = "failed to purge expired backups: %v"
)

// ─── List command output ────────────────────────────────────────────

const (
	// ListNoResults is shown when no matching log entries are found.
	ListNoResults = TagCmdguard + " no matching records found"

	// ListTableHeader is the header for table output.
	ListTableHeader = "%-8s  %-19s  %-6s  %-8s  %s\n"

	// ListTableSeparator is the separator line for table output.
	ListTableSeparator = "--------  --------------------  ------  --------  --------------------------------"

	// ListBypassTag is appended to table rows that used --bypass.
	ListBypassTag = "  [bypass:%s]"

	// ListExpiredTag is appended to expired entries.
	ListExpiredTag = " [expired]"
)

// ─── Undo command output ────────────────────────────────────────────

const (
	// UndoNoRecords is shown when there are no recoverable operations.
	UndoNoRecords = TagCmdguard + " no recoverable operations"

	// UndoSelectPrompt is the interactive selection prompt.
	UndoSelectPrompt = TagCmdguard + " select an operation to restore:"

	// UndoSelectItem is the format for each selectable item.
	UndoSelectItem = "  %d. %s  %s  %s"

	// UndoSelectInput is the input prompt.
	UndoSelectInput = "Enter number (0 to cancel): "

	// UndoCancelled is shown when the user cancels.
	UndoCancelled = TagCmdguard + " cancelled"

	// UndoIDNotFound is shown when the given ID is not found.
	UndoIDNotFound = TagCmdguard + " no record found for ID '%s'"

	// UndoExpired is shown when the vault backup has expired.
	UndoExpired = TagCmdguard + " vault backup for this operation has expired, cannot restore"

	// UndoRejected is shown when trying to undo a rejected operation.
	UndoRejected = TagCmdguard + " this operation was rejected, nothing to restore"

	// UndoBackupNotFound is shown when the vault backup directory is not found.
	UndoBackupNotFound = TagCmdguard + " no vault backup found for ID '%s'"

	// UndoDryRunHeader is shown in dry-run mode.
	UndoDryRunHeader = TagCmdguard + " will restore the following files:"

	// UndoRestored is shown when files are restored.
	UndoRestored = TagCmdguard + " restored %d file(s)"

	// UndoRestoreFailed is the warning when a file fails to restore.
	UndoRestoreFailed = TagCmdguard + " warning: restore of %s failed: %v"

	// UndoNoFilesRestored is shown when no files were restored.
	UndoNoFilesRestored = TagCmdguard + " no files were restored"

	// UndoUsage is shown when no ID and no piped input.
	UndoUsage = TagCmdguard + " usage: cmdguard undo --id <ID>"
	UndoUsagePipe = "        cmdguard list | cmdguard undo"
	UndoUsageInteractive = "        cmdguard undo (interactive)"
)

// ─── Vault command output ───────────────────────────────────────────

const (
	// VaultNoExpired is shown when there are no expired backups.
	VaultNoExpired = TagCmdguard + " no expired vault backups"

	// VaultWillPurge is shown in dry-run mode.
	VaultWillPurge = TagCmdguard + " will delete %d expired backup(s):"

	// VaultPurged is shown after successful cleanup.
	VaultPurged = TagCmdguard + " cleaned %d expired backup(s)"
)

// ─── Config command output ──────────────────────────────────────────

const (
	// ConfigHeader is the header for config display.
	ConfigHeader = TagCmdguard + " current configuration:"

	// ConfigFile is the config file path line.
	ConfigFile = "  Config file: %s"

	// ConfigGlobalRules is the header for global rules.
	ConfigGlobalRules = "  Global protection rules:"

	// ConfigCommandRules is the header for command-level rules.
	ConfigCommandRules = "  Command-level rules:"

	// ConfigVaultSettings is the header for vault settings.
	ConfigVaultSettings = "  Vault settings:"

	// ConfigRetentionDays is the retention days line.
	ConfigRetentionDays = "    retention_days: %d"

	// ConfigAutoPurge is the auto-purge line.
	ConfigAutoPurge = "    auto_purge: %v"

	// ConfigGuardSettings is the header for [guard] settings.
	ConfigGuardSettings = "  Guard settings:"

	// ConfigConfirmTimeout is the confirm-timeout line.
	ConfigConfirmTimeout = "    confirm_timeout: %ds"

	// ConfigConfirmDoubleTimeout is the confirm_double-timeout line.
	ConfigConfirmDoubleTimeout = "    confirm_double_timeout: %ds"
)
