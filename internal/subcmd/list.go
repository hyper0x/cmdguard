package subcmd

import (
	"encoding/json"
	"fmt"
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
		errExit(msg.ErrLogLoad, err)
	}

	q := log.Query{
		Recent: 20, // default
	}
	jsonOutput := false

	// Single-pass arg parser. Every branch consumes its own value
	// (advancing i when needed); the default arm rejects anything we
	// don't recognise. Earlier this loop silently dropped unknown
	// flags, which let typos like `--recnet 5` fall through to the
	// default Recent=20. Sweep finding (P2-2).
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--recent":
			if i+1 >= len(args) {
				errExit(msg.ErrFlagMissingValue, a)
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				errExit(msg.ErrInvalidRecent, args[i+1])
			}
			q.Recent = n
			i++
		case strings.HasPrefix(a, "--recent="):
			val := strings.TrimPrefix(a, "--recent=")
			n, err := strconv.Atoi(val)
			if err != nil || n <= 0 {
				errExit(msg.ErrInvalidRecent, val)
			}
			q.Recent = n
		case a == "--since":
			if i+1 >= len(args) {
				errExit(msg.ErrFlagMissingValue, a)
			}
			d, err := parseDuration(args[i+1])
			if err != nil {
				errExit(msg.ErrListSinceInvalid, args[i+1])
			}
			q.Since = d
			i++
		case strings.HasPrefix(a, "--since="):
			val := strings.TrimPrefix(a, "--since=")
			d, err := parseDuration(val)
			if err != nil {
				errExit(msg.ErrListSinceInvalid, val)
			}
			q.Since = d
		case a == "--cmd":
			if i+1 >= len(args) {
				errExit(msg.ErrFlagMissingValue, a)
			}
			q.Cmd = args[i+1]
			i++
		case strings.HasPrefix(a, "--cmd="):
			q.Cmd = strings.TrimPrefix(a, "--cmd=")
		case a == "--path":
			if i+1 >= len(args) {
				errExit(msg.ErrFlagMissingValue, a)
			}
			q.Path = args[i+1]
			i++
		case strings.HasPrefix(a, "--path="):
			q.Path = strings.TrimPrefix(a, "--path=")
		case a == "--json":
			jsonOutput = true
		default:
			errExit(msg.ErrUnknownFlag, a, "list")
		}
	}

	entries := logger.Search(q)

	if len(entries) == 0 {
		fmt.Println(msg.ListNoResults)
		return
	}

	if jsonOutput {
		// Use encoding/json so the output:
		//  - uses the Entry struct's json tags (id, timestamp, command, ...)
		//  - includes every field, notably Bypass (audit trail)
		//  - escapes targets/messages correctly
		data, err := json.Marshal(entries)
		if err != nil {
			errExit("failed to encode log entries: %v", err)
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
