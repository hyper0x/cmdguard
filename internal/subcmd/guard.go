package subcmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/guard"
	"github.com/hyper0x/cmdguard/internal/log"
	"github.com/hyper0x/cmdguard/internal/vault"
)

// RunGuard handles rm/mv/chmod commands
func RunGuard(cmdName string, args []string) {
	// Check for special flags
	dryRun := false
	for _, a := range args {
		switch a {
		case "--check":
			fmt.Printf("[cmdguard] 防护已生效 — %s 正在通过 cmdguard 运行\n", cmdName)
			os.Exit(0)
		case "--dry-run":
			dryRun = true
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
		os.Exit(1)
	}

	// Auto-purge expired vault backups if enabled
	if cfg.Vault.AutoPurge {
		v, err := vault.New(&cfg.Vault)
		if err == nil {
			purged, _ := v.PurgeExpired()
			if len(purged) > 0 {
				logger, err := log.New()
				if err == nil {
					logger.MarkExpired(purged)
				}
			}
		}
	}

	// Extract target paths from arguments
	targets := guard.ExtractTargets(cmdName, args)

	if len(targets) == 0 {
		// No file targets, just execute the original command
		execOriginal(cmdName, args)
		return
	}

	// Check protection rules
	result := guard.Check(cfg, cmdName, targets)

	switch result.Action {
	case "reject":
		guard.PrintWarning(cmdName, result)
		logEntry := log.Entry{
			Command: cmdName,
			Action:  "reject",
			Targets: strings.Join(targets, ", "),
			Rule:    result.Rule,
			Message: result.Message,
		}
		if logger, err := log.New(); err == nil {
			logger.Append(logEntry)
		}
		os.Exit(1)

	case "confirm_double":
		guard.PrintWarning(cmdName, result)
		if dryRun {
			fmt.Printf("[cmdguard] 将备份后执行 %s（需要双层确认）\n", cmdName)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "是否继续? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "[cmdguard] 已取消")
			logEntry := log.Entry{
				Command: cmdName,
				Action:  "reject",
				Targets: strings.Join(targets, ", "),
				Rule:    result.Rule,
				Message: "用户取消",
			}
			if logger, err := log.New(); err == nil {
				logger.Append(logEntry)
			}
			os.Exit(1)
		}
		// Second confirmation — require full word "yes"
		fmt.Fprintf(os.Stderr, "再次确认? 请输入 'yes' 确认: ")
		answer2, _ := reader.ReadString('\n')
		answer2 = strings.TrimSpace(strings.ToLower(answer2))
		if answer2 != "yes" {
			fmt.Fprintln(os.Stderr, "[cmdguard] 已取消")
			logEntry := log.Entry{
				Command: cmdName,
				Action:  "reject",
				Targets: strings.Join(targets, ", "),
				Rule:    result.Rule,
				Message: "用户取消（双重确认未通过）",
			}
			if logger, err := log.New(); err == nil {
				logger.Append(logEntry)
			}
			os.Exit(1)
		}
		// Both confirmed, fall through to backup + execute

	case "confirm":
		guard.PrintWarning(cmdName, result)
		if dryRun {
			fmt.Printf("[cmdguard] 将备份后执行 %s（需要确认）\n", cmdName)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "是否继续? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "[cmdguard] 已取消")
			logEntry := log.Entry{
				Command: cmdName,
				Action:  "reject",
				Targets: strings.Join(targets, ", "),
				Rule:    result.Rule,
				Message: "用户取消",
			}
			if logger, err := log.New(); err == nil {
				logger.Append(logEntry)
			}
			os.Exit(1)
		}
		// User confirmed, fall through to backup + execute

	case "warn":
		guard.PrintWarning(cmdName, result)
		if dryRun {
			fmt.Printf("[cmdguard] 将备份后执行 %s\n", cmdName)
			os.Exit(0)
		}
		// Continue to backup + execute

	case "allow":
		if dryRun {
			fmt.Printf("[cmdguard] 无匹配规则 — %s 将直接执行\n", cmdName)
			os.Exit(0)
		}
		// No protection matched, execute directly
		execOriginal(cmdName, args)
		return
	}

	// For confirm_double/confirm/warn: backup to vault then execute
	if result.Action == "confirm_double" || result.Action == "confirm" || result.Action == "warn" {
		if dryRun {
			fmt.Printf("[cmdguard] 将备份以下文件到 vault 后执行 %s:\n", cmdName)
			for _, t := range targets {
				fmt.Printf("  - %s\n", t)
			}
			os.Exit(0)
		}
		v, err := vault.New(&cfg.Vault)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[cmdguard] 错误: 创建 vault 失败: %v\n", err)
			execOriginal(cmdName, args)
			return
		}

		entry := log.Entry{
			Command: cmdName,
			Action:  result.Action,
			Targets: strings.Join(targets, ", "),
			Rule:    result.Rule,
			Message: result.Message,
		}
		// Generate ID before using it for backup directory
		entry.ID = fmt.Sprintf("%x", time.Now().UnixNano())[:12]
		entry.Timestamp = time.Now().Format(time.RFC3339)

		// For mv, store source paths in message for undo reference
		if cmdName == "mv" {
			allTargets := guard.ExtractAllTargets(args)
			if len(allTargets) > 1 {
				sources := allTargets[:len(allTargets)-1]
				entry.Message = fmt.Sprintf("源: %s → 目标: %s", strings.Join(sources, ", "), targets[0])
			}
		}

		// Backup files to vault
		backupDir := v.BackupDir(entry.ID)

		if cmdName == "mv" {
			// For mv, backup the destination file (last target) before overwrite
			dest := targets[len(targets)-1]
			if info, err := os.Stat(dest); err == nil && !info.IsDir() {
				if _, err := v.SaveFile(backupDir, dest); err != nil {
					fmt.Fprintf(os.Stderr, "[cmdguard] 警告: 备份 %s 失败: %v\n", dest, err)
				}
			}
		} else {
			// For rm, chmod: backup the target files
			for _, t := range targets {
				info, err := os.Stat(t)
				if err != nil {
					continue
				}
				if !info.IsDir() {
					if _, err := v.SaveFile(backupDir, t); err != nil {
						fmt.Fprintf(os.Stderr, "[cmdguard] 警告: 备份 %s 失败: %v\n", t, err)
					}
				}
			}
		}

		// Log the operation
		if logger, err := log.New(); err == nil {
			logger.Append(entry)
		}

		// Execute the original command
		execOriginal(cmdName, args)
	}
}

// execOriginal executes the original system command
func execOriginal(cmdName string, args []string) {
	// Find the real command in PATH
	realCmd, err := findRealCommand(cmdName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: 找不到命令 '%s': %v\n", cmdName, err)
		os.Exit(1)
	}

	// Execute using exec.Command
	c := exec.Command(realCmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

// findRealCommand finds the real command, skipping cmdguard itself
func findRealCommand(name string) (string, error) {
	pathEnv := os.Getenv("PATH")
	dirs := strings.Split(pathEnv, ":")

	self, _ := os.Executable()

	for _, dir := range dirs {
		fullPath := filepath.Join(dir, name)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() && (info.Mode().Perm()&0111) != 0 {
			// Skip if it's the same as the running binary
			if self != "" {
				selfInfo, _ := os.Stat(self)
				targetInfo, _ := os.Stat(fullPath)
				if selfInfo != nil && targetInfo != nil && os.SameFile(selfInfo, targetInfo) {
					continue
				}
			}
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("命令 %s 未找到", name)
}
