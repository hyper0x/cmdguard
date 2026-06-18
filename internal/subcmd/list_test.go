package subcmd

import (
	"testing"
	"time"
)

// TestParseDuration_Valid covers the accepted input shapes.
//
// Two parser paths converge here:
//   - the "Nd" extension (N days), our own pre-processing layer
//   - everything else, delegated straight to time.ParseDuration
//
// We test one example per path plus an edge case (zero) to lock in
// the contract: callers of parseDuration receive nil errors only for
// the formats listed in the user-facing help text.
func TestParseDuration_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"30m", 30 * time.Minute},
		{"2h", 2 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"0d", 0},
		{"  1h  ", 1 * time.Hour}, // surrounding whitespace tolerated
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseDuration(tc.in)
			if err != nil {
				t.Fatalf("parseDuration(%q) returned error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseDuration_Invalid is the regression that motivated the
// API change. Previously parseDuration returned (0, no-error) for
// these, and the list command silently ignored the --since flag,
// showing the unfiltered log. Now they must surface as errors so
// the CLI layer can reject them with a clean message.
func TestParseDuration_Invalid(t *testing.T) {
	cases := []string{
		"",         // empty
		"7days",    // wrong suffix
		"tomorrow", // free-form text
		"d",        // missing number
		"-7d",      // negative days
		"-1h",      // negative time.Duration
		"abc",      // garbage
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := parseDuration(in)
			if err == nil {
				t.Errorf("parseDuration(%q) expected error, got nil", in)
			}
		})
	}
}
