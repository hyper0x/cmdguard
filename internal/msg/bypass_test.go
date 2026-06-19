package msg

import (
	"strings"
	"testing"
)

func TestValidateBypass(t *testing.T) {
	tests := []struct {
		id      string
		wantOK  bool
		comment string
	}{
		// Valid
		{"mac-studio/qwenpaw/ai_research/cleanup-tmp-dirs", true, "standard 4-segment"},
		{"laptop-air/claude-code/default/refactor", true, "4-segment with dots and hyphens"},
		{"abc/qwen/coding/cleanup", true, "short but valid (no placeholders, 4 segments)"},

		// Invalid: too short
		{"a/b/c/d", false, "only 7 chars, < 12"},

		// Invalid: wrong segment count
		{"mac/qwen/ai/task/extra", false, "5 segments"},
		{"mac/qwen/ai", false, "3 segments"},
		{"mac/qwen/ai/", false, "trailing slash = 4th empty segment"},
		{"mac/qwen/ai/task/", false, "trailing slash = 5th empty segment"},

		// Invalid: bad characters in segments
		{"mac studio/qwenpaw/ai_research/cleanup", false, "space in first segment"},
		{"mac/中文/agent/task", false, "Chinese characters"},
		{"mac/qwenpaw/ai_research/task!", false, "exclamation mark"},

		// Invalid: empty segment
		{"mac//ai_research/cleanup", false, "empty second segment"},
		{"/qwenpaw/ai_research/task", false, "leading slash = empty first segment"},

		// Invalid: literal template placeholders copied from help
		{"<host>/<platform>/<agent>/<task>", false, "template copied verbatim"},
		{"host/platform/agent/task", false, "all-placeholder words"},
		{"mac-studio/qwenpaw/ai_research/task", false, "trailing 'task' placeholder"},
		{"mac-studio/qwenpaw/agent/cleanup", false, "'agent' placeholder"},
		{"host/qwenpaw/ai_research/cleanup", false, "'host' placeholder"},
		{"mac/platform/ai/cleanup-tmp", false, "'platform' placeholder"},
		{"xxx/yyy/zzz/foo-bar", false, "trivial placeholder values"},
		{"mac-studio/qwenpaw/ai_research/todo", false, "'todo' as task"},
		{"mac-studio/qwenpaw/ai_research/changeme", false, "'changeme'"},
		{"mac-studio/{platform}/ai_research/x", false, "curly braces"},
		// Newly-added dummy tokens: these are extremely common AI-agent
		// fillers that must not pollute the audit trail.
		{"test/test/test/test", false, "all-'test' filler"},
		{"mac-studio/qwenpaw/ai_research/test", false, "trailing 'test' placeholder"},
		{"mac-studio/qwenpaw/ai_research/demo", false, "'demo' as task"},
		{"dummy/qwenpaw/ai_research/cleanup", false, "'dummy' as host"},
		{"mac-studio/fake/ai_research/cleanup", false, "'fake' as platform"},
		{"mac-studio/qwenpaw/temp/cleanup", false, "'temp' as agent"},

		// Valid: real values that *contain* placeholder words but are not equal to them
		{"mac-host/qwen-platform/ai_agent/cleanup-task", true, "words appear inside segments but not equal"},
		{"agent_research/qwenpaw/ai_research/cleanup", true, "'agent_research' is a real ID, not 'agent'"},
		{"test-host/qwenpaw/ai_research/cleanup", true, "'test-host' contains 'test' but isn't equal"},
		{"mac-studio/qwenpaw/ai_research/test-cleanup", true, "'test-cleanup' contains 'test' but isn't equal"},
	}

	for _, tt := range tests {
		got := ValidateBypass(tt.id)
		if got != tt.wantOK {
			t.Errorf("ValidateBypass(%q) = %v, want %v (%s)", tt.id, got, tt.wantOK, tt.comment)
		}
	}
}

// FuzzValidateBypass generates random inputs to ensure ValidateBypass:
//  1. Never panics
//  2. For any "valid" id, the id genuinely satisfies the documented
//     format contract (4 segments of safe chars, no placeholders,
//     no template syntax, length >= 12).
//
// Run with:
//
//	go test -fuzz=FuzzValidateBypass -fuzztime=30s ./internal/msg/
func FuzzValidateBypass(f *testing.F) {
	seeds := []string{
		"mac-studio/qwenpaw/ai_research/cleanup-cache",
		"<host>/<platform>/<agent>/<task>",
		"host/platform/agent/task",
		"",
		"a/b/c/d",
		"a/b/c",
		"a//b/c",
		"agent_research/qwenpaw/ai_research/cleanup",
		"mac/qwen/ai/task/extra",
		"xxx/yyy/zzz/foo",
		"a/b/c/dddddddddddddddddd",
		"a/{b}/c/d",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, id string) {
		// Property 1: never panic.
		ok := ValidateBypass(id)

		if !ok {
			return // rejected; nothing more to check
		}

		// Property 2: anything ValidateBypass approves must satisfy the
		// documented format contract.
		if len(id) < 12 {
			t.Errorf("approved id %q is shorter than 12 chars", id)
		}
		if strings.ContainsAny(id, "<>{}") {
			t.Errorf("approved id %q contains template syntax", id)
		}
		parts := strings.Split(id, "/")
		if len(parts) != 4 {
			t.Errorf("approved id %q has %d segments, want 4", id, len(parts))
		}
		for _, p := range parts {
			if p == "" {
				t.Errorf("approved id %q has empty segment", id)
				return
			}
			if !bypassSegment.MatchString(p) {
				t.Errorf("approved id %q has invalid segment %q", id, p)
			}
		}
		// No segment may equal a placeholder token (case-insensitive).
		lower := strings.ToLower(id)
		for seg := range strings.SplitSeq(lower, "/") {
			for _, tok := range placeholderTokens {
				if seg == tok {
					t.Errorf("approved id %q has placeholder segment %q", id, tok)
				}
			}
		}
	})
}