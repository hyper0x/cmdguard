package subcmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hyper0x/cmdguard/internal/config"
)

// ---------------------------------------------------------------------------
// formatFileSize
// ---------------------------------------------------------------------------

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0B"},
		{1, "1B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
		{1610612736, "1.5GB"},
	}
	for _, tt := range tests {
		got := formatFileSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatFileSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// printDirFiles
// ---------------------------------------------------------------------------

// setupPathEnv creates a temp cmdguard directory structure and returns the root.
// It sets CMDGUARD_CONFIG_DIR so config.ConfigDir() points to tmp.
func setupPathEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)
	return tmp
}

func TestPrintDirFiles_ShowsFiles(t *testing.T) {
	tmp := setupPathEnv(t)
	dir := filepath.Join(tmp, "log")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "2026-06-03.log"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "2026-06-02.log"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "2026-06-01.log"), []byte("c"), 0644)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printDirFiles(dir, ".log")

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "2026-06-03.log") {
		t.Errorf("missing newest file, got:\n%s", out)
	}
	if !strings.Contains(out, "2026-06-01.log") {
		t.Errorf("missing oldest file, got:\n%s", out)
	}
	if !strings.Contains(out, "3 file(s)") {
		t.Errorf("missing count, got:\n%s", out)
	}
}

func TestPrintDirFiles_ShowsOnlyFirst5(t *testing.T) {
	tmp := setupPathEnv(t)
	dir := filepath.Join(tmp, "log")
	os.MkdirAll(dir, 0755)
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("2026-06-%02d.log", i+1)), []byte("a"), 0644)
	}

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printDirFiles(dir, ".log")

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "...") {
		t.Errorf("expected '...' for more than 5 files, got:\n%s", out)
	}
	if !strings.Contains(out, "10 file(s)") {
		t.Errorf("expected total count, got:\n%s", out)
	}
}

func TestPrintDirFiles_EmptyDir(t *testing.T) {
	tmp := setupPathEnv(t)
	dir := filepath.Join(tmp, "empty")
	os.MkdirAll(dir, 0755)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printDirFiles(dir, ".log")

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "empty") {
		t.Errorf("expected 'empty', got:\n%s", out)
	}
}

func TestPrintDirFiles_NotExist(t *testing.T) {
	tmp := setupPathEnv(t)
	dir := filepath.Join(tmp, "nonexistent")

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printDirFiles(dir, ".log")

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "not exist") {
		t.Errorf("expected 'not exist', got:\n%s", out)
	}
}

func TestPrintDirFiles_FiltersByExtension(t *testing.T) {
	tmp := setupPathEnv(t)
	dir := filepath.Join(tmp, "mixed")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "a.log"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "c.log"), []byte("c"), 0644)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printDirFiles(dir, ".log")

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if strings.Contains(out, "b.txt") {
		t.Errorf("b.txt should be filtered out, got:\n%s", out)
	}
	if !strings.Contains(out, "2 file(s)") {
		t.Errorf("expected 2 log files, got:\n%s", out)
	}
}

func TestPrintDirFiles_NoExtFilter(t *testing.T) {
	tmp := setupPathEnv(t)
	dir := filepath.Join(tmp, "bin")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "rm"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(dir, "mv"), []byte("#!/bin/sh"), 0755)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printDirFiles(dir, "")

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "rm") || !strings.Contains(out, "mv") {
		t.Errorf("expected both files, got:\n%s", out)
	}
	if !strings.Contains(out, "2 file(s)") {
		t.Errorf("expected count, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// printVaultInfo
// ---------------------------------------------------------------------------

func TestPrintVaultInfo_ShowsSummary(t *testing.T) {
	tmp := setupPathEnv(t)
	vaultDir := filepath.Join(tmp, "vault")

	// Create 2 backup dirs with files
	b1 := filepath.Join(vaultDir, "20260601_120000_backup1", "files")
	os.MkdirAll(b1, 0755)
	os.WriteFile(filepath.Join(b1, "doc.txt"), []byte("hello"), 0644)

	b2 := filepath.Join(vaultDir, "20260602_130000_backup2", "files")
	os.MkdirAll(b2, 0755)
	os.WriteFile(filepath.Join(b2, "data.txt"), []byte("world"), 0644)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printVaultInfo(vaultDir)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "2 backup(s)") {
		t.Errorf("expected 2 backups, got:\n%s", out)
	}
	if !strings.Contains(out, "B") {
		t.Errorf("expected size info, got:\n%s", out)
	}
}

func TestPrintVaultInfo_EmptyVault(t *testing.T) {
	tmp := setupPathEnv(t)
	vaultDir := filepath.Join(tmp, "vault")
	os.MkdirAll(vaultDir, 0755)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printVaultInfo(vaultDir)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "empty") {
		t.Errorf("expected 'empty', got:\n%s", out)
	}
}

