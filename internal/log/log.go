package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
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
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	l := &Log{
		dir:      dir,
		maxLines: 10000, // keep last 10000 lines in memory
	}

	if err := l.load(); err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 警告: 加载日志失败: %v\n", err)
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

// Append adds a new log entry and writes to file
func (l *Log) Append(entry Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%x", time.Now().UnixNano())[:12]
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
		return fmt.Errorf("序列化日志失败: %w", err)
	}

	f, err := os.OpenFile(l.logFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("写入日志文件失败: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("写入日志行失败: %w", err)
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

// FindByID finds a single entry by ID
func (l *Log) FindByID(id string) *Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, e := range l.entries {
		if e.ID == id {
			return &e
		}
	}
	return nil
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
	defer f.Close()

	for _, e := range l.entries {
		line, err := json.Marshal(e)
		if err != nil {
			continue
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// FormatEntry formats a log entry for display
func FormatEntry(e Entry) string {
	expired := ""
	if e.Expired {
		expired = " [expired]"
	}
	return fmt.Sprintf("%s  %s  %s  %s%s",
		e.ID[:min(len(e.ID), 8)],
		e.Timestamp,
		e.Command,
		e.Targets,
		expired,
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
