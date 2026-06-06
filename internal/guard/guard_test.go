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
	// After ExpandHome, ~/.ssh/** becomes /Users/xxx/.ssh/**
	// This tests the pattern matching after expansion
	if !matchPath("/Users/haolin/.ssh/id_rsa", "/Users/haolin/.ssh/**") {
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

func TestCheck_ConfirmDouble(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			ConfirmDouble: []string{"~/.config/**"},
		},
	}
	result := Check(cfg, "rm", []string{"~/.config/htop"})
	if result.Action != "confirm_double" {
		t.Errorf("Check should return confirm_double, got %s", result.Action)
	}
}

func TestCheck_Confirm(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Confirm: []string{"~/Documents/**"},
		},
	}
	result := Check(cfg, "rm", []string{"~/Documents/archive"})
	if result.Action != "confirm" {
		t.Errorf("Check should return confirm, got %s", result.Action)
	}
}

func TestCheck_Warn(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Warn: []string{"~/Downloads/**"},
		},
	}
	result := Check(cfg, "rm", []string{"~/Downloads/file.zip"})
	if result.Action != "warn" {
		t.Errorf("Check should return warn, got %s", result.Action)
	}
}

func TestCheck_CommandOverride(t *testing.T) {
	cfg := &config.Config{
		Protect: config.ProtectConfig{
			Reject: []string{"/etc/**"},
			Command: map[string]config.ProtectConfig{
				"rm": {
					Reject: []string{"~/Documents/不许删"},
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
	result2 := Check(cfg, "rm", []string{"~/Documents/不许删"})
	if result2.Action != "reject" {
		t.Errorf("Command-specific reject should apply, got %s", result2.Action)
	}
	// Other command should not have the override
	result3 := Check(cfg, "mv", []string{"~/Documents/不许删"})
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
