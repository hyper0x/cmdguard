// Package config loads and exposes cmdguard's configuration.
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ ⚠️  IMPORTANT: The config file (~/.cmdguard/config.toml) is        │
// │     READ-ONLY from cmdguard's perspective.                          │
// │                                                                     │
// │     cmdguard MUST NEVER modify, rewrite, or filter the user's       │
// │     config at runtime. Only `cmdguard init` (an explicit user       │
// │     action) is allowed to create or overwrite the file, and that    │
// │     only happens with `--force` (with the old file backed up first).│
// │                                                                     │
// │     Rationale: the config encodes the operator's safety policy.     │
// │     If cmdguard silently "fixed" or pruned entries (e.g. removing   │
// │     non-existent paths), it would erode the trust contract — the    │
// │     user can no longer be sure the active policy matches what they  │
// │     wrote. Any reconciliation between policy and reality belongs    │
// │     in the operator's hands, not in the guard.                      │
// └─────────────────────────────────────────────────────────────────────┘
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// PathProtectLevel defines the protection level for a path pattern
type PathProtectLevel string

const (
	LevelReject         PathProtectLevel = "reject"
	LevelConfirmDouble  PathProtectLevel = "confirm_double"
	LevelConfirm        PathProtectLevel = "confirm"
	LevelWarn           PathProtectLevel = "warn"
)

// PathRule represents a path protection rule
type PathRule struct {
	Path  string           `toml:"path"`
	Level PathProtectLevel `toml:"level"`
}

// ProtectConfig holds path lists grouped by protection level,
// plus per-command overrides
type ProtectConfig struct {
	Reject         []string                   `toml:"reject"`
	ConfirmDouble  []string                   `toml:"confirm_double"`
	Confirm        []string                   `toml:"confirm"`
	Warn           []string                   `toml:"warn"`
	Command        map[string]ProtectConfig   `toml:"command"`
}

// VaultConfig holds vault settings
type VaultConfig struct {
	RetentionDays int  `toml:"retention_days"`
	AutoPurge     bool `toml:"auto_purge"`
}

// GuardConfig holds interactive-prompt settings for protected paths.
//
// Both timeouts use 0 to mean "disable timeout, wait forever". Setting
// 0 is only sensible on personal machines where you trust that no
// automation will ever hit a confirm-level path — otherwise a forgotten
// agent invocation could hang indefinitely.
type GuardConfig struct {
	// ConfirmTimeout is the seconds to wait at a single 'confirm'
	// prompt before falling back to non-interactive rejection.
	// 0 disables the timeout.
	ConfirmTimeout int `toml:"confirm_timeout"`

	// ConfirmDoubleTimeout is the seconds to wait at EACH step of a
	// 'confirm_double' prompt (both the 'y' step and the 'yes' step).
	// 0 disables the timeout. Defaults higher than ConfirmTimeout
	// because confirm_double paths are by definition more dangerous
	// and deserve more deliberation time.
	ConfirmDoubleTimeout int `toml:"confirm_double_timeout"`
}

