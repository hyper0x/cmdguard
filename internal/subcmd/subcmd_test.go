package subcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/guard"
)

// ---------------------------------------------------------------------------
// isTerminal
// ---------------------------------------------------------------------------

func TestIsTerminal_Pipe(t *testing.T) {
	// Under `go test`, stdin may be a pseudo-TTY depending on the
	// test runner environment. We cannot reliably assert pipe vs TTY
	// here. The TTY test below (using pty.Open) is the real validation.
	// This test is a best-effort check: if stdin happens to be a pipe,
	// isTerminal must return false.
	stat, _ := os.Stdin.Stat()
	if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		if isTerminal() {
			t.Error("isTerminal() should be false when stdin is a pipe")
		}
	} else {
		t.Log("stdin is a char device in this environment; skipping pipe assertion")
	}
}

func TestIsTerminal_TTY(t *testing.T) {
	// Open a real pseudo-terminal. The slave end is a char device → isTerminal
	// must return true when os.Stdin points to it.
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	oldStdin := os.Stdin
	os.Stdin = slave
	defer func() { os.Stdin = oldStdin }()

	if !isTerminal() {
		t.Error("isTerminal() should be true when stdin is a real TTY")
	}
}

// ---------------------------------------------------------------------------
// readLineWithTimeout
// ---------------------------------------------------------------------------

func TestReadLineWithTimeout_TimeoutFires(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Capture stderr via a temp file.
	tmpFile, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	oldStderr := os.Stderr
	os.Stderr = tmpFile
	defer func() { os.Stderr = oldStderr }()

	start := time.Now()
	line, timedOut := readLineWithTimeout(1) // 1 second timeout
	elapsed := time.Since(start)

	if !timedOut {
		t.Error("expected timeout, got input")
	}
	if line != "" {
		t.Errorf("expected empty line on timeout, got %q", line)
	}
	if elapsed < 900*time.Millisecond {
		t.Errorf("timeout returned too fast: %v", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Errorf("timeout returned too slow: %v", elapsed)
	}

	// Read captured stderr.
	tmpFile.Sync()
	tmpFile.Seek(0, 0)
	var buf bytes.Buffer
	buf.ReadFrom(tmpFile)
	stderr := buf.String()

	if !strings.Contains(stderr, "no input within") {
		t.Errorf("expected timeout message in stderr, got: %s", stderr)
	}
}

func TestReadLineWithTimeout_NoTimeout(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		w.WriteString("y\n")
		w.Close()
	}()

	line, timedOut := readLineWithTimeout(5)
	if timedOut {
		t.Error("unexpected timeout")
	}
	if line != "y\n" && line != "y" {
		t.Errorf("expected 'y', got %q", line)
	}
}

func TestReadLineWithTimeout_Disabled(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		time.Sleep(100 * time.Millisecond)
		w.WriteString("yes\n")
		w.Close()
	}()

	start := time.Now()
	line, timedOut := readLineWithTimeout(0) // 0 = disabled
	elapsed := time.Since(start)

	if timedOut {
		t.Error("unexpected timeout when timeout is disabled")
	}
	if !strings.Contains(line, "yes") {
		t.Errorf("expected 'yes', got %q", line)
	}
	if elapsed < 50*time.Millisecond {
		t.Errorf("returned too fast: %v", elapsed)
	}
}

// ---------------------------------------------------------------------------
// emitNonTTYRejection
// ---------------------------------------------------------------------------

func TestEmitNonTTYRejection_ContainsBypassGuidance(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	tmpFile, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	oldStderr := os.Stderr
	os.Stderr = tmpFile
	defer func() { os.Stderr = oldStderr }()

	result := &guard.Result{
		Action:  "confirm",
		Rule:    "/tmp/test/**",
		Targets: []string{"/tmp/test/foo.txt"},
	}

	emitNonTTYRejection("rm", []string{"/tmp/test/foo.txt"}, []string{"/tmp/test/foo.txt"}, result, reasonNonTTY)

	tmpFile.Sync()
	tmpFile.Seek(0, 0)
	var buf bytes.Buffer
	buf.ReadFrom(tmpFile)
	output := buf.String()

	if !strings.Contains(output, "--bypass") {
		t.Errorf("expected --bypass guidance in output, got:\n%s", output)
	}
	if !strings.Contains(output, "rejected") {
		t.Errorf("expected 'rejected' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "/tmp/test/**") {
		t.Errorf("expected rule in output, got:\n%s", output)
	}
}

func TestEmitNonTTYRejection_LogsEntry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	tmpFile, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	oldStderr := os.Stderr
	os.Stderr = tmpFile
	defer func() { os.Stderr = oldStderr }()

	result := &guard.Result{
		Action:  "confirm",
		Rule:    "/tmp/test/**",
		Targets: []string{"/tmp/test/foo.txt"},
	}

	emitNonTTYRejection("rm", []string{"/tmp/test/foo.txt"}, []string{"/tmp/test/foo.txt"}, result, reasonNonTTY)

	// Verify a log file was created.
	logDir := tmp + "/log"
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no log entries created")
	}
}

func TestEmitNonTTYRejectionTimeout_ContainsTimeoutMsg(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	tmpFile, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	oldStderr := os.Stderr
	os.Stderr = tmpFile
	defer func() { os.Stderr = oldStderr }()

	result := &guard.Result{
		Action:  "confirm",
		Rule:    "/tmp/test/**",
		Targets: []string{"/tmp/test/foo.txt"},
	}

	emitNonTTYRejectionTimeout("rm", []string{"/tmp/test/foo.txt"}, []string{"/tmp/test/foo.txt"}, result, 5)

	tmpFile.Sync()
	tmpFile.Seek(0, 0)
	var buf bytes.Buffer
	buf.ReadFrom(tmpFile)
	output := buf.String()

	if !strings.Contains(output, "--bypass") {
		t.Errorf("expected --bypass guidance in output, got:\n%s", output)
	}
	if !strings.Contains(output, "5") {
		t.Errorf("expected timeout value in output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// RunVault — vault list
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
