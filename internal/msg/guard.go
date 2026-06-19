package msg

import (
	"fmt"
	"strings"
)

// ─── Guard entry point ──────────────────────────────────────────────

const (
	// GuardCheckOK is shown when --check confirms the guard is active.
	GuardCheckOK = TagCmdguard + " guard active — %s is running through cmdguard"

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

	// GuardDryRunBackup is shown in dry-run mode for confirm_double/confirm/warn.
	GuardDryRunBackup = TagCmdguard + " will backup and execute %s"

	// GuardExecuting is shown in verbose mode before executing the real command.
	GuardExecuting = TagCmdguard + " executing: %s %s"
)

// ─── Confirm / Confirm Double prompts ────────────────────────────────

const (
	// ConfirmTimeoutMsg is shown when the read timeout elapses.
	// Substituted with the actual configured timeout in seconds.
	ConfirmTimeoutMsg = TagCmdguard + " no input within %ds, treating as non-interactive"

	// ConfirmTimeoutLogMsg is the log message for timeout rejection.
	ConfirmTimeoutLogMsg = "timeout waiting for confirmation (%ds)"

	// ConfirmHint is a one-line tip shown before the single confirmation prompt.
	ConfirmHint = TagCmdguard + " press y to proceed, N or Enter to cancel, Ctrl+C to abort (timeout %ds)"

	// ConfirmDoubleHint is a one-line tip shown for double confirmation.
	ConfirmDoubleHint = TagCmdguard + " DOUBLE confirmation required: first press y, then type 'yes' and Enter (timeout %ds per step)"

	// ConfirmTimeoutDisabled is shown when timeout is configured as 0
	// (wait forever). Lets the user know they may need Ctrl+C.
	ConfirmTimeoutDisabled = TagCmdguard + " (timeout disabled — waiting indefinitely, use Ctrl+C to abort)"

	// ConfirmPrompt is the single-confirmation prompt.
	ConfirmPrompt = "Proceed? (y/N): "

	// ConfirmDoublePrompt1 is the first prompt for double confirmation.
	ConfirmDoublePrompt1 = "Proceed? (y/N): "

	// ConfirmDoublePrompt2 is the second prompt for double confirmation.
	ConfirmDoublePrompt2 = "Confirm again — type 'yes' to proceed: "

	// ConfirmCancelled is shown when the user cancels.
	ConfirmCancelled = TagCmdguard + " cancelled"

	// ConfirmCancelledMsg is the log message when the user cancels.
	ConfirmCancelledMsg = "cancelled by user"

	// ConfirmDoubleCancelledMsg is the log message when double confirmation fails.
	ConfirmDoubleCancelledMsg = "cancelled by user (double confirmation failed)"
)

// ─── Non-TTY / Bypass ───────────────────────────────────────────────

const (
	// GuardNonTTYRejected is shown when running non-interactively and a confirm-level path is hit.
	GuardNonTTYRejected = TagCmdguard + " non-interactive mode, execution rejected"

	// GuardNonTTYMsg is the log message for non-TTY rejection.
	GuardNonTTYMsg = "non-interactive mode, automatically rejected"

	// GuardEnvNonInteractiveMsg is the log message when the
	// CMDGUARD_NONINTERACTIVE env var triggered the rejection.
	GuardEnvNonInteractiveMsg = "rejected: CMDGUARD_NONINTERACTIVE is set"

	// GuardBypassHelp is the guidance shown when an agent must use --bypass.
	GuardBypassHelp = TagCmdguard + ` This is a protected path (level: %s).

 If you are an AI agent or automation:

 Step 1 — Declare yourself non-interactive (skip the 5s/10s wait):
   export CMDGUARD_NONINTERACTIVE=1

 Step 2 — If you have reviewed this operation and confirmed it is safe,
          retry with a --bypass=<identifier>:

   %s --bypass=<host>/<platform>/<agent>/<task>

 ⚠️  CMDGUARD_NONINTERACTIVE alone does NOT grant permission. The
     operation is still REJECTED unless --bypass=<id> is also given.
     The env var only saves the interactive wait — it never opens a door.

 ⚠️  DO NOT copy the template "<host>/<platform>/<agent>/<task>"
     verbatim. Replace each placeholder with a real value. Identifiers
     that literally contain "host", "platform", "agent", "task" or
     angle brackets will be REJECTED.

 The <identifier> MUST be a 4-segment path:
   host      hostname or machine alias  (e.g. mac-studio)
   platform  agent platform             (e.g. qwenpaw, claude-code, cursor)
   agent     unique identifier         (e.g. ai_research, coding)
   task      brief task slug            (e.g. cleanup-tmp-dirs)

 Allowed characters per segment: [a-zA-Z0-9._-], no empty segments.

 Examples (do NOT copy verbatim — use your own values):
   --bypass=mac-studio/qwenpaw/ai_research/cleanup-tmp-dirs
   --bypass=laptop-air/claude-code/default/refactor-tests
`

	// GuardBypassFlag is the flag name.
	GuardBypassFlag = "--bypass"

	// GuardBypassLogMsg is the log message template for bypass operations.
	//
	// #nosec G101 -- gosec flags "bypass" as a credential-looking
	// keyword. This is a user-facing message template, not a secret;
	// the "%s" holds an agent identifier (host/platform/agent/task).
	GuardBypassLogMsg = "bypassed by %s via --bypass"

	// GuardBypassInvalid is shown when --bypass identifier does not match the required format.
	GuardBypassInvalid = TagCmdguard + ` invalid --bypass identifier: %q

 The <identifier> MUST be a 4-segment path:
   host      hostname or machine alias  (e.g. mac-studio)
   platform  agent platform             (e.g. qwenpaw, claude-code, cursor)
   agent     unique identifier         (e.g. ai_research, coding)
   task      brief task slug            (e.g. cleanup-tmp-dirs)

 Format rules:
   - exactly 4 segments separated by '/'
   - each segment matches [a-zA-Z0-9._-]+ (no empty segments)
   - total length >= 12 characters
   - no template placeholders allowed (host, platform, agent, task,
     angle brackets <>, foo, xxx, todo, ...)

 ⚠️  Do NOT copy the template "<host>/<platform>/<agent>/<task>"
     verbatim. Replace each placeholder with a real value.

 Retry with a properly formed identifier:

   %s --bypass=<host>/<platform>/<agent>/<task>

 Examples (do NOT copy verbatim — use your own values):
   --bypass=mac-studio/qwenpaw/ai_research/cleanup-tmp-dirs
   --bypass=laptop-air/claude-code/default/refactor-tests
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
