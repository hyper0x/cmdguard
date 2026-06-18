package subcmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsCmdguardWrapper exercises the wrapper-detection logic that
// guards against the recursion observed during the v0.7.0 release
// sweep: a cmdguard wrapper from a *different* install (not the one
// matching the active CMDGUARD_CONFIG_DIR) ended up on PATH and was
// being exec'd as the "real" command.
//
// Scenarios covered:
//   - Real wrapper (sentinel present)         → recognised
//   - Future-version wrapper (v2 prefix)      → still recognised
//   - Plain shell script                      → not recognised
//   - Binary file (no shebang)                → not recognised
//   - Non-existent path                       → not recognised
//   - File too short to contain sentinel      → not recognised
//
// We deliberately do NOT test the full findRealCommand wiring here:
// that requires manipulating PATH and would couple the unit test to
// the test runner's environment. Wrapper detection is the load-bearing
// piece — guard the boundary, not the caller.
func TestIsCmdguardWrapper(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "real_wrapper_v1",
			content: `#!/bin/bash
` + WrapperSentinel + `
exec /usr/local/bin/cmdguard rm "$@"
`,
			want: true,
		},
		{
			name: "future_wrapper_v2",
			content: `#!/bin/bash
# cmdguard:wrapper:v2
exec /usr/local/bin/cmdguard rm "$@"
`,
			want: true,
		},
		{
			name: "plain_shell_script",
			content: `#!/bin/bash
echo hello
`,
			want: false,
		},
		{
			name:    "binary_no_shebang",
			content: "\x7fELF\x02\x01\x01\x00",
			want:    false,
		},
		{
			name:    "empty_file",
			content: "",
			want:    false,
		},
		{
			name:    "shebang_only",
			content: "#!/bin/bash\n",
			want:    false,
		},
		{
			name: "wrapper_invoking_other_tool_no_sentinel",
			// A user-written script that happens to invoke cmdguard
			// without the sentinel — must NOT be skipped, otherwise
			// findRealCommand silently drops a legit command.
			content: `#!/bin/bash
exec /usr/local/bin/cmdguard rm "$@"
`,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			if err := os.WriteFile(path, []byte(tc.content), 0755); err != nil {
				t.Fatalf("write: %v", err)
			}
			got := isCmdguardWrapper(path)
			if got != tc.want {
				t.Errorf("isCmdguardWrapper(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}

	t.Run("nonexistent_path", func(t *testing.T) {
		if isCmdguardWrapper(filepath.Join(dir, "does-not-exist")) {
			t.Errorf("nonexistent path should not be classified as wrapper")
		}
	})
}
