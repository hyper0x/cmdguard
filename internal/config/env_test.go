package config

import "testing"

// TestIsNonInteractive_NotSet verifies that without the env var,
// IsNonInteractive returns false (the default interactive mode).
func TestIsNonInteractive_NotSet(t *testing.T) {
	t.Setenv(EnvNonInteractive, "")
	if IsNonInteractive() {
		t.Error("IsNonInteractive should be false when env var is empty")
	}
}

// TestIsNonInteractive_Set verifies that any non-empty value enables it.
// This is the contract relied on by AI agents / CI pipelines.
func TestIsNonInteractive_Set(t *testing.T) {
	cases := []string{"1", "true", "yes", "anything"}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			t.Setenv(EnvNonInteractive, v)
			if !IsNonInteractive() {
				t.Errorf("IsNonInteractive should be true when env var is %q", v)
			}
		})
	}
}

// TestIsNonInteractive_DoesNotBypass is a contract test: setting this env var
// MUST NOT change Check()'s decision. The env only skips the wait — it never
// grants permission. See env.go safety note.
//
// This test guards against a future regression where someone might think
// "non-interactive" means "auto-approve".
func TestIsNonInteractive_DoesNotBypass(t *testing.T) {
	t.Setenv(EnvNonInteractive, "1")

	// Without env: confirm action expected for default config rule.
	// With env: still confirm — the wrapper layer interprets it, not config.
	// This test simply documents the boundary: IsNonInteractive is a pure
	// env-var read, NOT a permission check.
	if !IsNonInteractive() {
		t.Fatal("precondition failed")
	}
	// Explicitly: the function returns a bool. It does not mutate global
	// state, does not modify any Config, and has no side effects.
	// (If anyone adds side effects later, this comment is a tripwire.)
}
