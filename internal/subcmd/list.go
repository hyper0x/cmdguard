package subcmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hyper0x/cmdguard/internal/log"
)

// RunList handles the "list" command
func RunList(args []string) {
	logger, err := log.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
		os.Exit(1)
	}

	q := log.Query{
		Recent: 20, // default
	}

	// Parse arguments
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
				q.Since = parseDuration(args[i+1])
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
		fmt.Println("[cmdguard] 没有找到匹配的操作记录")
		return
	}

	// Check if --json flag is set
	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			break
		}
	}

	if jsonOutput {
		// JSON output — full ID for pipe to undo
		fmt.Print("[")
		for i, e := range entries {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`{"id":"%s","time":"%s","cmd":"%s","action":"%s","targets":"%s"}`,
				e.ID, e.Timestamp, e.Command, e.Action, e.Targets)
		}
		fmt.Println("]")
	} else {
		// Table output — truncated ID for readability
		fmt.Printf("%-8s  %-19s  %-6s  %-8s  %s\n", "ID", "时间", "命令", "动作", "目标路径")
		fmt.Println(strings.Repeat("-", 80))
		for _, e := range entries {
			expired := ""
			if e.Expired {
				expired = " [expired]"
			}
			ts := e.Timestamp
			if len(ts) > 19 {
				ts = ts[:19]
			}
			shortID := e.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			fmt.Printf("%-8s  %-19s  %-6s  %-8s  %s%s\n",
				shortID, ts, e.Command, e.Action, e.Targets, expired)
		}
	}
}

// parseDuration parses a human-readable duration like "2h", "7d", "30m"
func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)

	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
