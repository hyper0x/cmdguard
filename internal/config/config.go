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
// │     non-existent paths), it would erode the trust contract - the    │
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

// PathProtectLevel defines the protection level for a path pattern.
type PathProtectLevel string

const (
	// LevelReject means the path is permanently off-limits.
	// No --bypass can override it. The operation is refused and logged.
	LevelReject PathProtectLevel = "reject"

	// LevelGuarded means the path requires a valid --bypass to proceed.
	// Without --bypass: rejected + logged.
	// With valid --bypass: backup -> log -> execute.
	LevelGuarded PathProtectLevel = "guarded"

	// Legacy levels -- no longer produced by the guard flow, but kept
	// so that old log entries and old config files remain readable.
	// In flattenProtect, all three are mapped to LevelGuarded.
	LevelConfirmDouble PathProtectLevel = "confirm_double"
	LevelConfirm       PathProtectLevel = "confirm"
	LevelWarn          PathProtectLevel = "warn"
)

// PathRule represents a path protection rule.
type PathRule struct {
	Path  string           `toml:"path"`
	Level PathProtectLevel `toml:"level"`
}

// ProtectConfig holds path lists grouped by protection level,
// plus per-command overrides.
//
// Guarded is the canonical field for the new 3-level model.
// ConfirmDouble, Confirm, and Warn are retained for backward
// compatibility with existing config files; their values are merged
// into Guarded during Load().
type ProtectConfig struct {
	Reject        []string                 `toml:"reject"`
	Guarded       []string                 `toml:"guarded"`
	ConfirmDouble []string                 `toml:"confirm_double"` // deprecated -> guarded
	Confirm       []string                 `toml:"confirm"`        // deprecated -> guarded
	Warn          []string                 `toml:"warn"`           // deprecated -> guarded
	Command       map[string]ProtectConfig `toml:"command"`
}

// VaultConfig holds vault settings.
type VaultConfig struct {
	RetentionDays int  `toml:"retention_days"`
	AutoPurge     bool `toml:"auto_purge"`
}

// Config is the top-level configuration.
type Config struct {
	Protect ProtectConfig `toml:"protect"`
	Vault   VaultConfig   `toml:"vault"`
}

// DefaultConfig returns a config with sensible defaults.
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
				// Home directory critical config - non-recoverable, directly rejected
				"~/.ssh/**",
				"~/.gnupg/**",
				"~/.aws/**",
			},
			Guarded: []string{
				// Home directory app data - cleanable but requires --bypass
				"~/.config/**",
				"~/.local/share/**",
			},
			Command: map[string]ProtectConfig{},
		},
		Vault: VaultConfig{
			RetentionDays: 7,
			AutoPurge:     true,
		},
	}
}

// ConfigDir returns ~/.cmdguard.
func ConfigDir() string {
	if d := os.Getenv(EnvConfigDir); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cmdguard")
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

// BinDir returns the directory holding cmdguard wrapper scripts
// (rm/mv/chmod). Currently a child of ConfigDir, but exposed as its
// own function so callers - including the public `cmdguard config
// --bin-dir` command - don't bake the layout into themselves. If the
// wrapper location ever moves (e.g. to /usr/local/libexec/cmdguard
// for a system-wide install), callers using BinDir() pick up the
// change for free.
func BinDir() string {
	return filepath.Join(ConfigDir(), "bin")
}

// Load loads configuration from the config file.
//
// Behaviour:
//   - No config file exists -> returns DefaultConfig() (built-in defaults).
//   - Config file exists but has no [protect] section -> uses DefaultConfig()
//     for protection rules; vault values come from the file if present.
//   - Config file exists with [protect] section -> the file's Protect rules
//     are the single source of truth; defaults are NOT merged. This means
//     if you write `reject = []` explicitly, no reject rules apply at all.
//
// Backward compatibility: the deprecated fields confirm_double, confirm,
// and warn are merged into guarded during Load, so old config files
// continue to work without modification.
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
	// [vault] still gets the built-in protect rules).
	if meta.IsDefined("protect", "reject") ||
		meta.IsDefined("protect", "guarded") ||
		meta.IsDefined("protect", "confirm_double") ||
		meta.IsDefined("protect", "confirm") ||
		meta.IsDefined("protect", "warn") ||
		meta.IsDefined("protect", "command") {
		// Merge deprecated fields into Guarded for backward compat.
		fileCfg.Protect.Guarded = MergeGuarded(fileCfg.Protect)
		cfg.Protect = fileCfg.Protect
	}

	// Vault uses field-level merge: only override the specific
	// keys the file actually defines, so omitted fields keep their defaults.
	if meta.IsDefined("vault", "retention_days") {
		cfg.Vault.RetentionDays = fileCfg.Vault.RetentionDays
	}
	if meta.IsDefined("vault", "auto_purge") {
		cfg.Vault.AutoPurge = fileCfg.Vault.AutoPurge
	}

	return cfg, nil
}

// MergeGuarded combines the Guarded, ConfirmDouble, Confirm, and Warn
// slices into a single slice. This provides backward compatibility:
// old config files that use confirm_double/confirm/warn get those
// entries treated as guarded without requiring a config migration.
// Exported so that subcmd/config.go can merge per-command overrides
// for display.
func MergeGuarded(p ProtectConfig) []string {
	// Preallocate: sum of all source slices.
	result := make([]string, 0, len(p.Guarded)+len(p.ConfirmDouble)+len(p.Confirm)+len(p.Warn))
	result = append(result, p.Guarded...)
	result = append(result, p.ConfirmDouble...)
	result = append(result, p.Confirm...)
	result = append(result, p.Warn...)
	return result
}

// ExpandHome expands ~ or ~/ to the user's home directory.
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

// flattenProtect converts a ProtectConfig into a flat PathRule slice.
//
// Only two protection levels are produced:
//   - reject  (LevelReject)
//   - guarded (LevelGuarded)
//
// The deprecated confirm_double/confirm/warn fields are already merged
// into Guarded by Load(), so flattenProtect only needs to read Reject
// and Guarded.
func flattenProtect(p *ProtectConfig) []PathRule {
	rules := make([]PathRule, 0, len(p.Reject)+len(p.Guarded))
	for _, path := range p.Reject {
		rules = append(rules, PathRule{Path: ExpandHome(path), Level: LevelReject})
	}
	for _, path := range p.Guarded {
		rules = append(rules, PathRule{Path: ExpandHome(path), Level: LevelGuarded})
	}
	return rules
}

// GetProtectRules returns merged protection rules for a given command.
func (c *Config) GetProtectRules(cmd string) []PathRule {
	global := flattenProtect(&c.Protect)

	var cmdRules []PathRule
	if cc, ok := c.Protect.Command[cmd]; ok {
		// Command-level overrides may also use deprecated fields.
		cc.Guarded = MergeGuarded(cc)
		cmdRules = flattenProtect(&cc)
	}

	return append(global, cmdRules...)
}
