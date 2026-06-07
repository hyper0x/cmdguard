// Package config — environment variable contract.
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ ⚠️  All environment variables read by cmdguard MUST be declared    │
// │     here as constants and consumed via these constants.             │
// │                                                                     │
// │     Adding `os.Getenv("CMDGUARD_*")` elsewhere in the codebase is   │
// │     a code-review red flag — it bypasses this contract and makes    │
// │     it impossible to enumerate the program's runtime inputs.        │
// │                                                                     │
// │     Each constant below MUST have a doc comment explaining:         │
// │       - what the variable does                                       │
// │       - intended use case                                            │
// │       - example value                                                │
// │       - any safety implications                                      │
// └─────────────────────────────────────────────────────────────────────┘
package config

import "os"

const (
	// EnvConfigDir overrides the default config directory (~/.cmdguard).
	// Useful for testing, sandboxes, or per-project configurations.
	//
	// Example:
	//   export CMDGUARD_CONFIG_DIR=/tmp/cmdguard-test
	EnvConfigDir = "CMDGUARD_CONFIG_DIR"

	// EnvNonInteractive, when set to a non-empty value, tells cmdguard
	// to skip the interactive confirmation wait entirely and go straight
	// to the non-TTY rejection path (with --bypass guidance).
	//
	// Intended for AI agents, CI pipelines, and any automation that
	// cannot respond to interactive prompts. Setting this avoids the
	// 5s/10s default wait at the confirm/confirm_double prompts.
	//
	// ⚠️  Safety: this env var does NOT grant permission. The operation
	//     is still REJECTED unless a valid --bypass=<id> is also given.
	//     It only saves the wait time — it never opens a door.
	//
	// Example:
	//   export CMDGUARD_NONINTERACTIVE=1
	EnvNonInteractive = "CMDGUARD_NONINTERACTIVE"
)

// IsNonInteractive reports whether the EnvNonInteractive env var is set
// to any non-empty value. The exact value is not interpreted — presence
// alone is the signal.
func IsNonInteractive() bool {
	return os.Getenv(EnvNonInteractive) != ""
}
