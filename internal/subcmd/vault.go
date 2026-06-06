package subcmd

import (
	"fmt"
	"os"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/vault"
)

// RunVault handles the "vault" command
func RunVault(args []string) {
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

	dryRun := false
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
			break
		}
	}

	if len(args) == 0 || args[0] == "clean" {
		if dryRun {
			expired, err := v.ListExpired()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
				os.Exit(1)
			}
			if len(expired) == 0 {
				fmt.Println("[cmdguard] 没有过期的 vault 备份")
				return
			}
			fmt.Printf("[cmdguard] 将删除 %d 个过期备份:\n", len(expired))
			for _, e := range expired {
				fmt.Printf("  - %s\n", e)
			}
		} else {
			purged, err := v.PurgeExpired()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[cmdguard] 错误: %v\n", err)
				os.Exit(1)
			}
			if len(purged) == 0 {
				fmt.Println("[cmdguard] 没有过期的 vault 备份")
				return
			}
			fmt.Printf("[cmdguard] 已清理 %d 个过期备份\n", len(purged))
		}
	}
}
