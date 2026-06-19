// Package msg centralizes all user-facing messages, prompts, and templates.
// Messages are organized by category into separate files within this package.
package msg

import "fmt"

// Tags used in log entries and output.
const (
	TagCmdguard = "[cmdguard]"
)

// Level labels used in output.
const (
	LevelReject       = "reject"
	LevelConfirm      = "confirm"
	LevelConfirmDbl   = "confirm_double"
	LevelWarn         = "warn"
	LevelAllow        = "allow"
	LevelUndo         = "undo"
	LevelVaultClean   = "vault-clean"
)

// LevelIcons maps protection levels to their display icons.
var LevelIcons = map[string]string{
	LevelReject:     "🚫",
	LevelConfirmDbl: "🔒",
	LevelConfirm:    "❓",
	LevelWarn:       "⚠️",
}

// LevelLabels maps protection levels to their Chinese labels.
var LevelLabels = map[string]string{
	LevelReject:     "Reject",
	LevelConfirmDbl: "Double Confirm",
	LevelConfirm:    "Confirm",
	LevelWarn:       "Warning",
}

// LevelActions maps protection levels to action descriptions.
var LevelActions = map[string]string{
	LevelReject:     "directly rejected, not executed",
	LevelConfirmDbl: "double confirmation required (type 'yes') → backup → execute",
	LevelConfirm:    "single confirmation required (press 'y') → backup → execute",
	LevelWarn:       "warning shown → backup → execute",
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
