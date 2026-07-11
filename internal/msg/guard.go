package msg

import (
	"fmt"
	"strings"
)

// ─── Guard entry point ──────────────────────────────────────────────

const (
	// GuardCheckOK is shown when --check confirms the guard is active.
	GuardCheckOK = TagCmdguard + " guard active - %s is running through cmdguard"

	// GuardNoTargets is shown when no file paths are found in arguments.
	GuardNoTargets = TagCmdguard + " no file paths detected, executing directly"

	// GuardTargetMissing is shown when a target path does not exist.
	// cmdguard exits early without invoking the underlying command.
	GuardTargetMissing = TagCmdguard + " %s: '%s': no such file or directory"

	// GuardMvDestNew is shown in verbose mode when mv's destination
	// does not exist (i.e. mv is creating a new file/path). Protection
	// rules are skipped and the command is executed directly.
	GuardMvDestNew = TagCmdguard + " mv destination does not exist, no protection needed, executing directly"

	// GuardRuleMatched is shown in verbose mode when a rule matches.
	GuardRuleMatched = TagCmdguard + " matched rule: %s (level: %s)"

	// GuardNoRule is shown in verbose mode when no rule matches.
	GuardNoRule = TagCmdguard + " no matching rule"

	// GuardRejected is shown after a reject action.
	GuardRejected = TagCmdguard + " execution rejected"

	// GuardDryRunBackup is shown in dry-run mode for guarded paths.
	GuardDryRunBackup = TagCmdguard + " will backup and execute %s"

	// GuardExecuting is shown in verbose mode before executing the real command.
	GuardExecuting = TagCmdguard + " executing: %s %s"
)

// ─── Guarded-path rejection (no --bypass) ───────────────────────────

const (
	// GuardNoBypassRejected is shown when a guarded path is hit
	// without --bypass. In non-interactive mode this is the only
	// rejection path: there is no interactive confirmation fallback.
	GuardNoBypassRejected = TagCmdguard + " guarded path: execution rejected (no --bypass provided)"

	// GuardNoBypassMsg is the log message for guarded-path rejection.
	GuardNoBypassMsg = "rejected: guarded path, no --bypass provided"

	// GuardBackupFailed is shown when a vault backup fails. Backup
	// failure is always fatal (exit 1): the undo safety net is broken,
	// so proceeding would leave the user without a rollback path.
	//
	// The %s placeholder receives the vault directory path so the
	// caller (human or agent) can immediately check permissions /
	// disk space / sandbox constraints without having to run
	// `cmdguard path` separately.
	GuardBackupFailed = TagCmdguard + ` backup failed, aborting (undo safety net broken)

  Vault dir: %s
  Common causes:
    - vault directory not writable (check permissions)
    - disk full
    - sandbox restriction (e.g. macOS Seatbelt blocking writes outside allowed paths)
  Run 'cmdguard path' to inspect directory layout and sizes.`
)

// ─── Bypass ─────────────────────────────────────────────────────────

const (
	// GuardBypassHelp is the guidance shown when a guarded path is
	// hit without --bypass.
	GuardBypassHelp = TagCmdguard + ` This is a protected path (level: %s).

 If you have reviewed this operation and confirmed it is safe,
 retry with a --bypass=<identifier>:

   %s --bypass=<platform>/<agent>/<task>

 ⚠️  --bypass does NOT bypass audit logging. The operation is
     still recorded with the bypass identifier for traceability.

 ⚠️  DO NOT copy the template "<platform>/<agent>/<task>"
     verbatim. Replace each placeholder with a real value. Identifiers
     that literally contain "platform", "agent", "task" or angle
     brackets will be REJECTED.

 The <identifier> MUST be a 3-segment path:
   platform  caller platform            (e.g. qwenpaw, claude-code, manual)
   agent     unique caller identifier   (e.g. ai_research, haolin)
   task      brief task slug            (e.g. cleanup-tmp-dirs)

 Allowed characters per segment: [a-zA-Z0-9._-], no empty segments.

 Examples (do NOT copy verbatim - use your own values):
   --bypass=qwenpaw/ai_research/cleanup-tmp-dirs
   --bypass=manual/haolin/remove-old-logs
`

	// GuardBypassFlag is the flag name.
	GuardBypassFlag = "--bypass"

	// GuardBypassLogMsg is the log message template for bypass operations.
	//
	// #nosec G101 -- gosec flags "bypass" as a credential-looking
	// keyword. This is a user-facing message template, not a secret;
	// the "%s" holds an agent identifier (platform/agent/task).
	GuardBypassLogMsg = "bypassed by %s via --bypass"

	// GuardBypassInvalid is shown when --bypass identifier does not match the required format.
	GuardBypassInvalid = TagCmdguard + ` invalid --bypass identifier: %q

 The <identifier> MUST be a 3-segment path:
   platform  caller platform            (e.g. qwenpaw, claude-code, manual)
   agent     unique caller identifier   (e.g. ai_research, haolin)
   task      brief task slug            (e.g. cleanup-tmp-dirs)

 Format rules:
   - exactly 3 segments separated by '/'
   - each segment matches [a-zA-Z0-9._-]+ (no empty segments)
   - total length >= 10 characters
   - no template placeholders allowed (platform, agent, task,
     angle brackets <>, foo, xxx, todo, ...)

 ⚠️  Do NOT copy the template "<platform>/<agent>/<task>"
     verbatim. Replace each placeholder with a real value.

 Retry with a properly formed identifier:

   %s --bypass=<platform>/<agent>/<task>

 Examples (do NOT copy verbatim - use your own values):
   --bypass=qwenpaw/ai_research/cleanup-tmp-dirs
   --bypass=manual/haolin/remove-old-logs
`

	// GuardBypassInvalidMsg is the log message for invalid bypass.
	GuardBypassInvalidMsg = "rejected: invalid --bypass identifier %q"
)

// ─── Warning output ─────────────────────────────────────────────────

// GuardWarningFmt returns the formatted warning block.
func GuardWarningFmt(cmd string, icon, level, rule, msg string, targets []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s [cmdguard] %s: %s\n", icon, level, msg)
	fmt.Fprintf(&b, "  Command: %s\n", cmd)
	for _, t := range targets {
		fmt.Fprintf(&b, "  Path: %s\n", t)
	}
	fmt.Fprintf(&b, "  Rule: %s\n", rule)
	return b.String()
}

// ─── Verbose mode ───────────────────────────────────────────────────

const (
	// VerboseBackupDir is shown in verbose mode indicating the backup directory.
	VerboseBackupDir = TagCmdguard + " backup directory: %s"

	// VerboseBackupFile is shown in verbose mode when backing up a file.
	VerboseBackupFile = TagCmdguard + " backing up: %s"

	// VerboseBackupFailed is shown when a backup fails (warning, not error).
	VerboseBackupFailed = TagCmdguard + " warning: backup of %s failed: %v"
)

// ─── Dry-run output ─────────────────────────────────────────────────

const (
	// DryRunWillBackup is shown in dry-run mode listing files to backup.
	DryRunWillBackup = TagCmdguard + " will backup the following files before executing %s:"
)
