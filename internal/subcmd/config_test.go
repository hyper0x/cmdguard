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
// RunConfig
// ---------------------------------------------------------------------------

func TestRunConfig_Effective(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	// Write a minimal config.toml
	cfgPath := filepath.Join(tmp, "config.toml")
	os.WriteFile(cfgPath, []byte("[protect]\nreject = [\"/custom/**\"]\n"), 0644)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunConfig(nil)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "effective configuration") {
		t.Errorf("expected 'effective configuration' header, got:\n%s", out)
	}
	if !strings.Contains(out, "/custom/**") {
		t.Errorf("expected custom rule, got:\n%s", out)
	}
}

func TestRunConfig_Default(t *testing.T) {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunConfig([]string{"--default"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "built-in default configuration") {
		t.Errorf("expected 'built-in default configuration' header, got:\n%s", out)
	}
	if !strings.Contains(out, "/etc/**") {
		t.Errorf("expected default reject rule /etc/**, got:\n%s", out)
	}
	if !strings.Contains(out, "retention_days: 7") {
		t.Errorf("expected default retention_days, got:\n%s", out)
	}
}

func TestRunConfig_Raw(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	cfgPath := filepath.Join(tmp, "config.toml")
	content := "[protect]\nreject = [\"/custom/**\"]\n"
	os.WriteFile(cfgPath, []byte(content), 0644)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunConfig([]string{"--raw"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "raw config file") {
		t.Errorf("expected 'raw config file' header, got:\n%s", out)
	}
	if !strings.Contains(out, "/custom/**") {
		t.Errorf("expected custom rule in raw output, got:\n%s", out)
	}
}

func TestRunConfig_RawNotExist(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunConfig([]string{"--raw"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "does not exist") {
		t.Errorf("expected 'does not exist' message, got:\n%s", out)
	}
}

func TestRunConfig_EffectiveWithCommandRules(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	cfgPath := filepath.Join(tmp, "config.toml")
	content := `[protect]
reject = ["/etc/**"]

[protect.command.rm]
reject = ["~/.ssh/**"]
`
	os.WriteFile(cfgPath, []byte(content), 0644)

	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunConfig(nil)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "effective configuration") {
		t.Errorf("expected 'effective configuration' header, got:\n%s", out)
	}
	if !strings.Contains(out, "[rm]") {
		t.Errorf("expected per-command [rm] section, got:\n%s", out)
	}
	if !strings.Contains(out, "~/.ssh/**") {
		t.Errorf("expected per-command rule, got:\n%s", out)
	}
}

func TestRunConfig_DefaultShowsVaultAndGuard(t *testing.T) {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	RunConfig([]string{"--default"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "retention_days: 7") {
		t.Errorf("expected default retention_days, got:\n%s", out)
	}
	if !strings.Contains(out, "confirm_timeout: 5") {
		t.Errorf("expected default confirm_timeout, got:\n%s", out)
	}
	if !strings.Contains(out, "confirm_double_timeout: 10") {
		t.Errorf("expected default confirm_double_timeout, got:\n%s", out)
	}
}
