package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
)

// setupTestVault creates an isolated vault rooted at t.TempDir().
// It uses CMDGUARD_CONFIG_DIR so the vault writes under the temp tree.
func setupTestVault(t *testing.T, cfg *config.VaultConfig) (*Vault, string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	if cfg == nil {
		cfg = &config.VaultConfig{RetentionDays: 1, AutoPurge: true}
	}
	v, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v, tmp
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestSaveAndRestoreFile(t *testing.T) {
	v, tmp := setupTestVault(t, nil)

	src := filepath.Join(tmp, "work", "doc.txt")
	writeFile(t, src, "hello vault")

	backupDir := v.BackupDir("abc123def456")
	dest, err := v.SaveFile(backupDir, src)
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	// Backup file exists and content matches
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != "hello vault" {
		t.Errorf("backup content mismatch: %q", got)
	}

	// Simulate the destructive op
	if err := os.Remove(src); err != nil {
		t.Fatalf("remove src: %v", err)
	}

	// Restore from vault
	if err := v.RestoreFile(backupDir, src); err != nil {
		t.Fatalf("RestoreFile: %v", err)
	}
	restored, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if string(restored) != "hello vault" {
		t.Errorf("restored content mismatch: %q", restored)
	}
}

func TestSaveFile_SourceMissing(t *testing.T) {
	v, tmp := setupTestVault(t, nil)
	backupDir := v.BackupDir("missing00")
	_, err := v.SaveFile(backupDir, filepath.Join(tmp, "nonexistent.txt"))
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}

func TestBackupDir_FormatAndUniqueness(t *testing.T) {
	v, _ := setupTestVault(t, nil)
	d1 := v.BackupDir("idsame00idsame00")
	// Different ID → different dir
	d2 := v.BackupDir("idnewxxidnewxx00")
	if d1 == d2 {
		t.Errorf("expected different dirs for different ids; got %q == %q", d1, d2)
	}
	// Dir name must match the parser convention: YYYYMMDD_HHMMSS_<id>
	base := filepath.Base(d1)
	parts := strings.SplitN(base, "_", 3)
	if len(parts) != 3 {
		t.Errorf("BackupDir name %q must have 3 underscore-separated parts", base)
	}
	if _, err := time.Parse("20060102_150405", parts[0]+"_"+parts[1]); err != nil {
		t.Errorf("BackupDir timestamp parse failed: %v", err)
	}
}

// fabricateBackup creates a backup directory with a backdated timestamp,
// bypassing v.BackupDir() (which always uses time.Now).
func fabricateBackup(t *testing.T, vaultRoot string, when time.Time, id string) string {
	t.Helper()
	name := when.Format("20060102_150405") + "_" + id
	dir := filepath.Join(vaultRoot, name)
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o755); err != nil {
		t.Fatalf("fabricate: %v", err)
	}
	writeFile(t, filepath.Join(dir, "files", "x.txt"), "old")
	return dir
}

func TestPurgeExpired(t *testing.T) {
	v, tmp := setupTestVault(t, &config.VaultConfig{RetentionDays: 7, AutoPurge: true})
	vaultRoot := filepath.Join(tmp, "vault")

	// 3 backups: 30 days old (expired), 1 day old (kept), today (kept)
	fabricateBackup(t, vaultRoot, time.Now().AddDate(0, 0, -30), "oldid000aaaa")
	fabricateBackup(t, vaultRoot, time.Now().AddDate(0, 0, -1), "newid000bbbb")
	fabricateBackup(t, vaultRoot, time.Now(), "todayidcccc0")

	purged, err := v.PurgeExpired()
	if err != nil {
		t.Fatalf("PurgeExpired: %v", err)
	}
	if len(purged) != 1 {
		t.Fatalf("expected 1 purged, got %d: %v", len(purged), purged)
	}
	if purged[0] != "oldid000aaaa" {
		t.Errorf("expected purged=oldid000aaaa, got %q", purged[0])
	}

	// Confirm kept backups still exist
	entries, _ := os.ReadDir(vaultRoot)
	if len(entries) != 2 {
		t.Errorf("expected 2 backups left, got %d", len(entries))
	}
}

func TestPurgeExpired_NoRetention(t *testing.T) {
	v, tmp := setupTestVault(t, &config.VaultConfig{RetentionDays: 0})
	vaultRoot := filepath.Join(tmp, "vault")
	fabricateBackup(t, vaultRoot, time.Now().AddDate(0, 0, -100), "ancientxxx0")

	purged, err := v.PurgeExpired()
	if err != nil {
		t.Fatalf("PurgeExpired: %v", err)
	}
	if len(purged) != 0 {
		t.Errorf("RetentionDays=0 must not purge; got %v", purged)
	}
}

func TestListExpired_DoesNotDelete(t *testing.T) {
	v, tmp := setupTestVault(t, &config.VaultConfig{RetentionDays: 7})
	vaultRoot := filepath.Join(tmp, "vault")
	fabricateBackup(t, vaultRoot, time.Now().AddDate(0, 0, -30), "expirerxxx0")
	fabricateBackup(t, vaultRoot, time.Now(), "freshxxxxxx0")

	expired, err := v.ListExpired()
	if err != nil {
		t.Fatalf("ListExpired: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired, got %d", len(expired))
	}
	// Must NOT have actually deleted anything
	entries, _ := os.ReadDir(vaultRoot)
	if len(entries) != 2 {
		t.Errorf("ListExpired must not delete; got %d entries left", len(entries))
	}
}

func TestFindBackupDir(t *testing.T) {
	v, tmp := setupTestVault(t, nil)
	vaultRoot := filepath.Join(tmp, "vault")

	fabricateBackup(t, vaultRoot, time.Now(), "uniqueidaaaa")
	fabricateBackup(t, vaultRoot, time.Now().Add(-time.Second), "ambig111bbbb")
	fabricateBackup(t, vaultRoot, time.Now().Add(-2*time.Second), "ambig222cccc")

	// Exact match
	if got := v.FindBackupDir("uniqueidaaaa"); got == "" {
		t.Errorf("expected to find uniqueidaaaa, got empty")
	}
	// Prefix match (unambiguous)
	if got := v.FindBackupDir("uniqueid"); got == "" {
		t.Errorf("expected unambiguous prefix match")
	}
	// Ambiguous prefix → empty
	if got := v.FindBackupDir("ambig"); got != "" {
		t.Errorf("expected empty for ambiguous prefix, got %q", got)
	}
	// Not found
	if got := v.FindBackupDir("noexist000"); got != "" {
		t.Errorf("expected empty for missing id, got %q", got)
	}
}

func TestBackupExists(t *testing.T) {
	v, tmp := setupTestVault(t, nil)
	vaultRoot := filepath.Join(tmp, "vault")
	fabricateBackup(t, vaultRoot, time.Now(), "existidxxx00")

	if !v.BackupExists("existidxxx00") {
		t.Errorf("expected BackupExists=true")
	}
	if !v.BackupExists("existid") {
		t.Errorf("expected BackupExists=true for prefix")
	}
	if v.BackupExists("missing") {
		t.Errorf("expected BackupExists=false for missing id")
	}
}
