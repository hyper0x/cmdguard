package log

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
)

func setupTestLog(t *testing.T) (*Log, string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)
	l, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return l, tmp
}

func TestAppend_AssignsIDAndTimestamp(t *testing.T) {
	l, _ := setupTestLog(t)

	e := Entry{Command: "rm", Action: "allow", Targets: "/tmp/x"}
	if err := l.Append(e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got := l.Search(Query{Recent: 1})
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
	if got[0].Timestamp == "" {
		t.Error("expected auto-generated timestamp")
	}
	if _, err := time.Parse(time.RFC3339, got[0].Timestamp); err != nil {
		t.Errorf("timestamp not RFC3339: %v", err)
	}
}

func TestAppend_PreservesBypassField(t *testing.T) {
	l, tmp := setupTestLog(t)

	e := Entry{
		Command: "rm",
		Action:  "allow",
		Targets: "/Users/x/Documents/old.txt",
		Bypass:  "mac-studio/qwenpaw/ai_research/cleanup-cache",
		Rule:    "~/Documents/**",
	}
	if err := l.Append(e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Verify in-memory
	got := l.Search(Query{Recent: 1})
	if got[0].Bypass != e.Bypass {
		t.Errorf("Bypass not preserved in memory: got %q", got[0].Bypass)
	}

	// Verify on disk (JSON Lines format, includes the bypass field)
	logFile := filepath.Join(tmp, "log", time.Now().Format("2006-01-02")+".log")
	raw, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(raw), `"bypass":"mac-studio/qwenpaw/ai_research/cleanup-cache"`) {
		t.Errorf("bypass field missing from disk: %s", raw)
	}
	// Verify each line is valid JSON
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("malformed JSON line: %q: %v", line, err)
		}
	}
}

func TestSearch_FilterByCmd(t *testing.T) {
	l, _ := setupTestLog(t)

	_ = l.Append(Entry{Command: "rm", Action: "allow", Targets: "/tmp/a"})
	_ = l.Append(Entry{Command: "mv", Action: "allow", Targets: "/tmp/b"})
	_ = l.Append(Entry{Command: "rm", Action: "reject", Targets: "/tmp/c"})

	got := l.Search(Query{Cmd: "rm"})
	if len(got) != 2 {
		t.Fatalf("expected 2 rm entries, got %d", len(got))
	}
	for _, e := range got {
		if e.Command != "rm" {
			t.Errorf("non-rm entry leaked: %+v", e)
		}
	}
}

func TestSearch_FilterByPath(t *testing.T) {
	l, _ := setupTestLog(t)

	_ = l.Append(Entry{Command: "rm", Targets: "/Users/x/Documents/a.txt"})
	_ = l.Append(Entry{Command: "rm", Targets: "/tmp/b.txt"})
	_ = l.Append(Entry{Command: "rm", Targets: "/Users/x/Documents/sub/c.txt"})

	got := l.Search(Query{Path: "Documents"})
	if len(got) != 2 {
		t.Fatalf("expected 2 Documents entries, got %d", len(got))
	}
}

