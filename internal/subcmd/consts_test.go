package subcmd

import "testing"

// TestIsGuarded_KnownCommands verifies the three guarded commands are
// recognized. This is a contract test: if someone changes the set of
// guarded commands without updating the docs/wrappers, this fails.
func TestIsGuarded_KnownCommands(t *testing.T) {
	for _, cmd := range []string{"rm", "mv", "chmod"} {
		if !IsGuarded(cmd) {
			t.Errorf("IsGuarded(%q) = false, want true", cmd)
		}
	}
}

// TestIsGuarded_UnknownCommands verifies that arbitrary commands and
// near-misses do NOT pass through as guarded. This is what stops
// `cmdguard banana` from being routed to RunGuard.
func TestIsGuarded_UnknownCommands(t *testing.T) {
	for _, cmd := range []string{
		"ls", "cat", "rmdir", "RM", "rm ", " rm", "", "init", "vault",
	} {
		if IsGuarded(cmd) {
			t.Errorf("IsGuarded(%q) = true, want false", cmd)
		}
	}
}

// TestGuardedCommands_StableOrder verifies GuardedCommands returns a
// non-empty list with all expected entries. The order matters because
// init dry-run prints them in this order in user-visible output.
func TestGuardedCommands_StableOrder(t *testing.T) {
	got := GuardedCommands()
	want := []string{"rm", "mv", "chmod"}
	if len(got) != len(want) {
		t.Fatalf("GuardedCommands() len = %d, want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("GuardedCommands()[%d] = %q, want %q", i, got[i], v)
		}
	}
}

// TestGuardedCommands_AlignsWithIsGuarded ensures the slice and the
// membership map stay in sync. If a future contributor adds a command
// to one and forgets the other, this test fires.
func TestGuardedCommands_AlignsWithIsGuarded(t *testing.T) {
	for _, cmd := range GuardedCommands() {
		if !IsGuarded(cmd) {
			t.Errorf("GuardedCommands listed %q but IsGuarded says false", cmd)
		}
	}
}
