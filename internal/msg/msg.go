// Package msg centralizes all user-facing messages, prompts, and templates.
// Messages are organized by category into separate files within this package.
package msg

import "fmt"

// Tags used in log entries and output.
const (
	TagCmdguard = "[cmdguard]"
)

// Level labels used in output and log entries.
//
// The protection model has exactly three outcomes:
//
//   - reject:  the path is permanently off-limits. No --bypass can
//     override it. The operation is refused and logged.
//   - guarded: the path requires an explicit --bypass to proceed.
//     Without --bypass the operation is refused and logged.
//     With a valid --bypass the file is backed up to the
//     vault, the operation is logged, then executed.
//   - allow:   no rule matched. The operation executes directly and
//     is logged.
const (
	LevelReject  = "reject"
	LevelGuarded = "guarded"
	LevelAllow   = "allow"

	// Legacy level strings kept for log-reading compatibility
	// (undo, list). They are NOT produced by the current guard flow
	// but may appear in historical log entries.
	LevelConfirm    = "confirm"
	LevelConfirmDbl = "confirm_double"
	LevelWarn       = "warn"
	LevelUndo       = "undo"
	LevelVaultClean = "vault-clean"
)

// LevelIcons maps protection levels to their display icons.
var LevelIcons = map[string]string{
	LevelReject:  "🚫",
	LevelGuarded: "🔒",
	LevelAllow:   "✅",
	// Legacy entries for reading old logs.
	LevelConfirmDbl: "🔒",
	LevelConfirm:    "❓",
	LevelWarn:       "⚠️",
}

// LevelLabels maps protection levels to their display labels.
var LevelLabels = map[string]string{
	LevelReject:  "Reject",
	LevelGuarded: "Guarded",
	LevelAllow:   "Allow",
	// Legacy entries for reading old logs.
	LevelConfirmDbl: "Double Confirm",
	LevelConfirm:    "Confirm",
	LevelWarn:       "Warning",
}

// Fmt returns a formatted [cmdguard] message.
func Fmt(format string, args ...any) string {
	return fmt.Sprintf(TagCmdguard+" "+format, args...)
}

// FmtErr returns a formatted [cmdguard] error message.
func FmtErr(format string, args ...any) string {
	return fmt.Sprintf(TagCmdguard+" error: "+format, args...)
}

// FmtWarn returns a formatted [cmdguard] warning message.
func FmtWarn(format string, args ...any) string {
	return fmt.Sprintf(TagCmdguard+" warning: "+format, args...)
}
