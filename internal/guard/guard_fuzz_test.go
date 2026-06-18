package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

// FuzzMatchPath stress-tests matchPath with random inputs to:
//
//   1. Confirm it never panics (the function is called inside Check
//      on every cmdguard invocation; a panic here equals a hard
//      crash on a destructive command path).
//
//   2. Reassert the security invariants that motivated the matchPath
//      boundary fix:
//
//        a. Whenever pattern is "<prefix>/**", any path that starts
//           with `<prefix>` but is *not* `<prefix>` itself or under
//           `<prefix>/...` must NOT match. The classic regression is
//           "/etc/**" wrongly matching "/etcd". This is a security
//           invariant — false matches expand the protection set
//           into unrelated directories and cause spurious rejections,
//           but missed boundary checks (the symmetric direction) leak
//           protection. We only assert the false-positive direction
//           because that's the one that previously broke.
//
//        b. Whenever pattern is "**<suffix>" (no slashes in suffix),
//           the suffix must be a true filename suffix, not a substring.
//           "**.key" must not match "file.keystore". Same security
//           reasoning.
//
// IMPORTANT: matchPath internally runs filepath.Clean on both inputs
// before comparing, so the invariants only apply to the *cleaned*
// pattern. Otherwise the fuzzer trivially "finds" cases like "./**"
// (which Clean rewrites to "**", a legitimate match-everything
// pattern) and reports them as boundary violations.
func FuzzMatchPath(f *testing.F) {
	seeds := []struct {
		path    string
		pattern string
	}{
		{"/etc/passwd", "/etc/**"},
		{"/etcd/foo", "/etc/**"},
		{"/etcd-data/snap", "/etc/**"},
		{"/a/b/secret.key", "**.key"},
		{"/a/b/file.keystore", "**.key"},
		{"file.keystore", "**.key"},
		{".key", "**.key"},
		{"/", "/"},
		{"", ""},
		{"a", "*"},
		{"a/b/c", "**"},
		{"a/b/c", "a/**/c"},
		{"~/.ssh/id_rsa", "~/.ssh/**"},
		{"~/.ssh-backup/id_rsa", "~/.ssh/**"},
	}
	for _, s := range seeds {
		f.Add(s.path, s.pattern)
	}

	f.Fuzz(func(t *testing.T, path, pattern string) {
		// 1. No-panic invariant. A defer-recover would mask real bugs;
		//    we want the fuzzer to fail loudly. Just calling matchPath
		//    here is enough — if it panics, the test fails by default.
		got := matchPath(path, pattern)
		if !got {
			return // negative match never violates these invariants
		}

		// matchPath cleans inputs internally; reproduce that here so
		// our predicates run on the same canonical forms.
		cleanedPath := filepath.Clean(path)
		cleanedPattern := filepath.Clean(pattern)

		// 2a. Prefix boundary: `<prefix>/**` patterns.
		//     Skip the bare "**" (matches everything by definition)
		//     and any pattern not ending in `/**`. Skip the trivial
		//     "/**" (prefix is empty — every absolute path is under it).
		if strings.HasSuffix(cleanedPattern, "/**") &&
			cleanedPattern != "/**" {
			prefix := strings.TrimSuffix(cleanedPattern, "/**")
			if cleanedPath != prefix &&
				!strings.HasPrefix(cleanedPath, prefix+"/") {
				t.Errorf("matchPath(%q, %q) = true but cleaned path %q is not under %q (boundary violation)",
					path, pattern, cleanedPath, prefix)
			}
		}

		// 2b. Suffix invariant: `**<literal-suffix>` with no wildcards
		//     in the suffix. The cleaned pattern must still start with
		//     "**" — Clean can sometimes rewrite "**foo" if it contains
		//     separators (e.g. "**/foo"), in which case the suffix
		//     family no longer applies and we skip.
		if strings.HasPrefix(cleanedPattern, "**") {
			suffix := strings.TrimPrefix(cleanedPattern, "**")
			if suffix != "" &&
				!strings.ContainsAny(suffix, "*?") &&
				!strings.Contains(suffix, "/") {
				if !strings.HasSuffix(cleanedPath, suffix) {
					t.Errorf("matchPath(%q, %q) = true but cleaned path %q does not end with %q (suffix violation)",
						path, pattern, cleanedPath, suffix)
				}
			}
		}
	})
}