func TestSearch_NewestFirst(t *testing.T) {
	l, _ := setupTestLog(t)

	// Pre-seed entries with explicit timestamps to control ordering
	_ = l.Append(Entry{ID: "aaaaaaaaaaaa", Timestamp: "2026-01-01T10:00:00Z", Command: "rm", Targets: "/old"})
	_ = l.Append(Entry{ID: "bbbbbbbbbbbb", Timestamp: "2026-01-02T10:00:00Z", Command: "rm", Targets: "/newer"})
	_ = l.Append(Entry{ID: "cccccccccccc", Timestamp: "2026-01-03T10:00:00Z", Command: "rm", Targets: "/newest"})

	got := l.Search(Query{Recent: 10})
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	// Newest first
	if got[0].ID != "cccccccccccc" || got[2].ID != "aaaaaaaaaaaa" {
		t.Errorf("not newest-first: %s, %s, %s", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestSearch_RecentLimit(t *testing.T) {
	l, _ := setupTestLog(t)
	for i := 0; i < 10; i++ {
		_ = l.Append(Entry{Command: "rm", Targets: "/tmp/x"})
	}
	got := l.Search(Query{Recent: 3})
	if len(got) != 3 {
		t.Errorf("expected 3, got %d", len(got))
	}
}

func TestFindByID(t *testing.T) {
	l, _ := setupTestLog(t)
	_ = l.Append(Entry{ID: "uniqueid0000", Command: "rm", Targets: "/x"})
	_ = l.Append(Entry{ID: "ambigprefa00", Command: "rm", Targets: "/y"})
	_ = l.Append(Entry{ID: "ambigprefb00", Command: "rm", Targets: "/z"})

	// Exact
	if e := l.FindByID("uniqueid0000"); e == nil {
		t.Error("expected exact match")
	}
	// Prefix
	if e := l.FindByID("uniqueid"); e == nil {
		t.Error("expected prefix match")
	}
	// Ambiguous
	if e := l.FindByID("ambigpref"); e != nil {
		t.Errorf("expected nil for ambiguous prefix, got %+v", e)
	}
	// Not found
	if e := l.FindByID("zzzzzz"); e != nil {
		t.Errorf("expected nil for missing id, got %+v", e)
	}
}

func TestMarkExpired_PersistsToDisk(t *testing.T) {
	l, tmp := setupTestLog(t)

	_ = l.Append(Entry{ID: "purgemexxxxx", Command: "rm", Targets: "/old"})
	_ = l.Append(Entry{ID: "keepmexxxxxx", Command: "rm", Targets: "/keep"})

	if err := l.MarkExpired([]string{"purgemexxxxx"}); err != nil {
		t.Fatalf("MarkExpired: %v", err)
	}

	// Re-open from disk → expired flag must persist
	l2, err := New()
	if err != nil {
		t.Fatalf("re-open log: %v", err)
	}
	_ = tmp

	all := l2.Search(Query{Recent: 10})
	var purged, kept *Entry
	for i := range all {
		if all[i].ID == "purgemexxxxx" {
			purged = &all[i]
		}
		if all[i].ID == "keepmexxxxxx" {
			kept = &all[i]
		}
	}
	if purged == nil || !purged.Expired {
		t.Errorf("purged entry must have Expired=true after reload")
	}
	if kept == nil || kept.Expired {
		t.Errorf("kept entry must have Expired=false after reload")
	}
}

func TestLoad_SkipsMalformedLines(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(config.EnvConfigDir, tmp)

	// Manually write a log file with one good line and two malformed
	logDir := filepath.Join(tmp, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	content := `{"id":"validxxx0000","timestamp":"2026-06-07T12:00:00Z","command":"rm","action":"allow","targets":"/tmp/ok"}
this-is-not-json
{"broken": "still missing fields, but valid json"}
`
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	l, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := l.Search(Query{Recent: 10})
	// Should load the valid one and the second JSON (even though incomplete),
	// but skip the plain text line. So 2 entries.
	if len(got) != 2 {
		t.Errorf("expected 2 entries after skipping malformed, got %d", len(got))
	}
}

// TestNewID_NoCollision verifies NewID doesn't repeat in tight loops.
// The previous implementation used the first 12 hex chars of UnixNano,
// which on fast machines could yield identical values for sub-ns calls
// — a problem when those IDs become vault directory names. crypto/rand
// removes that risk entirely.
func TestNewID_NoCollision(t *testing.T) {
	const N = 10000
	seen := make(map[string]struct{}, N)
	for i := 0; i < N; i++ {
		id := NewID()
		if len(id) != 12 {
			t.Fatalf("NewID() = %q, want 12 hex chars", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("collision at iteration %d: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}

// TestNewID_HexFormat checks NewID produces only [0-9a-f].
func TestNewID_HexFormat(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := NewID()
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Fatalf("NewID() = %q contains non-hex char %q", id, c)
			}
		}
	}
}
