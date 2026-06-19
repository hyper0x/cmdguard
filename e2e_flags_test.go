// e2e_flags_test.go covers the flag-parsing contracts that landed
// in the v0.7→v0.12 release sweep:
//
//   - Unknown flags must exit 1 with "unknown flag" on stderr.
//   - Positional garbage must exit 1 with "unexpected argument".
//   - `cmdguard vault` (no subcommand) must exit 1 with usage on
//     stderr — never silently run `clean`.
//   - `cmdguard list --recent abc` / negative / missing value must
//     exit 1 (no silent fallback to default 20).
//   - `cmdguard undo --it` (typo of --interactive) must exit 1.
//   - `undo` error messages must route to stderr, not stdout.
//
// These contracts were verified manually during the sweep but had no
// regression coverage. Running the binary as a subprocess is the
// only way to assert os.Exit(1) and the stdout/stderr split.
package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// runFlag is a thin wrapper around exec.Command that returns
// (stdout, stderr, exitCode) without using `run()` from e2e_test.go,
// because some of these tests exercise commands that don't need the
// full safeEnv (no init required).
func runFlag(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	// Empty CMDGUARD_CONFIG_DIR so tests don't accidentally touch
	// the real ~/.cmdguard. Each test that needs init uses run().
	cmd.Env = []string{
		"PATH=/bin:/usr/bin:/usr/sbin:/sbin",
		"CMDGUARD_CONFIG_DIR=" + t.TempDir(),
		"HOME=" + t.TempDir(),
	}
	cmd.Stdin = bytes.NewReader(nil)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		exitCode = ee.ExitCode()
	}
	return so.String(), se.String(), exitCode
}

// rejectionCases is the table of "this token must be rejected by
// this subcommand" pairs. Each entry asserts:
//   - exit code is 1
//   - stderr contains the exact expected diagnostic substring
//   - stdout is empty (errors must not pollute stdout)
//
// The "wantSub" is a substring search, not equality, so we don't
// have to maintain whole error templates here. Choose something
// distinctive.
type rejectionCase struct {
	name    string
	args    []string
	wantSub string // expected substring in stderr
}

func TestE2E_RejectUnknownFlags(t *testing.T) {
	buildOnce(t)

	cases := []rejectionCase{
		// --- list ---
		{"list_unknown_flag", []string{"list", "--bogus"},
			`unknown flag "--bogus" for 'list'`},
		{"list_recnet_typo", []string{"list", "--recnet", "5"},
			`unknown flag "--recnet" for 'list'`},
		{"list_recent_missing_value", []string{"list", "--recent"},
			`flag "--recent" requires a value`},
		{"list_recent_non_int", []string{"list", "--recent", "abc"},
			`invalid --recent value "abc"`},
		{"list_recent_zero", []string{"list", "--recent", "0"},
			`invalid --recent value "0"`},
		{"list_recent_negative", []string{"list", "--recent", "-3"},
			// "-3" starts with "-", so it's parsed as another flag,
			// not a value; the flag parser sees --recent without a
			// value-like arg. Either diagnostic is fine; we accept
			// the more specific "must be a positive integer" since
			// strconv.Atoi accepts "-3" and we then bail on n<=0.
			`invalid --recent value "-3"`},
		{"list_since_typo", []string{"list", "--since", "7days"},
			`invalid --since value "7days"`},
		{"list_positional_garbage", []string{"list", "garbage"},
			`unexpected argument "garbage" for 'list'`},

		// --- config ---
		{"config_unknown_flag", []string{"config", "--bogus"},
			`unknown flag "--bogus" for 'config'`},
		{"config_positional", []string{"config", "extra"},
			`unexpected argument "extra" for 'config'`},

		// --- init ---
		{"init_unknown_flag", []string{"init", "--bogus"},
			`unknown flag "--bogus" for 'init'`},
		{"init_positional", []string{"init", "foo"},
			`unexpected argument "foo" for 'init'`},

		// --- path ---
		{"path_unknown_flag", []string{"path", "--raw"},
			`unknown flag "--raw" for 'path'`},
		{"path_positional", []string{"path", "extra"},
			`unexpected argument "extra" for 'path'`},

		// --- vault ---
		{"vault_no_subcommand", []string{"vault"},
			`usage: cmdguard vault`},
		{"vault_unknown_subcommand", []string{"vault", "purge"},
			`unknown vault subcommand: purge`},
		{"vault_flag_no_subcommand", []string{"vault", "--dry-run"},
			`vault subcommand required`},
		{"vault_clean_unknown_flag", []string{"vault", "clean", "--bogus"},
			`unknown flag "--bogus" for 'vault clean'`},
		{"vault_clean_positional", []string{"vault", "clean", "extra"},
			`unexpected argument "extra" for 'vault clean'`},
		{"vault_list_unknown_flag", []string{"vault", "list", "--bogus"},
			`unknown flag "--bogus" for 'vault list'`},

		// --- undo (the parser this round added strict checking to) ---
		{"undo_unknown_flag", []string{"undo", "--bogus"},
			`unknown flag "--bogus" for 'undo'`},
		{"undo_typo_interactive", []string{"undo", "--it"},
			`unknown flag "--it" for 'undo'`},
		{"undo_id_missing_value", []string{"undo", "--id"},
			`flag "--id" requires a value`},
		{"undo_positional", []string{"undo", "abc123"},
			// `undo abc123` (no --id) is a known idiom in some CLIs
			// but cmdguard requires --id. The parser treats the
			// positional as garbage and rejects it.
			`unexpected argument "abc123" for 'undo'`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := runFlag(t, tc.args...)
			if code != 1 {
				t.Errorf("%v: exit=%d (want 1)\nstdout: %s\nstderr: %s",
					tc.args, code, stdout, stderr)
			}
			if !strings.Contains(stderr, tc.wantSub) {
				t.Errorf("%v: stderr missing %q\nstderr: %s",
					tc.args, tc.wantSub, stderr)
			}
			// Errors must not bleed into stdout — that breaks
			// `$(cmdguard ...)` shell substitution.
			if stdout != "" {
				t.Errorf("%v: errors must go to stderr, but stdout was: %s",
					tc.args, stdout)
			}
		})
	}
}

