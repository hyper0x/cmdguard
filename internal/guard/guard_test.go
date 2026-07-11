package guard

import (
	"testing"

	"github.com/hyper0x/cmdguard/internal/config"
)

func TestMatchPath_Exact(t *testing.T) {
	if !matchPath("/etc/passwd", "/etc/passwd") {
		t.Error("matchPath(/etc/passwd, /etc/passwd) should be true")
	}
}

func TestMatchPath_ExactNoMatch(t *testing.T) {
	if matchPath("/etc/passwd", "/etc/shadow") {
		t.Error("matchPath(/etc/passwd, /etc/shadow) should be false")
	}
}

func TestMatchPath_DoubleStarSuffix(t *testing.T) {
	if !matchPath("/etc/passwd", "/etc/**") {
		t.Error("matchPath(/etc/passwd, /etc/**) should be true")
	}
	if !matchPath("/etc/ssh/sshd_config", "/etc/**") {
		t.Error("matchPath(/etc/ssh/sshd_config, /etc/**) should be true")
	}
}

func TestMatchPath_DoubleStarPrefix(t *testing.T) {
	if !matchPath("myfile.key", "*.key") {
		t.Error("matchPath(myfile.key, *.key) should be true")
	}
	if !matchPath("path/to/secret.pem", "*.pem") {
		t.Error("matchPath(path/to/secret.pem, *.pem) should be true")
	}
}

func TestMatchPath_DoubleStarPrefixNoMatch(t *testing.T) {
	if matchPath("myfile.txt", "*.key") {
		t.Error("matchPath(myfile.txt, *.key) should be false")
	}
}

func TestMatchPath_DoubleStarOnly(t *testing.T) {
	if !matchPath("/anything/at/all", "**") {
		t.Error("matchPath(anything, **) should be true")
	}
}

func TestMatchPath_DoubleStarInfix(t *testing.T) {
	// After ExpandHome, ~/.ssh/** becomes /home/x/.ssh/** (or similar).
	// Use a neutral path here — never embed the developer's real $HOME
	// in tests, both for portability and to avoid leaking PII into the
	// public repo.
	if !matchPath("/home/x/.ssh/id_rsa", "/home/x/.ssh/**") {
		t.Error("matchPath should match .ssh/**")
	}
}

func TestMatchPath_GlobStar(t *testing.T) {
	if !matchPath("myfile.key", "*.key") {
		t.Error("matchPath(myfile.key, *.key) should be true")
	}
	if !matchPath("a.b.c.key", "*.key") {
		t.Error("matchPath(a.b.c.key, *.key) should be true")
	}
}

func TestMatchPath_GlobSingleChar(t *testing.T) {
	if !matchPath("file.txt", "file.???") {
		t.Error("matchPath(file.txt, file.???) should be true")
	}
	if matchPath("file.txt", "file.????") {
		t.Error("matchPath(file.txt, file.????) should be false")
	}
}

// TestMatchPath_DoubleStarPrefixSuffix covers the "**suffix" branch:
// pattern starts with `**` followed by a non-empty suffix.
// (Distinct from the bare "**" case above.)
//
// Important: `**suffix` matches only when the path ENDS WITH suffix.
// We deliberately do not match arbitrary substrings — `**.key` must
// not match `file.keystore` or `/a/.key/file`, otherwise protected
// rules would over-match.
func TestMatchPath_DoubleStarPrefixSuffix(t *testing.T) {
	// Match: path ends with the suffix.
	if !matchPath("/a/b/secret.key", "**.key") {
		t.Error("matchPath(/a/b/secret.key, **.key) should be true")
	}
	// No match: suffix appears in the middle, not at the end.
	if matchPath("/a/.key/file", "**.key") {
		t.Error("matchPath(/a/.key/file, **.key) should be false (mid-path)")
	}
	// No match: longer extension that merely contains the suffix.
	if matchPath("/a/b/file.keystore", "**.key") {
		t.Error("matchPath(/a/b/file.keystore, **.key) should be false")
	}
	// No match: completely unrelated.
	if matchPath("/a/b/file.txt", "**.key") {
		t.Error("matchPath(/a/b/file.txt, **.key) should be false")
	}
}

