package subcmd

import (
	"fmt"
	"os"

	"github.com/hyper0x/cmdguard/internal/config"
)

// RunConfig handles the "config" command
func RunConfig(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[cmdguard] 当前配置:")
	fmt.Printf("  配置文件: %s\n", config.ConfigPath())
	fmt.Println()

	fmt.Println("  全局保护规则:")
	printProtectLevel("reject", cfg.Protect.Reject)
	printProtectLevel("confirm_double", cfg.Protect.ConfirmDouble)
	printProtectLevel("confirm", cfg.Protect.Confirm)
	printProtectLevel("warn", cfg.Protect.Warn)

	if len(cfg.Protect.Command) > 0 {
		fmt.Println()
		fmt.Println("  命令级规则:")
		for cmdName, pc := range cfg.Protect.Command {
			fmt.Printf("    [%s]\n", cmdName)
			printProtectLevel("reject", pc.Reject)
			printProtectLevel("confirm_double", pc.ConfirmDouble)
			printProtectLevel("confirm", pc.Confirm)
			printProtectLevel("warn", pc.Warn)
		}
	}

	fmt.Println()
	fmt.Println("  Vault 设置:")
	fmt.Printf("    retention_days: %d\n", cfg.Vault.RetentionDays)
	fmt.Printf("    auto_purge: %v\n", cfg.Vault.AutoPurge)
}

func printProtectLevel(level string, paths []string) {
	if len(paths) == 0 {
		return
	}
	fmt.Printf("    [%s]\n", level)
	for _, p := range paths {
		fmt.Printf("      - %s\n", p)
	}
}
