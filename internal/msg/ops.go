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
	// Pass through msg.FmtWarn / subcmd.warn — the [cmdguard] tag and
	// "warning:" prefix are added at the call site, NOT embedded here.
	// Embedding the prefix in the template caused double-tagging once
	// callers were standardised on the FmtWarn helper.
	ErrLogLoad = "failed to load log: %v"

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
	// Same convention as ErrLogLoad: no embedded "warning:" prefix.
	ErrVaultBackup = "backup of %s failed: %v"

	// ErrListExpired is the error template when listing expired backups fails.
	ErrListExpired = "failed to list expired backups: %v"

	// ErrPurgeExpired is the error template when purging expired backups fails.
	ErrPurgeExpired = "failed to purge expired backups: %v"

	// ErrListSinceInvalid is shown when --since receives an unparseable value.
	// Examples of malformed input: "7days", "tomorrow", "" (empty).
	// Examples of accepted input: "30m", "2h", "7d".
	ErrListSinceInvalid = "invalid --since value %q (use formats like 30m, 2h, 7d)"
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

	// VaultListNoBackups is shown when the vault is empty.
	VaultListNoBackups = TagCmdguard + " no vault backups"

	// VaultListTableHeader is the header for vault list table output.
	VaultListTableHeader = "%-12s  %-19s  %-20s  %s\n"

	// VaultListTableSeparator is the separator line for vault list table output.
	VaultListTableSeparator = "------------  -------------------  --------------------  --------"

	// VaultListTableRow is the format for each vault list row.
	VaultListTableRow = "%-12s  %-19s  %-20s  %s\n"

	// VaultListSummary is the summary line for vault list.
	VaultListSummary = TagCmdguard + " total: %d backup(s)"

	// VaultListStatusExpired is the status tag for expired backups.
	VaultListStatusExpired = "expired"
)

// ─── Path command output ────────────────────────────────────────────

const (
	// PathHeader is the header for path display.
	PathHeader = TagCmdguard + " cmdguard directory structure"

	// PathConfigDir is the config directory line.
	PathConfigDir = "  Config directory: %s"

	// PathConfigFile is the config file line.
	PathConfigFile = "  Config file:      %s  (%s)"

	// PathLogDir is the log directory line.
	PathLogDir = "  Log directory: %s"

	// PathVaultDir is the vault directory line.
	PathVaultDir = "  Vault directory: %s"

	// PathBinDir is the bin directory line.
	PathBinDir = "  Bin directory: %s"

	// PathFileNotExist is shown when a file doesn't exist.
	PathFileNotExist = "not exist"

	// PathDirNotExist is shown when a directory doesn't exist.
	PathDirNotExist = "not exist"

	// PathDirEmpty is shown when a directory is empty.
	PathDirEmpty = "empty"

	// PathDirError is shown when reading a directory fails.
	PathDirError = "error"

	// PathFileCount is the file count summary.
	PathFileCount = "%d file(s)"

	// PathVaultSummary is the vault summary line.
	PathVaultSummary = "%d backup(s), %s"
)

const (
	// ConfigEffectiveHeader is the header for effective (merged) config display.
	ConfigEffectiveHeader = TagCmdguard + " effective configuration:"

	// ConfigDefaultHeader is the header for default config display.
	ConfigDefaultHeader = TagCmdguard + " built-in default configuration:"

	// ConfigRawHeader is the header for raw config file display.
	ConfigRawHeader = TagCmdguard + " raw config file: %s"

	// ConfigRawNotExist is shown when the config file doesn't exist.
	ConfigRawNotExist = TagCmdguard + " config file %s does not exist"

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