// Config is the top-level configuration
type Config struct {
	Protect ProtectConfig `toml:"protect"`
	Vault   VaultConfig   `toml:"vault"`
	Guard   GuardConfig   `toml:"guard"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Protect: ProtectConfig{
			Reject: []string{
				// macOS/Linux system directories
				"/bin/**",
				"/boot/**",
				"/dev/**",
				"/etc/**",
				"/lib/**",
				"/lib64/**",
				"/proc/**",
				"/sbin/**",
				"/sys/**",
				"/usr/**",
				"/var/**",
				"/opt/**",
				"/System/**",
				"/Library/**",
				"/Applications/**",
				// Key files
				"*.key",
				"*.pem",
				"*.crt",
				"*.p12",
				"*.pfx",
				"*.asc",
				// Home directory critical config — non-recoverable, directly rejected
				"~/.ssh/**",
				"~/.gnupg/**",
				"~/.aws/**",
			},
			ConfirmDouble: []string{
				// Home directory app data — cleanable but requires double confirmation
				"~/.config/**",
				"~/.local/share/**",
			},
			Command: map[string]ProtectConfig{},
		},
		Vault: VaultConfig{
			RetentionDays: 7,
			AutoPurge:     true,
		},
		Guard: GuardConfig{
			ConfirmTimeout:       5,
			ConfirmDoubleTimeout: 10,
		},
	}
}

// ConfigDir returns ~/.cmdguard
func ConfigDir() string {
	if d := os.Getenv(EnvConfigDir); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cmdguard")
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

// Load loads configuration from the config file.
//
// Behaviour:
//   - No config file exists → returns DefaultConfig() (built-in defaults).
//   - Config file exists but has no [protect] section → uses DefaultConfig()
//     for protection rules; vault/guard values come from the file if present.
//   - Config file exists with [protect] section → the file's Protect rules
//     are the single source of truth; defaults are NOT merged. This means
//     if you write `reject = []` explicitly, no reject rules apply at all.
//
// Vault and Guard use field-level merge: only the specific keys written
// in the file override the defaults; omitted keys keep their default values.
func Load() (*Config, error) {
	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Decode into a fresh config so we can inspect what the file actually defines.
	var fileCfg Config
	meta, err := toml.DecodeFile(path, &fileCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Start with defaults.
	cfg := DefaultConfig()

	// If the file defines any [protect] key, use the file's Protect as-is.
	// Otherwise fall back to defaults (so a minimal file that only sets
	// [vault] or [guard] still gets the built-in protect rules).
	if meta.IsDefined("protect", "reject") ||
		meta.IsDefined("protect", "confirm_double") ||
		meta.IsDefined("protect", "confirm") ||
		meta.IsDefined("protect", "warn") ||
		meta.IsDefined("protect", "command") {
		cfg.Protect = fileCfg.Protect
	}

	// Vault and Guard use field-level merge: only override the specific
	// keys the file actually defines, so omitted fields keep their defaults.
	if meta.IsDefined("vault", "retention_days") {
		cfg.Vault.RetentionDays = fileCfg.Vault.RetentionDays
	}
	if meta.IsDefined("vault", "auto_purge") {
		cfg.Vault.AutoPurge = fileCfg.Vault.AutoPurge
	}
	if meta.IsDefined("guard", "confirm_timeout") {
		cfg.Guard.ConfirmTimeout = fileCfg.Guard.ConfirmTimeout
	}
	if meta.IsDefined("guard", "confirm_double_timeout") {
		cfg.Guard.ConfirmDoubleTimeout = fileCfg.Guard.ConfirmDoubleTimeout
	}

	return cfg, nil
}

// ExpandHome expands ~ or ~/ to the user's home directory
func ExpandHome(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// flattenProtect converts a ProtectConfig into a flat PathRule slice
func flattenProtect(p *ProtectConfig) []PathRule {
	var rules []PathRule
	for _, path := range p.Reject {
		rules = append(rules, PathRule{Path: ExpandHome(path), Level: LevelReject})
	}
	for _, path := range p.ConfirmDouble {
		rules = append(rules, PathRule{Path: ExpandHome(path), Level: LevelConfirmDouble})
	}
	for _, path := range p.Confirm {
		rules = append(rules, PathRule{Path: ExpandHome(path), Level: LevelConfirm})
	}
	for _, path := range p.Warn {
		rules = append(rules, PathRule{Path: ExpandHome(path), Level: LevelWarn})
	}
	return rules
}

// GetProtectRules returns merged protection rules for a given command
func (c *Config) GetProtectRules(cmd string) []PathRule {
	global := flattenProtect(&c.Protect)

	var cmdRules []PathRule
	if cc, ok := c.Protect.Command[cmd]; ok {
		cmdRules = flattenProtect(&cc)
	}

	return append(global, cmdRules...)
}
