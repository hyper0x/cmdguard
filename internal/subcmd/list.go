package subcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hyper0x/cmdguard/internal/log"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// RunList handles the "list" command
func RunList(args []string) {
	logger, err := log.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, msg.FmtErr(msg.ErrLogLoad)+"\n", err)
		os.Exit(1)
	}

	q := log.Query{
		Recent: 20, // default
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--recent":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err == nil && n > 0 {
					q.Recent = n
				}
				i++
			}
		case "--since":
			if i+1 < len(args) {
				d, err := parseDuration(args[i+1])
				if err != nil {
					// Reject malformed input loudly. Previously a typo like
					// `--since 7days` silently parsed to zero, which the
					// log query treats as "no time filter" — so the user
					// got the full unfiltered log without any indication
					// that their flag was ignored.
					fmt.Fprintf(os.Stderr, msg.FmtErr(msg.ErrListSinceInvalid)+"\n", args[i+1])
					os.Exit(1)
				}
				q.Since = d
				i++
			}
		case "--cmd":
			if i+1 < len(args) {
				q.Cmd = args[i+1]
				i++
			}
		case "--path":
			if i+1 < len(args) {
				q.Path = args[i+1]
				i++
			}
		case "--json":
			// handled below
		}
	}

	entries := logger.Search(q)

	if len(entries) == 0 {
		fmt.Println(msg.ListNoResults)
		return
	}

	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			break
		}
	}

	if jsonOutput {
		// Use encoding/json so the output:
		//  - uses the Entry struct's json tags (id, timestamp, command, ...)
		//  - includes every field, notably Bypass (audit trail)
		//  - escapes targets/messages correctly
		data, err := json.Marshal(entries)
		if err != nil {
			fmt.Fprintf(os.Stderr, msg.FmtErr("failed to encode log entries: %v")+"\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf(msg.ListTableHeader, "ID", "Time", "Cmd", "Action", "Target")
		fmt.Println(msg.ListTableSeparator)
		for _, e := range entries {
			expired := ""
			if e.Expired {
				expired = msg.ListExpiredTag
			}
			ts := e.Timestamp
			if len(ts) > 19 {
				ts = ts[:19]
			}
			shortID := e.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			bypassTag := ""
			if e.Bypass != "" {
				bypassTag = fmt.Sprintf(msg.ListBypassTag, e.Bypass)
			}
			fmt.Printf("%-8s  %-19s  %-6s  %-8s  %s%s%s\n",
				shortID, ts, e.Command, e.Action, e.Targets, bypassTag, expired)
		}
	}
}

// parseDuration parses a human-readable duration like "2h", "7d", "30m".
// Returns an error for empty input or unparseable suffixes so callers
// can surface a clean error to the user instead of silently treating
// "7days" as "no filter".
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %w", err)
		}
		if days < 0 {
			return 0, fmt.Errorf("negative duration: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, fmt.Errorf("negative duration: %s", s)
	}
	return d, nil
}