// TestMatchPath_DoubleStarSuffixBoundary guards against the regression
// where /etc/** matched /etcd/foo (bug fixed by requiring `/` boundary).
// Without this test, the prefix-only check would silently regress and
// rules would falsely fire on directories whose names happen to start
// with the same letters as a protected one.
func TestMatchPath_DoubleStarSuffixBoundary(t *testing.T) {
	pattern := "/etc/**"
	// Match: exact prefix path itself.
	if !matchPath("/etc", pattern) {
		t.Error("/etc should match /etc/**")
	}
	// Match: anything strictly inside /etc/.
	if !matchPath("/etc/passwd", pattern) {
		t.Error("/etc/passwd should match /etc/**")
	}
	if !matchPath("/etc/ssh/sshd_config", pattern) {
		t.Error("/etc/ssh/sshd_config should match /etc/**")
	}
	// No match: different directory whose name starts with "etc".
	if matchPath("/etcd", pattern) {
		t.Error("/etcd should NOT match /etc/**")
	}
	if matchPath("/etcd/foo", pattern) {
		t.Error("/etcd/foo should NOT match /etc/**")
	}
	// No match: different ssh-related dir.
	if matchPath("/Users/x/.ssh-backup/id_rsa", "/Users/x/.ssh/**") {
		t.Error("/.ssh-backup/* should NOT match /.ssh/**")
	}
}

// TestMatchPath_GlobFallback covers line 89: full-path glob fallback
// for patterns that contain "/" and wildcards but don't start/end with **.
func TestMatchPath_GlobFallback(t *testing.T) {
	// Single * in the middle of a path — falls through to matchGlob.
	if !matchPath("/tmp/file.log", "/tmp/file.*") {
		t.Error("matchPath(/tmp/file.log, /tmp/file.*) should be true")
	}
	if matchPath("/var/file.log", "/tmp/file.*") {
		t.Error("matchPath(/var/file.log, /tmp/file.*) should be false")
	}
}

// TestMatchPath_TildeNoExpand confirms matchPath itself does NOT expand ~.
// (Expansion happens in config.ExpandHome before this is called.)
// This is a defensive test: if someone removes ExpandHome upstream, callers
// still get a sane "no match" rather than a silent false positive.
func TestMatchPath_TildeNoExpand(t *testing.T) {
	// Literal tilde paths should match each other exactly.
	if !matchPath("~/.ssh/id_rsa", "~/.ssh/**") {
		t.Error("matchPath should match literal tilde paths against tilde patterns")
	}
}

// TestCheck_PrefixBoundaryRegression is an end-to-end regression test
// for the matchPath fix: a rule of /etc/** must NOT cause Check to
// reject paths under /etcd/. Without the `/` boundary requirement,
// users got false rejections on innocent paths like /etcd-data.
func TestCheck_PrefixBoundaryRegression(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject: []string{"/etc/**"},
		},
	}
	// Inside /etc/ — must be rejected.
	if r := Check(cfg, "rm", []string{"/etc/passwd"}); r.Action != "reject" {
		t.Errorf("/etc/passwd: action=%s, want reject", r.Action)
	}
	// /etc itself — must be rejected (the prefix path).
	if r := Check(cfg, "rm", []string{"/etc"}); r.Action != "reject" {
		t.Errorf("/etc: action=%s, want reject", r.Action)
	}
	// /etcd — different directory; must NOT be rejected.
	if r := Check(cfg, "rm", []string{"/etcd"}); r.Action == "reject" {
		t.Errorf("/etcd: should NOT be rejected by /etc/** rule")
	}
	// /etcd-data/snapshot — must NOT be rejected.
	if r := Check(cfg, "rm", []string{"/etcd-data/snapshot"}); r.Action == "reject" {
		t.Errorf("/etcd-data/snapshot: should NOT be rejected by /etc/** rule")
	}
}

func TestCheck_Reject(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject: []string{"/etc/**"},
		},
	}
	result := Check(cfg, "rm", []string{"/etc/passwd"})
	if result.Action != "reject" {
		t.Errorf("Check should return reject, got %s", result.Action)
	}
}

func TestCheck_Allow(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject: []string{"/etc/**"},
		},
	}
	result := Check(cfg, "rm", []string{"/tmp/testfile"})
	if result.Action != "allow" {
		t.Errorf("Check should return allow, got %s", result.Action)
	}
}

func TestCheck_Guarded(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Guarded: []string{"~/.config/**"},
		},
	}
	result := Check(cfg, "rm", []string{"~/.config/htop"})
	if result.Action != "guarded" {
		t.Errorf("Check should return guarded, got %s", result.Action)
	}
}

func TestCheck_Guarded_FromDeprecatedConfirm(t *testing.T) {
	// Old config files may still use 'confirm' - it should be treated
	// as guarded after Load() merges deprecated fields.
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Guarded: config.MergeGuarded(config.ProtectConfig{
				Confirm: []string{"~/Documents/**"},
			}),
		},
	}
	result := Check(cfg, "rm", []string{"~/Documents/archive"})
	if result.Action != "guarded" {
		t.Errorf("Check should return guarded (from deprecated confirm), got %s", result.Action)
	}
}

