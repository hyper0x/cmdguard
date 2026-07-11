// Package config - environment variable contract.
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ ⚠️  All environment variables read by cmdguard MUST be declared    │
// │     here as constants and consumed via these constants.             │
// │                                                                     │
// │     Adding `os.Getenv("CMDGUARD_*")` elsewhere in the codebase is   │
// │     a code-review red flag - it bypasses this contract and makes    │
// │     it impossible to enumerate the program's runtime inputs.        │
// │                                                                     │
// │     Each constant below MUST have a doc comment explaining:         │
// │       - what the variable does                                       │
// │       - intended use case                                            │
// │       - example value                                                │
// │       - any safety implications                                      │
// └─────────────────────────────────────────────────────────────────────┘
package config

const (
	// EnvConfigDir overrides the default config directory (~/.cmdguard).
	// Useful for testing, sandboxes, or per-project configurations.
	//
	// Example:
	//   export CMDGUARD_CONFIG_DIR=/tmp/cmdguard-test
	EnvConfigDir = "CMDGUARD_CONFIG_DIR"
)
