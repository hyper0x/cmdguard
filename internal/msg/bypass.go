package msg

import (
	"regexp"
	"strings"
)

// bypassSegment matches a single segment of a bypass identifier.
var bypassSegment = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// placeholderTokens are the literal placeholder words used in the
// usage / help text. If an agent copy-pastes the template verbatim
// (e.g. "--bypass=<host>/<platform>/<agent>/<task>") instead of
// filling in real values, we MUST reject it. These tokens should
// never appear in a real identifier.
var placeholderTokens = []string{
	"<host>", "<platform>", "<agent>", "<task>",
	"host", "platform", "agent", "task",
	"your-identifier", "identifier",
	"example", "sample", "placeholder",
	"xxx", "yyy", "zzz", "foo", "bar", "baz",
	"todo", "tbd", "fixme", "changeme",
}

// containsPlaceholder reports whether any segment of id is (after
// lowercasing) equal to a known placeholder token. We check whole
// segments rather than substrings so that a legitimate value like
// "host-laptop" or "agent_research" is not mistakenly rejected.
func containsPlaceholder(id string) bool {
	// Also catch any segment containing angle brackets (template syntax).
	if strings.ContainsAny(id, "<>{}") {
		return true
	}
	lower := strings.ToLower(id)
	for _, seg := range strings.Split(lower, "/") {
		for _, tok := range placeholderTokens {
			if seg == tok {
				return true
			}
		}
	}
	return false
}

// ValidateBypass checks whether a --bypass identifier conforms to the
// required format: <host>/<platform>/<agent>/<task>
//
//   - exactly 4 segments separated by '/'
//   - each segment matches [a-zA-Z0-9._-]+ (no empty segments)
//   - total length >= 12 characters
//   - no segment may be a literal template placeholder such as
//     "host", "platform", "agent", "task", "xxx", "foo", "todo", ...
//   - no angle brackets or template syntax allowed
//
// The placeholder check exists because LLM agents are prone to copy
// the example string from the help text verbatim ("agent" / "<agent>")
// instead of supplying a real identifier. Rejecting those keeps the
// audit log meaningful.
func ValidateBypass(id string) bool {
	if len(id) < 12 {
		return false
	}
	if containsPlaceholder(id) {
		return false
	}
	// Use a manual split so we reject empty segments and extra slashes.
	parts := []string{}
	start := 0
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			parts = append(parts, id[start:i])
			start = i + 1
		}
	}
	parts = append(parts, id[start:])
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if !bypassSegment.MatchString(p) {
			return false
		}
	}
	return true
}
