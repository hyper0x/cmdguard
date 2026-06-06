package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
)

// Backup represents a single vault backup entry
type Backup struct {
	ID        string
	Timestamp time.Time
	Dir       string // full path to the backup directory
	Command   string
	Source    string // original path
	Target    string // for mv: destination path
	Files     []string
}

// Vault manages file backups
type Vault struct {
	dir string
	cfg *config.VaultConfig
}

// New creates a new Vault instance
func New(cfg *config.VaultConfig) (*Vault, error) {
	dir := filepath.Join(config.ConfigDir(), "vault")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建 vault 目录失败: %w", err)
	}
	return &Vault{dir: dir, cfg: cfg}, nil
}

// BackupDir returns the path for a new backup
func (v *Vault) BackupDir(id string) string {
	ts := time.Now().Format("20060102_150405")
	return filepath.Join(v.dir, fmt.Sprintf("%s_%s", ts, id))
}

// SaveFile copies a file into the vault
func (v *Vault) SaveFile(backupDir, srcPath string) (string, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	destName := filepath.Base(srcPath)
	destPath := filepath.Join(backupDir, "files", destName)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("创建 files 目录失败: %w", err)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取源文件失败: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("写入备份文件失败: %w", err)
	}

	return destPath, nil
}

// RestoreFile restores a file from vault to its original location
func (v *Vault) RestoreFile(backupDir, destPath string) error {
	srcName := filepath.Base(destPath)
	srcPath := filepath.Join(backupDir, "files", srcName)

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("恢复文件失败: %w", err)
	}

	return nil
}

// PurgeExpired removes backups older than retention_days
// Returns the list of purged backup IDs
func (v *Vault) PurgeExpired() ([]string, error) {
	if v.cfg.RetentionDays <= 0 {
		return nil, nil
	}

	cutoff := time.Now().AddDate(0, 0, -v.cfg.RetentionDays)
	var purged []string

	entries, err := os.ReadDir(v.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取 vault 目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse timestamp from directory name: 20260606_103005_abc123
		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) < 3 {
			continue
		}

		t, err := time.Parse("20060102_150405", parts[0]+"_"+parts[1])
		if err != nil {
			continue
		}

		if t.Before(cutoff) {
			backupPath := filepath.Join(v.dir, entry.Name())
			if err := os.RemoveAll(backupPath); err != nil {
				fmt.Fprintf(os.Stderr, "[cmdguard] 警告: 删除过期备份 %s 失败: %v\n", entry.Name(), err)
				continue
			}
			purged = append(purged, parts[2]) // the ID
		}
	}

	return purged, nil
}

// ListExpired returns backup directories older than retention_days (without deleting)
func (v *Vault) ListExpired() ([]string, error) {
	if v.cfg.RetentionDays <= 0 {
		return nil, nil
	}

	cutoff := time.Now().AddDate(0, 0, -v.cfg.RetentionDays)
	var expired []string

	entries, err := os.ReadDir(v.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取 vault 目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) < 3 {
			continue
		}

		t, err := time.Parse("20060102_150405", parts[0]+"_"+parts[1])
		if err != nil {
			continue
		}

		if t.Before(cutoff) {
			expired = append(expired, entry.Name())
		}
	}

	sort.Strings(expired)
	return expired, nil
}

// BackupExists checks if a backup directory exists for the given ID
func (v *Vault) BackupExists(id string) bool {
	entries, err := os.ReadDir(v.dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) == 3 && parts[2] == id {
			return true
		}
	}
	return false
}

// FindBackupDir finds the backup directory for a given ID
func (v *Vault) FindBackupDir(id string) string {
	entries, err := os.ReadDir(v.dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) == 3 && parts[2] == id {
			return filepath.Join(v.dir, entry.Name())
		}
	}
	return ""
}
