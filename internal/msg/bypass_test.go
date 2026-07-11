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
		// Valid (3 segments)
		{"qwenpaw/ai_research/cleanup-tmp-dirs", true, "standard 3-segment"},
		{"claude-code/default/refactor-tests", true, "3-segment with dots and hyphens"},
		{"manual/haolin/remove-old-logs", true, "human caller"},
		{"qwen/coding/cleanup", true, "short but valid (3 segments, >= 10 chars)"},

		// Invalid: too short
		{"a/b/c", false, "only 5 chars, < 10"},

		// Invalid: wrong segment count
		{"mac/qwen/ai/task/extra", false, "5 segments"},
		{"mac-studio/qwenpaw/ai_research/cleanup-tmp-dirs", false, "4 segments (old format)"},
		{"mac/qwen/ai/", false, "trailing slash = 4th empty segment"},

		// Invalid: bad characters in segments
		{"mac studio/qwenpaw/cleanup", false, "space in first segment"},
		{"qwenpaw/中文/cleanup", false, "Chinese characters"},
		{"qwenpaw/ai_research/task!", false, "exclamation mark"},

		// Invalid: empty segment
		{"qwenpaw//cleanup", false, "empty second segment"},
		{"/qwenpaw/cleanup", false, "leading slash = empty first segment"},

		// Invalid: literal template placeholders copied from help
		{"<platform>/<agent>/<task>", false, "template copied verbatim"},
		{"platform/agent/task", false, "all-placeholder words"},
		{"qwenpaw/ai_research/task", false, "trailing 'task' placeholder"},
		{"qwenpaw/agent/cleanup", false, "'agent' placeholder"},
		{"platform/ai/cleanup-tmp", false, "'platform' placeholder"},
		{"xxx/yyy/zzz", false, "trivial placeholder values"},
		{"qwenpaw/ai_research/todo", false, "'todo' as task"},
		{"qwenpaw/ai_research/changeme", false, "'changeme'"},
		{"qwenpaw/{platform}/x", false, "curly braces"},
		// Newly-added dummy tokens: these are extremely common AI-agent
		// fillers that must not pollute the audit trail.
		{"test/test/test", false, "all-'test' filler"},
		{"qwenpaw/ai_research/test", false, "trailing 'test' placeholder"},
		{"qwenpaw/ai_research/demo", false, "'demo' as task"},
		{"dummy/qwenpaw/cleanup", false, "'dummy' as platform"},
		{"qwenpaw/fake/cleanup", false, "'fake' as agent"},
		{"qwenpaw/temp/cleanup", false, "'temp' as agent"},

		// Valid: real values that *contain* placeholder words but are not equal to them
		{"qwen-platform/ai_agent/cleanup-task", true, "words appear inside segments but not equal"},
		{"qwenpaw/ai_research/cleanup", true, "'ai_research' is a real ID, not 'agent'"},
		{"qwenpaw/test-host/cleanup", true, "'test-host' contains 'test' but isn't equal"},
		{"qwenpaw/ai_research/test-cleanup", true, "'test-cleanup' contains 'test' but isn't equal"},
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
//     format contract (3 segments of safe chars, no placeholders,
//     no template syntax, length >= 10).
//
// Run with:
//
//	go test -fuzz=FuzzValidateBypass -fuzztime=30s ./internal/msg/
func FuzzValidateBypass(f *testing.F) {
	seeds := []string{
		"qwenpaw/ai_research/cleanup-cache",
		"<platform>/<agent>/<task>",
		"platform/agent/task",
		"",
		"a/b/c",
		"a//b",
		"ai_research/qwenpaw/cleanup",
		"mac/qwen/ai/task/extra",
		"xxx/yyy/zzz",
		"a/b/ccccc",
		"a/{b}/c",
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
		if len(id) < 10 {
			t.Errorf("approved id %q is shorter than 10 chars", id)
		}
		if strings.ContainsAny(id, "<>{}") {
			t.Errorf("approved id %q contains template syntax", id)
		}
		parts := strings.Split(id, "/")
		if len(parts) != 3 {
			t.Errorf("approved id %q has %d segments, want 3", id, len(parts))
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
