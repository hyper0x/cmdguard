// e2e_test.go runs cmdguard as a real subprocess against an isolated
// CMDGUARD_CONFIG_DIR. It exercises the full user-visible flow:
//
//	init → rm (with --bypass) → list → undo → verify restored
//
// Why a separate binary build: testing.Main can't simulate alias hijacking
// or stdin TTY behaviour. A real subprocess with an isolated config dir
// is the closest we can get to "what the user actually sees".
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	binPath  string
	binOnce  sync.Once
	errBinBuild error
)

// safeEnv returns a minimal environment for the subprocess that:
//   - points CMDGUARD_CONFIG_DIR to the test's isolated directory
//   - sets a clean PATH with only system dirs (no user cmdguard wrappers)
//   - sets CMDGUARD_NONINTERACTIVE=1 to avoid TTY detection flakiness
func safeEnv(configDir string) []string {
	return []string{
		"PATH=/bin:/usr/bin:/usr/sbin:/sbin:/usr/local/bin",
		"CMDGUARD_CONFIG_DIR=" + configDir,
		"CMDGUARD_NONINTERACTIVE=1",
		"HOME=" + filepath.Dir(configDir),
	}
}

// buildOnce compiles cmdguard into a temp binary, reused across tests.
func buildOnce(t *testing.T) string {
	t.Helper()
	binOnce.Do(func() {
		dir, err := os.MkdirTemp("", "cmdguard-e2e-bin-*")
		if err != nil {
			errBinBuild = err
			return
		}
		binPath = filepath.Join(dir, "cmdguard")
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			errBinBuild = err
		}
	})
	if errBinBuild != nil {
		t.Fatalf("build cmdguard: %v", errBinBuild)
	}
	return binPath
}

// run executes the cmdguard binary inside the given isolated config dir
// with a clean PATH and CMDGUARD_NONINTERACTIVE=1.
func run(t *testing.T, configDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = safeEnv(configDir)
	cmd.Stdin = bytes.NewReader(nil)

	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	return so.String(), se.String(), exitCode
}

