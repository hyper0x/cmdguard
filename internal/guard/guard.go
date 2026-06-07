package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// Result represents the outcome of a guard check
type Result struct {
	Action  string   // "reject", "confirm_double", "confirm", "warn", "allow"
	Rule    string   // the matched rule path
	Targets []string
	Message string
}

// Check evaluates whether the given targets are protected
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

// matchPath checks if a path matches a glob-like pattern
// Supports ** (any depth), * (within a single component)
func matchPath(path, pattern string) bool {
	// Clean paths
	path = filepath.Clean(path)
	pattern = filepath.Clean(pattern)

	// If pattern starts with **, it matches any prefix
	if strings.HasPrefix(pattern, "**") {
		suffix := strings.TrimPrefix(pattern, "**")
		if suffix == "" {
			return true
		}
		return strings.HasSuffix(path, suffix) || strings.Contains(path, suffix)
	}

	// If pattern ends with **, it matches any suffix
	if strings.HasSuffix(pattern, "**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix)
	}

	// For patterns like *.key or file.???, match basename
	if !strings.Contains(pattern, "/") && (strings.Contains(pattern, "*") || strings.Contains(pattern, "?")) {
		base := filepath.Base(path)
		return matchGlob(base, pattern)
	}

	// For full path patterns without wildcards, exact match
	if !strings.Contains(pattern, "*") {
		return path == pattern
	}

	// Fall back to path prefix matching for patterns like /etc/**
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix)
	}

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

// ExtractTargets extracts file/directory paths from command arguments
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

// ExtractAllTargets extracts ALL file/directory paths from command arguments
// (unlike ExtractTargets which for mv only returns the destination)
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

// PrintWarning prints a warning message for protected paths
func PrintWarning(cmd string, result *Result) {
	icon := msg.LevelIcons[msg.LevelReject]
	level := msg.LevelLabels[msg.LevelReject]
	switch result.Action {
	case msg.LevelConfirmDbl:
		icon = msg.LevelIcons[msg.LevelConfirmDbl]
		level = msg.LevelLabels[msg.LevelConfirmDbl]
	case msg.LevelConfirm:
		icon = msg.LevelIcons[msg.LevelConfirm]
		level = msg.LevelLabels[msg.LevelConfirm]
	case msg.LevelWarn:
		icon = msg.LevelIcons[msg.LevelWarn]
		level = msg.LevelLabels[msg.LevelWarn]
	}

	fmt.Fprint(os.Stderr, msg.GuardWarningFmt(cmd, icon, level, result.Rule, result.Message, result.Targets))
}
