package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome_Tilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := ExpandHome("~")
	if got != home {
		t.Errorf("ExpandHome(~) = %q, want %q", got, home)
	}
}

func TestExpandHome_TildePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "Downloads")
	got := ExpandHome("~/Downloads")
	if got != want {
		t.Errorf("ExpandHome(~/Downloads) = %q, want %q", got, want)
	}
}

func TestExpandHome_NormalPath(t *testing.T) {
	got := ExpandHome("/etc/passwd")
	if got != "/etc/passwd" {
		t.Errorf("ExpandHome(/etc/passwd) = %q, want /etc/passwd", got)
	}
}

func TestExpandHome_Empty(t *testing.T) {
	got := ExpandHome("")
	if got != "" {
		t.Errorf("ExpandHome() = %q, want empty", got)
	}
}

func TestDefaultConfig_HasRejectRules(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.Protect.Reject) == 0 {
		t.Fatal("DefaultConfig should have reject rules")
	}
}

func TestDefaultConfig_HasVaultDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Vault.RetentionDays != 7 {
		t.Errorf("Default retention_days = %d, want 7", cfg.Vault.RetentionDays)
	}
	if !cfg.Vault.AutoPurge {
		t.Error("Default auto_purge should be true")
	}
}

func TestDefaultConfig_CommandMapEmpty(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Protect.Command == nil {
		t.Fatal("Command map should be initialized")
	}
	if len(cfg.Protect.Command) != 0 {
		t.Errorf("Command map should be empty, got %d entries", len(cfg.Protect.Command))
	}
}

func TestConfigDir_Default(t *testing.T) {
	// Unset env var
	os.Unsetenv("CMDGUARD_CONFIG_DIR")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".cmdguard")
	got := ConfigDir()
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_EnvOverride(t *testing.T) {
	os.Setenv("CMDGUARD_CONFIG_DIR", "/tmp/test-cmdguard")
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")
	got := ConfigDir()
	if got != "/tmp/test-cmdguard" {
		t.Errorf("ConfigDir() = %q, want /tmp/test-cmdguard", got)
	}
}

func TestConfigPath(t *testing.T) {
	got := ConfigPath()
	want := filepath.Join(ConfigDir(), "config.toml")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestFlattenProtect_Empty(t *testing.T) {
	p := &ProtectConfig{}
	rules := flattenProtect(p)
	if len(rules) != 0 {
		t.Errorf("flattenProtect(empty) = %d rules, want 0", len(rules))
	}
}

func TestFlattenProtect_AllLevels(t *testing.T) {
	p := &ProtectConfig{
		Reject:        []string{"/etc/**"},
		ConfirmDouble: []string{"~/.config/**"},
		Confirm:       []string{"~/Documents/**"},
		Warn:          []string{"~/Downloads/**"},
	}
	rules := flattenProtect(p)
	if len(rules) != 4 {
		t.Fatalf("flattenProtect = %d rules, want 4", len(rules))
	}

	// Check order: reject, confirm_double, confirm, warn
	if rules[0].Level != LevelReject {
		t.Errorf("rules[0].Level = %q, want reject", rules[0].Level)
	}
	if rules[1].Level != LevelConfirmDouble {
		t.Errorf("rules[1].Level = %q, want confirm_double", rules[1].Level)
	}
	if rules[2].Level != LevelConfirm {
		t.Errorf("rules[2].Level = %q, want confirm", rules[2].Level)
	}
	if rules[3].Level != LevelWarn {
		t.Errorf("rules[3].Level = %q, want warn", rules[3].Level)
	}
}

func TestFlattenProtect_ExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	p := &ProtectConfig{
		Reject: []string{"~/.ssh/**"},
	}
	rules := flattenProtect(p)
	if len(rules) != 1 {
		t.Fatalf("flattenProtect = %d rules, want 1", len(rules))
	}
	want := filepath.Join(home, ".ssh/**")
	if rules[0].Path != want {
		t.Errorf("rules[0].Path = %q, want %q", rules[0].Path, want)
	}
}