func TestCheck_CommandOverride(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject: []string{"/etc/**"},
			Command: map[string]config.ProtectConfig{
				"rm": {
					Reject: []string{"~/Documents/forbidden-to-delete"},
				},
			},
		},
	}
	// Global rule should still apply
	result := Check(cfg, "rm", []string{"/etc/passwd"})
	if result.Action != "reject" {
		t.Errorf("Global reject should still apply, got %s", result.Action)
	}
	// Command-specific rule should also apply
	result2 := Check(cfg, "rm", []string{"~/Documents/forbidden-to-delete"})
	if result2.Action != "reject" {
		t.Errorf("Command-specific reject should apply, got %s", result2.Action)
	}
	// Other command should not have the override
	result3 := Check(cfg, "mv", []string{"~/Documents/forbidden-to-delete"})
	if result3.Action == "reject" {
		t.Error("mv should not be affected by rm-specific rule")
	}
}

func TestCheck_FirstMatchWins(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject:  []string{"/etc/**"},
			Confirm: []string{"/etc/ssh/**"},
		},
	}
	// /etc/passwd matches reject first
	result := Check(cfg, "rm", []string{"/etc/passwd"})
	if result.Action != "reject" {
		t.Errorf("/etc/passwd should match reject first, got %s", result.Action)
	}
}

func TestCheck_MultipleTargets(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject: []string{"/etc/**"},
		},
	}
	// If any target matches, it should be rejected
	result := Check(cfg, "rm", []string{"/tmp/ok", "/etc/passwd"})
	if result.Action != "reject" {
		t.Errorf("Should reject when any target matches, got %s", result.Action)
	}
}

func TestCheck_ExpandHome(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject: []string{"~/.ssh/**"},
		},
	}
	// Check with expanded path
	result := Check(cfg, "rm", []string{"~/.ssh/id_rsa"})
	if result.Action != "reject" {
		t.Errorf("~/.ssh/id_rsa should be rejected, got %s", result.Action)
	}
}

func TestExtractTargets_Rm(t *testing.T) {
	targets := ExtractTargets("rm", []string{"-rf", "/tmp/test", "/tmp/test2"})
	if len(targets) != 2 {
		t.Fatalf("rm should extract 2 targets, got %d: %v", len(targets), targets)
	}
	if targets[0] != "/tmp/test" {
		t.Errorf("targets[0] = %q, want /tmp/test", targets[0])
	}
	if targets[1] != "/tmp/test2" {
		t.Errorf("targets[1] = %q, want /tmp/test2", targets[1])
	}
}

func TestExtractTargets_Mv(t *testing.T) {
	targets := ExtractTargets("mv", []string{"/tmp/src", "/tmp/dst"})
	if len(targets) != 1 {
		t.Fatalf("mv should extract 1 target (destination), got %d: %v", len(targets), targets)
	}
	if targets[0] != "/tmp/dst" {
		t.Errorf("mv target = %q, want /tmp/dst", targets[0])
	}
}

func TestExtractTargets_Chmod(t *testing.T) {
	targets := ExtractTargets("chmod", []string{"755", "/tmp/script.sh"})
	if len(targets) != 1 {
		t.Fatalf("chmod should extract 1 target, got %d: %v", len(targets), targets)
	}
	if targets[0] != "/tmp/script.sh" {
		t.Errorf("chmod target = %q, want /tmp/script.sh", targets[0])
	}
}

func TestExtractTargets_ChmodMultiple(t *testing.T) {
	targets := ExtractTargets("chmod", []string{"755", "/tmp/a.sh", "/tmp/b.sh"})
	if len(targets) != 2 {
		t.Fatalf("chmod should extract 2 targets, got %d: %v", len(targets), targets)
	}
}

func TestExtractTargets_FlagsSkipped(t *testing.T) {
	targets := ExtractTargets("rm", []string{"-rf", "--verbose", "/tmp/test"})
	if len(targets) != 1 {
		t.Fatalf("Should skip flags, got %d: %v", len(targets), targets)
	}
}

func TestExtractAllTargets(t *testing.T) {
	targets := ExtractAllTargets([]string{"-rf", "/tmp/src", "/tmp/dst"})
	if len(targets) != 2 {
		t.Fatalf("ExtractAllTargets should return 2, got %d: %v", len(targets), targets)
	}
	if targets[0] != "/tmp/src" {
		t.Errorf("targets[0] = %q, want /tmp/src", targets[0])
	}
}
