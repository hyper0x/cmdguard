package subcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/log"
	"github.com/hyper0x/cmdguard/internal/vault"
)

// RunUndo handles the "undo" command
func RunUndo(args []string) {
	logger, err := log.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
		os.Exit(1)
	}

	// Check if we have piped input or --id
	id := ""
	interactive := true
	dryRun := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				id = args[i+1]
				interactive = false
				i++
			}
		case "--interactive":
			interactive = true
		case "--dry-run":
			dryRun = true
		}
	}

	// Check piped input (from cmdguard list)
	if id == "" && interactive {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Piped input — supports both table output and JSON
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "-") {
					continue
				}
				// Try JSON pipe: [{"id":"...",...},...]
				if strings.HasPrefix(line, "[") {
					// Extract first id from JSON array
					idx := strings.Index(line, `"id":"`)
					if idx >= 0 {
						start := idx + 6
						end := strings.Index(line[start:], `"`)
						if end >= 0 {
							id = line[start : start+end]
							break
						}
					}
				}
				// Table pipe: parse ID from first column
				fields := strings.Fields(line)
				if len(fields) > 0 && fields[0] != "ID" {
					id = fields[0]
					break
				}
			}
		}
	}

	if id != "" {
		// Direct undo by ID
		undoByID(logger, id, dryRun)
		return
	}

	if interactive {
		// Interactive mode: list recent operations and let user choose
		entries := logger.Search(log.Query{Recent: 20})
		if len(entries) == 0 {
			fmt.Println("[cmdguard] 没有可恢复的操作记录")
			return
		}

		fmt.Println("[cmdguard] 选择要恢复的操作:")
		for i, e := range entries {
			if e.Expired {
				continue
			}
			fmt.Printf("  %d. %s  %s  %s\n", i+1, e.Timestamp[:19], e.Command, e.Targets)
		}
		fmt.Print("输入编号 (0 取消): ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		var idx int
		_, err := fmt.Sscanf(answer, "%d", &idx)
		if err != nil || idx <= 0 || idx > len(entries) {
			fmt.Println("[cmdguard] 已取消")
			return
		}

		selected := entries[idx-1]
		if selected.Expired {
			fmt.Fprintf(os.Stderr, "[cmdguard] 该操作的 vault 备份已过期，无法恢复\n")
			return
		}

		undoByID(logger, selected.ID, dryRun)
		return
	}

	fmt.Fprintln(os.Stderr, "[cmdguard] 用法: cmdguard undo --id <ID>")
	fmt.Fprintln(os.Stderr, "        cmdguard list | cmdguard undo")
	fmt.Fprintln(os.Stderr, "        cmdguard undo (交互式)")
	os.Exit(1)
}

// undoByID restores a single operation by its log ID
func undoByID(logger *log.Log, id string, dryRun bool) {
	entry := logger.FindByID(id)
	if entry == nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 未找到 ID 为 '%s' 的操作记录\n", id)
		os.Exit(1)
	}

	if entry.Expired {
		fmt.Fprintf(os.Stderr, "[cmdguard] 该操作的 vault 备份已过期，无法恢复\n")
		os.Exit(1)
	}

	if entry.Action == "reject" {
		fmt.Fprintf(os.Stderr, "[cmdguard] 该操作已被拒绝，没有可恢复的内容\n")
		os.Exit(1)
	}

	// Initialize vault
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
		os.Exit(1)
	}
	v, err := vault.New(&cfg.Vault)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
		os.Exit(1)
	}

	backupDir := v.FindBackupDir(id)
	if backupDir == "" {
		fmt.Fprintf(os.Stderr, "[cmdguard] 未找到 ID 为 '%s' 的 vault 备份\n", id)
		os.Exit(1)
	}

	// Check if dry-run
	if dryRun {
		fmt.Printf("[cmdguard] 将恢复以下文件:\n")
		targets := strings.Split(entry.Targets, ", ")
		for _, t := range targets {
			fmt.Printf("  - %s\n", t)
		}
		return
	}

	// Restore files
	targets := strings.Split(entry.Targets, ", ")
	restored := 0
	for _, t := range targets {
		if err := v.RestoreFile(backupDir, t); err != nil {
			fmt.Fprintf(os.Stderr, "[cmdguard] 警告: 恢复 %s 失败: %v\n", t, err)
			continue
		}
		restored++
	}

	if restored > 0 {
		fmt.Printf("[cmdguard] 已恢复 %d 个文件\n", restored)

		// Log the undo operation
		undoEntry := log.Entry{
			Command: "undo",
			Action:  "undo",
			Targets: entry.Targets,
			Message: fmt.Sprintf("恢复操作 %s (%s %s)", entry.ID, entry.Command, entry.Targets),
		}
		logger.Append(undoEntry)
	} else {
		fmt.Fprintln(os.Stderr, "[cmdguard] 没有文件被恢复")
		os.Exit(1)
	}
}
