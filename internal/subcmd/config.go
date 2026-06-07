package subcmd

import (
	"fmt"
	"os"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// RunConfig handles the "config" command
func RunConfig(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, msg.FmtErr(msg.ErrConfigLoad)+"\n", err)
		os.Exit(1)
	}

	fmt.Println(msg.ConfigHeader)
	fmt.Printf(msg.ConfigFile+"\n", config.ConfigPath())
	fmt.Println()

	fmt.Println(msg.ConfigGlobalRules)
	printProtectLevel(msg.LevelReject, cfg.Protect.Reject)
	printProtectLevel(msg.LevelConfirmDbl, cfg.Protect.ConfirmDouble)
	printProtectLevel(msg.LevelConfirm, cfg.Protect.Confirm)
	printProtectLevel(msg.LevelWarn, cfg.Protect.Warn)

	if len(cfg.Protect.Command) > 0 {
		fmt.Println()
		fmt.Println(msg.ConfigCommandRules)
		for cmdName, pc := range cfg.Protect.Command {
			fmt.Printf("    [%s]\n", cmdName)
			printProtectLevel(msg.LevelReject, pc.Reject)
			printProtectLevel(msg.LevelConfirmDbl, pc.ConfirmDouble)
			printProtectLevel(msg.LevelConfirm, pc.Confirm)
			printProtectLevel(msg.LevelWarn, pc.Warn)
		}
	}

	fmt.Println()
	fmt.Println(msg.ConfigVaultSettings)
	fmt.Printf(msg.ConfigRetentionDays+"\n", cfg.Vault.RetentionDays)
	fmt.Printf(msg.ConfigAutoPurge+"\n", cfg.Vault.AutoPurge)

	fmt.Println()
	fmt.Println(msg.ConfigGuardSettings)
	fmt.Printf(msg.ConfigConfirmTimeout+"\n", cfg.Guard.ConfirmTimeout)
	fmt.Printf(msg.ConfigConfirmDoubleTimeout+"\n", cfg.Guard.ConfirmDoubleTimeout)
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