func TestPrintVaultInfo_NotExist(t *testing.T) {
	tmp := setupPathEnv(t)
	vaultDir := filepath.Join(tmp, "nonexistent")

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	printVaultInfo(vaultDir)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "not exist") {
		t.Errorf("expected 'not exist', got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// RunPath
// ---------------------------------------------------------------------------

func TestRunPath_AllExist(t *testing.T) {
	tmp := setupPathEnv(t)

	// Config file
	configPath := filepath.Join(tmp, "config.toml")
	os.WriteFile(configPath, []byte("[protect]\nreject = []\n"), 0644)

	// Log dir with files
	logDir := filepath.Join(tmp, "log")
	os.MkdirAll(logDir, 0755)
	os.WriteFile(filepath.Join(logDir, "2026-06-03.log"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(logDir, "2026-06-02.log"), []byte("b"), 0644)

	// Vault dir with backups
	vaultDir := filepath.Join(tmp, "vault")
	b1 := filepath.Join(vaultDir, "20260601_120000_backup1", "files")
	os.MkdirAll(b1, 0755)
	os.WriteFile(filepath.Join(b1, "doc.txt"), []byte("hello"), 0644)

	// Bin dir with scripts
	binDir := filepath.Join(tmp, "bin")
	os.MkdirAll(binDir, 0755)
	os.WriteFile(filepath.Join(binDir, "rm"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(binDir, "mv"), []byte("#!/bin/sh"), 0755)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunPath(nil)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// Check all sections present
	if !strings.Contains(out, "Config directory") {
		t.Errorf("missing Config directory section")
	}
	if !strings.Contains(out, "Config file") {
		t.Errorf("missing Config file section")
	}
	if !strings.Contains(out, "Log directory") {
		t.Errorf("missing Log directory section")
	}
	if !strings.Contains(out, "Vault directory") {
		t.Errorf("missing Vault directory section")
	}
	if !strings.Contains(out, "Bin directory") {
		t.Errorf("missing Bin directory section")
	}

	// Check config file details
	if !strings.Contains(out, "2 lines") {
		t.Errorf("missing line count, got:\n%s", out)
	}

	// Check log files
	if !strings.Contains(out, "2026-06-03.log") {
		t.Errorf("missing log file, got:\n%s", out)
	}
	if !strings.Contains(out, "2 file(s)") {
		t.Errorf("missing log count, got:\n%s", out)
	}

	// Check vault
	if !strings.Contains(out, "1 backup(s)") {
		t.Errorf("missing vault summary, got:\n%s", out)
	}

	// Check bin
	if !strings.Contains(out, "rm") {
		t.Errorf("missing rm script, got:\n%s", out)
	}
	if !strings.Contains(out, "2 file(s)") {
		t.Errorf("missing bin count, got:\n%s", out)
	}
}

func TestRunPath_AllDirsNotExist(t *testing.T) {
	_ = setupPathEnv(t)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunPath(nil)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// All directories should show "not exist"
	if !strings.Contains(out, "not exist") {
		t.Errorf("expected 'not exist' for missing dirs, got:\n%s", out)
	}
}
