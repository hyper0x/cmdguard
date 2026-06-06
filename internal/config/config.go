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

// Config is the top-level configuration
type Config struct {
	Protect ProtectConfig `toml:"protect"`
	Vault   VaultConfig   `toml:"vault"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Protect: ProtectConfig{
			Reject: []string{
				// macOS/Linux 系统目录
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
				// 密钥文件
				"*.key",
				"*.pem",
				"*.crt",
				"*.p12",
				"*.pfx",
				"*.asc",
				// 家目录关键配置 — 不可再生，直接拒绝
				"~/.ssh/**",
				"~/.gnupg/**",
				"~/.aws/**",
			},
			ConfirmDouble: []string{
				// 家目录应用数据 — 可清理但需双重确认
				"~/.config/**",
				"~/.local/share/**",
			},
			Command: map[string]ProtectConfig{},
		},
		Vault: VaultConfig{
			RetentionDays: 30,
			AutoPurge:     true,
		},
	}
}

// ConfigDir returns ~/.cmdguard
func ConfigDir() string {
	if d := os.Getenv("CMDGUARD_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cmdguard")
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

// Load loads configuration from the config file, merging with defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
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
