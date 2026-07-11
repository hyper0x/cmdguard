package subcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hyper0x/cmdguard/internal/config"
)

// ---------------------------------------------------------------------------
// RunVault - vault list
// ---------------------------------------------------------------------------

func TestRunVaultList_TableOutput(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	// Create vault directory structure with a few backups
	vaultDir := filepath.Join(tmp, "vault")
	createBackup := func(ts string, id string, files ...string) {
		backupPath := filepath.Join(vaultDir, ts+"_"+id, "files")
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(backupPath, f), []byte("data"), 0644); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
	}
	createBackup("20260601_120000", "testid1111111", "doc.txt")
	createBackup("20260602_130000", "testid2222222", "a.txt", "b.txt")
	createBackup("20260603_140000", "testid3333333", "photo.png")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	RunVault([]string{"list"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Must contain table header
	if !strings.Contains(output, "ID") || !strings.Contains(output, "Time") {
		t.Errorf("output missing table header, got:\n%s", output)
	}
	// Must contain backup IDs (truncated to 12 chars)
	if !strings.Contains(output, "testid111111") {
		t.Errorf("output missing testid111111, got:\n%s", output)
	}
	if !strings.Contains(output, "testid222222") {
		t.Errorf("output missing testid222222, got:\n%s", output)
	}
	if !strings.Contains(output, "testid333333") {
		t.Errorf("output missing testid333333, got:\n%s", output)
	}
	// Must contain summary
	if !strings.Contains(output, "total: 3") {
		t.Errorf("output missing summary, got:\n%s", output)
	}
}

func TestRunVaultList_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	vaultDir := filepath.Join(tmp, "vault")
	backupPath := filepath.Join(vaultDir, "20260601_120000_testid1111111", "files")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupPath, "doc.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	RunVault([]string{"list", "--json"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "testid1111111") {
		t.Errorf("JSON output missing testid1111111, got:\n%s", output)
	}
	if !strings.Contains(output, "doc.txt") {
		t.Errorf("JSON output missing doc.txt, got:\n%s", output)
	}
	if !strings.Contains(output, `"id"`) {
		t.Errorf("output doesn't look like JSON, got:\n%s", output)
	}
}

func TestRunVaultList_EmptyVault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	RunVault([]string{"list"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "no vault backups") {
		t.Errorf("expected 'no vault backups' message, got:\n%s", output)
	}
}
