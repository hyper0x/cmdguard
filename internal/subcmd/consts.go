// Package subcmd — constants and package-level variables.
package subcmd

// guardedCommands is the single source of truth for which commands
// cmdguard intercepts. Order matters: it determines the iteration
// order in init dry-run output and `cmdguard path` listings.
//
// Kept private so callers go through IsGuarded / GuardedCommands.
var guardedCommands = []string{"rm", "mv", "chmod"}

// guardedSet is derived from guardedCommands at package init time
// to give IsGuarded an O(1) membership lookup. Since guardedCommands
// is small (3 entries), the linear scan would also be fine — the
// map exists mainly to make the boundary between "list" and
// "membership test" explicit at the API level.
var guardedSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(guardedCommands))
	for _, c := range guardedCommands {
		m[c] = struct{}{}
	}
	return m
}()

// IsGuarded reports whether the given command is intercepted by cmdguard.
// O(1) lookup; preferred over GuardedCommands() for membership tests.
func IsGuarded(cmd string) bool {
	_, ok := guardedSet[cmd]
	return ok
}

// GuardedCommands returns the list of commands protected by cmdguard,
// in stable iteration order. Use this when you need to enumerate all
// guarded commands (e.g. init dry-run, path command). For membership
// tests, prefer IsGuarded.
//
// Returns a copy so callers cannot mutate the underlying slice.
func GuardedCommands() []string {
	out := make([]string, len(guardedCommands))
	copy(out, guardedCommands)
	return out
}

// Version and Commit are set via -ldflags at build time.
var Version = "dev"
var Commit = "none"