func TestGetProtectRules_GlobalOnly(t *testing.T) {
	cfg := DefaultConfig()
	rules := cfg.GetProtectRules("rm")
	if len(rules) == 0 {
		t.Fatal("GetProtectRules(rm) should return global rules")
	}
}

func TestGetProtectRules_WithCommandOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Protect.Command["rm"] = ProtectConfig{
		Reject: []string{"~/Documents/forbidden-to-delete"},
	}
	rules := cfg.GetProtectRules("rm")
	// Should have global rules + command rules
	globalCount := len(cfg.Protect.Reject) + len(cfg.Protect.ConfirmDouble)
	if len(rules) <= globalCount {
		t.Errorf("GetProtectRules should merge global + command rules, got %d rules, global has %d",
			len(rules), globalCount)
	}
}

func TestGetProtectRules_DifferentCommands(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Protect.Command["rm"] = ProtectConfig{
		Reject: []string{"~/Documents/forbidden-to-delete"},
	}
	cfg.Protect.Command["mv"] = ProtectConfig{
		Reject: []string{"~/Projects/important"},
	}

	rmRules := cfg.GetProtectRules("rm")
	mvRules := cfg.GetProtectRules("mv")
	chmodRules := cfg.GetProtectRules("chmod")

	// chmod has no command-specific rules, should only have global (reject + confirm_double)
	globalCount := len(cfg.Protect.Reject) + len(cfg.Protect.ConfirmDouble)
	if len(chmodRules) != globalCount {
		t.Errorf("chmod should only have global rules, got %d, want %d",
			len(chmodRules), globalCount)
	}

	// rm and mv should have more than global
	if len(rmRules) <= len(chmodRules) {
		t.Error("rm should have more rules than chmod (global + command-specific)")
	}
	if len(mvRules) <= len(chmodRules) {
		t.Error("mv should have more rules than chmod (global + command-specific)")
	}
}

func TestLoad_ConfigNotFound(t *testing.T) {
	// Point to a non-existent config dir
	os.Setenv("CMDGUARD_CONFIG_DIR", "/tmp/cmdguard-test-nonexistent")
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error when config not found: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() should return default config")
	}
}

func TestLoad_InvalidToml(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("CMDGUARD_CONFIG_DIR", dir)
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("invalid [[[toml"), 0644)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should error on invalid TOML")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("CMDGUARD_CONFIG_DIR", dir)
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	configContent := `
[protect]
reject = ["/custom/**"]
confirm_double = ["~/.config/**"]
confirm = ["~/custom-confirm/**"]
warn = ["~/custom-warn/**"]

[protect.command.rm]
reject = ["~/custom-rm/**"]

[vault]
retention_days = 7
auto_purge = false
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(configContent), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(cfg.Protect.Reject) != 1 || cfg.Protect.Reject[0] != "/custom/**" {
		t.Errorf("Reject = %v, want [/custom/**]", cfg.Protect.Reject)
	}
	if len(cfg.Protect.ConfirmDouble) != 1 || cfg.Protect.ConfirmDouble[0] != "~/.config/**" {
		t.Errorf("ConfirmDouble = %v, want [~/.config/**]", cfg.Protect.ConfirmDouble)
	}
	if len(cfg.Protect.Confirm) != 1 || cfg.Protect.Confirm[0] != "~/custom-confirm/**" {
		t.Errorf("Confirm = %v, want [~/custom-confirm/**]", cfg.Protect.Confirm)
	}
	if len(cfg.Protect.Warn) != 1 || cfg.Protect.Warn[0] != "~/custom-warn/**" {
		t.Errorf("Warn = %v, want [~/custom-warn/**]", cfg.Protect.Warn)
	}
	if cfg.Vault.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d, want 7", cfg.Vault.RetentionDays)
	}
	if cfg.Vault.AutoPurge {
		t.Error("AutoPurge should be false")
	}

	// Check command-specific rules
	rmPC, ok := cfg.Protect.Command["rm"]
	if !ok {
		t.Fatal("Command rm should exist")
	}
	if len(rmPC.Reject) != 1 || rmPC.Reject[0] != "~/custom-rm/**" {
		t.Errorf("rm reject = %v, want [~/custom-rm/**]", rmPC.Reject)
	}
}

// TestLoad_FileProtectReplacesDefault verifies that when the file defines
// a [protect] section, the file's rules are the single source of truth
// (defaults are NOT merged).
func TestLoad_FileProtectReplacesDefault(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("CMDGUARD_CONFIG_DIR", dir)
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	// File only defines confirm, no reject at all
	configContent := `
