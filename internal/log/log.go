package log

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// Entry represents a single log entry
type Entry struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Command   string `json:"command"`
	Action    string `json:"action"` // reject, confirm, warn, allow, undo, vault-clean
	Targets   string `json:"targets"`
	Rule      string `json:"rule,omitempty"`
	Message   string `json:"message,omitempty"`
	Bypass    string `json:"bypass,omitempty"`
	Expired   bool   `json:"expired,omitempty"`
}

// Log manages operation logs
type Log struct {
	mu       sync.RWMutex
	dir      string
	entries  []Entry
	maxLines int
}

// New creates a new Log instance
func New() (*Log, error) {
	dir := filepath.Join(config.ConfigDir(), "log")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf(msg.ErrLogDir, err)
	}

	l := &Log{
		dir:      dir,
		maxLines: 10000, // keep last 10000 lines in memory
	}

	if err := l.load(); err != nil {
		// Single-pass format: msg.FmtWarn does the Sprintf itself.
		// The previous "FmtWarn(template)+\\n" + Fprintf pattern
		// double-formatted the string and produced %!v(MISSING) noise.
		fmt.Fprintln(os.Stderr, msg.FmtWarn(msg.ErrLogLoad, err))
	}

	return l, nil
}

// logFilePath returns the path to today's log file
func (l *Log) logFilePath() string {
	return filepath.Join(l.dir, time.Now().Format("2006-01-02")+".log")
}

// load reads log entries from today's log file
func (l *Log) load() error {
	path := l.logFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		l.entries = append(l.entries, entry)
	}
	return nil
}

// NewID generates a fresh 12-character hex identifier for a log/vault
// entry. Backed by crypto/rand so two calls within the same nanosecond
// will not collide. Falls back to a UnixNano-derived value only if
// crypto/rand is unavailable (extremely unlikely on supported platforms).
//
// 6 random bytes → 12 hex chars → 2^48 ≈ 2.8e14 distinct values.
// Even at one operation per microsecond, expected collision time is
// in the hundreds of years (birthday paradox).
//
// Centralised here because both this package's Append and the guard
// flow need to mint IDs; previously the formula was duplicated.
func NewID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	// Fallback: keep deterministic-ish output instead of panicking.
	// time.Now().UnixNano() is monotonically increasing on a single
	// process so successive calls still differ.
	return fmt.Sprintf("%012x", time.Now().UnixNano())[:12]
}

// Append adds a new log entry and writes to file
func (l *Log) Append(entry Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = NewID()
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}

	l.entries = append(l.entries, entry)

	// Trim if too many in memory
	if len(l.entries) > l.maxLines {
		l.entries = l.entries[len(l.entries)-l.maxLines:]
	}

	// Append to today's log file
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf(msg.ErrLogSerialize, err)
	}

	f, err := os.OpenFile(l.logFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf(msg.ErrLogWrite, err)
	}
	// Close is intentionally ignored here: f.Sync below is the
	// durability barrier we actually care about, and once Sync
	// returns the bytes are already on disk. A Close error after
	// a successful Sync is informational at best (and on Linux
	// always nil for regular files), so swallowing it keeps the
	// hot path simple. errcheck silenced via `_ =` rather than a
	// nolint directive because the reasoning is type-level, not
	// linter-specific.
	defer func() { _ = f.Close() }()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf(msg.ErrLogLine, err)
	}

	// Force the audit log entry to disk. cmdguard's log is the only
	// trail of which destructive operations were attempted, approved,
	// or bypassed — we cannot afford to lose the most recent entries
	// to a system crash. The cost (one fsync per guarded command) is
	// acceptable given how rarely rm/mv/chmod run compared to other I/O.
	if err := f.Sync(); err != nil {
		return fmt.Errorf(msg.ErrLogLine, err)
	}

	return nil
}

// Query searches log entries with filters
type Query struct {
	Recent int
	Since  time.Duration
	Cmd    string
	Path   string
}

// Search returns matching log entries, newest first
func (l *Log) Search(q Query) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Work on a copy
	entries := make([]Entry, len(l.entries))
	copy(entries, l.entries)

	// Reverse to get newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	var result []Entry
	now := time.Now()

	for _, e := range entries {
		// Filter by command
		if q.Cmd != "" && e.Command != q.Cmd {
			continue
		}

		// Filter by path
		if q.Path != "" && !strings.Contains(e.Targets, q.Path) {
			continue
		}

		// Filter by time
		if q.Since > 0 {
			t, err := time.Parse(time.RFC3339, e.Timestamp)
			if err == nil {
				if now.Sub(t) > q.Since {
					continue
				}
			}
		}

		result = append(result, e)

		// Limit
		if q.Recent > 0 && len(result) >= q.Recent {
			break
		}
	}

	if q.Recent > 0 && len(result) > q.Recent {
		result = result[:q.Recent]
	}

	return result
}

// FindByID finds a single entry by ID (supports prefix matching)
func (l *Log) FindByID(id string) *Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var match *Entry
	for _, e := range l.entries {
		if e.ID == id {
			return &e
		}
		if strings.HasPrefix(e.ID, id) {
			if match != nil {
				// Ambiguous prefix
				return nil
			}
			match = &e
		}
	}
	return match
}

// MarkExpired marks entries whose vault has been purged
func (l *Log) MarkExpired(ids []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	for i := range l.entries {
		if idSet[l.entries[i].ID] {
			l.entries[i].Expired = true
		}
	}

	// Rewrite today's log file
	return l.rewrite()
}

// rewrite rewrites today's log file with current in-memory entries
func (l *Log) rewrite() error {
	path := l.logFilePath()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	// Write path → Close error must be reported. A failed Close on a
	// just-written file usually means the buffered tail did not reach
	// disk (full FS, quota exceeded, transient I/O error). Silently
	// dropping that would leave a truncated audit log claiming success.
	// We return the first error encountered: write/Sync error has
	// priority over close error, but if everything else succeeded we
	// still surface the close error.
	for _, e := range l.entries {
		line, err := json.Marshal(e)
		if err != nil {
			continue
		}
		if _, werr := f.Write(append(line, '\n')); werr != nil {
			_ = f.Close()
			return werr
		}
	}
	if serr := f.Sync(); serr != nil {
		_ = f.Close()
		return serr
	}
	return f.Close()
}