// TestE2E_FullFlow runs the end-to-end happy path that an AI agent
// would follow:
//
//  1. cmdguard init             → populate config + bin + dirs
//  2. Write config with confirm rule for ~/playground/**
//  3. Create a file under that path
//  4. cmdguard rm <file> --bypass=<id>
//     → must succeed (bypass forces allow, log gets bypass field)
//  5. Verify file is gone, log contains bypass identifier
//  6. cmdguard list --json      → find the entry
//  7. cmdguard undo --id <id>   → must restore the file
//  8. Verify file is back with original content
func TestE2E_FullFlow(t *testing.T) {
	buildOnce(t)

	configDir := filepath.Join(t.TempDir(), ".cmdguard")
	playground := filepath.Join(configDir, "..", "playground")
	// Resolve to absolute path
	playground, _ = filepath.Abs(playground)

	// 1. init
	out, errOut, code := run(t, configDir, "init")
	if code != 0 {
		t.Fatalf("init failed (code=%d):\nstdout=%s\nstderr=%s", code, out, errOut)
	}
	cfgPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}

	// 2. Write a config with explicit empty reject/double/warn lists.
	//    Without "reject = []" the TOML decoder merges with DefaultConfig
	//    (which includes /var/** — macOS TempDir lives under /var).
	cfg := `[protect]
reject = []
confirm_double = []
confirm = ["` + playground + `/**"]
warn = []

[vault]
retention_days = 7
auto_purge = true

[guard]
confirm_timeout = 5
confirm_double_timeout = 10
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 3. Create a target file.
	if err := os.MkdirAll(playground, 0o755); err != nil {
		t.Fatalf("mkdir playground: %v", err)
	}
	target := filepath.Join(playground, "important.txt")
	const content = "this file matters"
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	// 4. rm with bypass (CMDGUARD_NONINTERACTIVE=1 is set by safeEnv,
	//    and bypass provides the permission to proceed).
	bypassID := "ci-host/e2e-test/cmdguard/full-flow"
	out, errOut, code = run(t, configDir, "rm", target, "--bypass="+bypassID)
	if code != 0 {
		t.Fatalf("rm --bypass failed (code=%d):\nstdout=%s\nstderr=%s", code, out, errOut)
	}

	// 5a. File must be gone.
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target should be deleted, but stat returned err=%v", err)
	}

	// 5b. Log file must contain the bypass identifier.
	var logRaw []byte
	_ = filepath.Walk(filepath.Join(configDir, "log"),
		func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".log") {
				b, _ := os.ReadFile(path)
				logRaw = append(logRaw, b...)
			}
			return nil
		})
	if !strings.Contains(string(logRaw), bypassID) {
		t.Errorf("log file missing bypass identifier %q:\n%s", bypassID, logRaw)
	}

	// 6. list --json → parse and find our entry's id
	out, errOut, code = run(t, configDir, "list", "--json")
	if code != 0 {
		t.Fatalf("list --json failed (code=%d):\nstderr=%s", code, errOut)
	}
	var entries []struct {
		ID      string `json:"id"`
		Command string `json:"command"`
		Targets string `json:"targets"`
		Bypass  string `json:"bypass"`
	}
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		t.Fatalf("parse list --json: %v\nraw: %s", err, out)
	}
	var rmID string
	for _, e := range entries {
		if e.Command == "rm" && e.Targets == target {
			rmID = e.ID
			if e.Bypass != bypassID {
				t.Errorf("list entry bypass mismatch: got %q want %q", e.Bypass, bypassID)
			}
			break
		}
	}
	if rmID == "" {
		t.Fatalf("rm entry not found in list output:\n%s", out)
	}

	// 7. undo --id
	out, errOut, code = run(t, configDir, "undo", "--id", rmID)
	if code != 0 {
		t.Fatalf("undo failed (code=%d):\nstdout=%s\nstderr=%s", code, out, errOut)
	}

	// 8. File restored with original content
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target not restored: %v", err)
	}
	if string(got) != content {
		t.Errorf("restored content mismatch: got %q want %q", got, content)
	}
}

// TestE2E_NonInteractiveRejection verifies that without --bypass,
// cmdguard refuses (fast, no hang) and gives the agent the bypass
// guidance in stderr. With CMDGUARD_NONINTERACTIVE=1 it must be
// immediate (0-delay).
func TestE2E_NonInteractiveRejection(t *testing.T) {
	buildOnce(t)

	configDir := filepath.Join(t.TempDir(), ".cmdguard")
	playground := filepath.Join(configDir, "..", "playground")
	playground, _ = filepath.Abs(playground)

	_, _, code := run(t, configDir, "init")
	if code != 0 {
		t.Fatal("init failed")
	}
	cfgPath := filepath.Join(configDir, "config.toml")
	cfg := `[protect]
reject = []
confirm_double = []
confirm = ["` + playground + `/**"]
warn = []

[vault]
retention_days = 7
auto_purge = true

[guard]
confirm_timeout = 5
confirm_double_timeout = 10
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_ = os.MkdirAll(playground, 0o755)
	target := filepath.Join(playground, "x.txt")
	_ = os.WriteFile(target, []byte("data"), 0o644)

	// Run WITHOUT --bypass, WITH CMDGUARD_NONINTERACTIVE=1
	cmd := exec.Command(binPath, "rm", target)
	cmd.Env = safeEnv(configDir)
	cmd.Stdin = bytes.NewReader(nil)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se

	err := cmd.Run()
	exitCode := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		exitCode = ee.ExitCode()
	}

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0. stderr=%s", se.String())
	}
	// File must NOT have been deleted.
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target was deleted despite rejection: %v", err)
	}
	// Stderr must contain --bypass guidance.
	if !strings.Contains(se.String(), "--bypass") {
		t.Errorf("expected --bypass hint in stderr, got:\n%s", se.String())
	}
}