[protect]
confirm = ["~/my-only-rule/**"]
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(configContent), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Must NOT have default reject rules (e.g. /var/**)
	if len(cfg.Protect.Reject) != 0 {
		t.Errorf("Reject should be empty (file didn't define it), got %d rules: %v",
			len(cfg.Protect.Reject), cfg.Protect.Reject)
	}
	// Must have exactly the one confirm rule from the file
	if len(cfg.Protect.Confirm) != 1 || cfg.Protect.Confirm[0] != "~/my-only-rule/**" {
		t.Errorf("Confirm = %v, want [~/my-only-rule/**]", cfg.Protect.Confirm)
	}
}

// TestLoad_NoProtectSection_FallsBackToDefaults verifies that when the
// file exists but has no [protect] section, the built-in defaults are used.
func TestLoad_NoProtectSection_FallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("CMDGUARD_CONFIG_DIR", dir)
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	// File only defines vault, no protect at all
	configContent := `
[vault]
retention_days = 7
auto_purge = false
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(configContent), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Must have default reject rules (fallback)
	if len(cfg.Protect.Reject) == 0 {
		t.Fatal("Reject should have default rules when [protect] is absent")
	}
	// Must have default confirm_double rules
	if len(cfg.Protect.ConfirmDouble) == 0 {
		t.Fatal("ConfirmDouble should have default rules when [protect] is absent")
	}
	// Vault should come from the file
	if cfg.Vault.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d, want 7 (from file)", cfg.Vault.RetentionDays)
	}
	if cfg.Vault.AutoPurge {
		t.Error("AutoPurge should be false (from file)")
	}
}

// TestLoad_VaultFieldLevelMerge verifies that only the specific vault
// keys written in the file override defaults; omitted keys keep defaults.
func TestLoad_VaultFieldLevelMerge(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("CMDGUARD_CONFIG_DIR", dir)
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	// Only set retention_days, omit auto_purge
	configContent := `
[vault]
retention_days = 7
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(configContent), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Vault.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d, want 7", cfg.Vault.RetentionDays)
	}
	// auto_purge was NOT in the file, must keep default (true)
	if !cfg.Vault.AutoPurge {
		t.Error("AutoPurge should be true (default), got false")
	}
}

// TestLoad_GuardFieldLevelMerge verifies that only the specific guard
// keys written in the file override defaults; omitted keys keep defaults.
func TestLoad_GuardFieldLevelMerge(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("CMDGUARD_CONFIG_DIR", dir)
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	// Only set confirm_timeout, omit confirm_double_timeout
	configContent := `
[guard]
confirm_timeout = 15
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(configContent), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Guard.ConfirmTimeout != 15 {
		t.Errorf("ConfirmTimeout = %d, want 15", cfg.Guard.ConfirmTimeout)
	}
	// confirm_double_timeout was NOT in the file, must keep default (10)
	if cfg.Guard.ConfirmDoubleTimeout != 10 {
		t.Errorf("ConfirmDoubleTimeout = %d, want 10 (default)", cfg.Guard.ConfirmDoubleTimeout)
	}
}
func TestLoad_EmptyProtectSection_NoDefaults(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("CMDGUARD_CONFIG_DIR", dir)
	defer os.Unsetenv("CMDGUARD_CONFIG_DIR")

	configContent := `
[protect]
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(configContent), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// An empty [protect] section has no defined keys, so it falls back to defaults.
	// This is debatable — but consistent: the file doesn't define any key.
	if len(cfg.Protect.Reject) == 0 {
		t.Fatal("Reject should have default rules when [protect] has no keys")
	}
}
