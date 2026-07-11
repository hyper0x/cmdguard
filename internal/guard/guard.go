package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// Result represents the outcome of a guard check.
type Result struct {
	Action  string // "reject", "guarded", "allow"
	Rule    string // the matched rule path
	Targets []string
	Message string
}

// Check evaluates whether the given targets are protected.
func Check(cfg *config.Config, cmd string, targets []string) *Result {
	rules := cfg.GetProtectRules(cmd)

	for _, target := range targets {
		expanded := config.ExpandHome(target)
		abs, err := filepath.Abs(expanded)
		if err != nil {
			abs = expanded
		}

		for _, rule := range rules {
			if matchPath(abs, rule.Path) {
				return &Result{
					Action:  string(rule.Level),
					Rule:    rule.Path,
					Targets: []string{target},
					Message: fmt.Sprintf("path matches protection rule '%s'", rule.Path),
				}
			}
		}
	}

	return &Result{
		Action:  msg.LevelAllow,
		Targets: targets,
	}
}

// matchPath checks if a path matches a glob-like pattern.
// Supports ** (any depth), * (within a single component).
func matchPath(path, pattern string) bool {
	// Clean paths
	path = filepath.Clean(path)
	pattern = filepath.Clean(pattern)

	// Pattern starts with `**` — matches if `path` ends with the suffix.
	//
	// We deliberately do NOT use strings.Contains here: `**` is meant to
	// span path components, not arbitrary substrings. Using Contains would
	// make `**.key` match `file.keystore` (because ".key" is a substring
	// of ".keystore"), which is a security false-positive — protected
	// files would block unrelated paths.
	if suffix, ok := strings.CutPrefix(pattern, "**"); ok {
		if suffix == "" {
			return true
		}
		return strings.HasSuffix(path, suffix)
	}

	// Pattern ends with `/**` — matches `prefix` itself or anything strictly
	// inside `prefix/`. The trailing `/` boundary is REQUIRED to prevent
	// `/etc/**` from matching `/etcd/foo` (different directory whose name
	// merely starts with the prefix).
	if prefix, ok := strings.CutSuffix(pattern, "/**"); ok {
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}

	// Basename glob: pattern has wildcards but no `/` (e.g. *.key, file.???).
	if !strings.Contains(pattern, "/") && (strings.Contains(pattern, "*") || strings.Contains(pattern, "?")) {
		base := filepath.Base(path)
		return matchGlob(base, pattern)
	}

	// Full path without wildcards — exact match.
	if !strings.Contains(pattern, "*") {
		return path == pattern
	}

	// Full-path glob fallback (e.g. /tmp/file.*).
	return matchGlob(path, pattern)
}

// matchGlob is a simple glob matcher supporting * and ?
func matchGlob(path, pattern string) bool {
	px := 0
	py := 0
	nextPx := 0
	nextPy := 0

	for py < len(pattern) || px < len(path) {
		if py < len(pattern) && (pattern[py] == '*' || pattern[py] == '?') {
			nextPx = px
			nextPy = py + 1
			if pattern[py] == '?' {
				if px < len(path) {
					px++
					py++
					continue
				}
				return false
			}
			// '*' matches anything
			py++
			continue
		}
		if py < len(pattern) && px < len(path) && pattern[py] == path[px] {
			px++
			py++
			continue
		}
		if nextPy > 0 && nextPy <= len(pattern) && px < len(path) {
			px = nextPx + 1
			py = nextPy
			nextPx = px
			continue
		}
		return false
	}
	return true
}

// ExtractTargets extracts file/directory paths from command arguments.
func ExtractTargets(cmd string, args []string) []string {
	var targets []string
	skippedMode := false

	for _, arg := range args {
		// Skip flags (anything starting with -)
		if strings.HasPrefix(arg, "-") {
			continue
		}

		// For chmod, skip the mode argument (first non-flag)
		if cmd == "chmod" && !skippedMode {
			skippedMode = true
			continue
		}

		targets = append(targets, arg)
	}

	// For mv, only the destination (last arg) is protected
	if cmd == "mv" && len(targets) > 1 {
		targets = targets[len(targets)-1:]
	}

	return targets
}

// ExtractAllTargets extracts ALL file/directory paths from command arguments.
// (Unlike ExtractTargets, for mv it returns both source and destination.)
func ExtractAllTargets(args []string) []string {
	var targets []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		targets = append(targets, arg)
	}
	return targets
}

// PrintWarning prints a warning message for protected paths.
func PrintWarning(cmd string, result *Result) {
	icon := msg.LevelIcons[msg.LevelReject]
	level := msg.LevelLabels[msg.LevelReject]

	switch result.Action {
	case msg.LevelGuarded:
		icon = msg.LevelIcons[msg.LevelGuarded]
		level = msg.LevelLabels[msg.LevelGuarded]
	case msg.LevelAllow:
		icon = msg.LevelIcons[msg.LevelAllow]
		level = msg.LevelLabels[msg.LevelAllow]
	}

	fmt.Fprint(os.Stderr, msg.GuardWarningFmt(cmd, icon, level, result.Rule, result.Message, result.Targets))
}