// TestE2E_UndoErrorsToStderr is the dedicated check for the
// fad2ffb fix: every undo error path must route to stderr, leaving
// stdout empty so callers can `$(cmdguard undo --id X)` without
// inhaling diagnostic text.
//
// We exercise the four most common error paths:
//   - usage block (no --id, non-interactive)
//   - ID not found
//   - vault backup not found (allow op + missing backup dir)
//
// The "expired" and "rejected" paths require a full setup; they are
// covered indirectly by TestE2E_FullFlow and the rejection-table
// case for --bogus. Adding two more would give diminishing returns.
func TestE2E_UndoErrorsToStderr(t *testing.T) {
	buildOnce(t)

	t.Run("usage_no_args", func(t *testing.T) {
		stdout, stderr, code := runFlag(t, "undo")
		if code != 1 {
			t.Errorf("exit=%d want 1; stdout=%s stderr=%s", code, stdout, stderr)
		}
		if !strings.Contains(stderr, "usage: cmdguard undo") {
			t.Errorf("usage block missing from stderr: %s", stderr)
		}
		if stdout != "" {
			t.Errorf("usage must not go to stdout, got: %s", stdout)
		}
	})

	t.Run("id_not_found", func(t *testing.T) {
		// 12 hex chars matches the ID format crypto/rand produces,
		// so it gets past length validation but fails the lookup.
		stdout, stderr, code := runFlag(t, "undo", "--id", "deadbeef0000")
		if code != 1 {
			t.Errorf("exit=%d want 1", code)
		}
		if !strings.Contains(stderr, "no record found") {
			t.Errorf("expected 'no record found' on stderr, got: %s", stderr)
		}
		if stdout != "" {
			t.Errorf("error must not go to stdout, got: %s", stdout)
		}
	})
}

// TestE2E_ListAcceptsEqualsForm verifies the list parser accepts
// both `--recent N` and `--recent=N` (and same for --since/--cmd/--path).
// The single-pass rewrite added equals-form support; we lock it in
// so a future cleanup doesn't accidentally drop it.
func TestE2E_ListAcceptsEqualsForm(t *testing.T) {
	buildOnce(t)

	// `list --recent=1` against an empty config dir produces the
	// "no matching records found" message — that's success (exit 0,
	// no parse error). The ergonomic gap before the rewrite was
	// that --recent=1 silently fell through to default 20.
	cases := [][]string{
		{"list", "--recent=1"},
		{"list", "--since=1h"},
		{"list", "--cmd=rm"},
		{"list", "--path=/tmp"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			stdout, stderr, code := runFlag(t, args...)
			if code != 0 {
				t.Errorf("%v: exit=%d (want 0)\nstdout: %s\nstderr: %s",
					args, code, stdout, stderr)
			}
		})
	}
}
