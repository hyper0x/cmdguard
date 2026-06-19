package subcmd

import (
	"fmt"
	"os"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// RunConfig handles the "config" command
func RunConfig(args []string) {
	// Parse flags. Each branch is explicit; unknown flags are rejected
	// with errExit so typos like `--recnet` don't silently fall through
	// to the default `effective` view. Sweep finding (P2-2).
	showDefault := false
	showRaw := false
	showBinDir := false
	for _, a := range args {
		switch a {
		case "--default":
			showDefault = true
		case "--raw":
			showRaw = true
		case "--bin-dir":
			showBinDir = true
		default:
			rejectUnknownArg(a, "config")
		}
	}

	// --bin-dir: print the wrapper script directory and exit.
	//
	// Designed for shell composition. The integration guide tells
	// users to run:
	//
	//   export PATH="$(cmdguard config --bin-dir):$PATH"
	//
	// so the output is intentionally a bare path with no decoration,
	// no trailing newline beyond what fmt.Println adds, no logging
	// tag — anything else would corrupt the resulting PATH.
	//
	// Mutually exclusive with --default and --raw: those produce
	// human-oriented multi-line output that's incompatible with
	// command substitution. We pick --bin-dir over them when more
	// than one is supplied; documenting "first flag wins" would be
	// surprising in either direction, but --bin-dir is the
	// machine-readable mode so it takes precedence.
	if showBinDir {
		fmt.Println(config.BinDir())
		return
	}

	// --raw: dump raw config.toml content
	if showRaw {
		cfgPath := config.ConfigPath()
		// #nosec G304 -- cfgPath is the cmdguard config.toml
		// path, computed from CMDGUARD_CONFIG_DIR or ~/.cmdguard.
		// Not user-supplied argv.
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf(msg.ConfigRawNotExist+"\n", cfgPath)
			} else {
				errExit("failed to read config file: %v", err)
			}
			return
		}
		fmt.Printf(msg.ConfigRawHeader+"\n", cfgPath)
		fmt.Println(string(data))
		return
	}

	// --default: show built-in defaults
	if showDefault {
		cfg := config.DefaultConfig()
		fmt.Println(msg.ConfigDefaultHeader)
		printConfig(cfg)
		return
	}

	// Default: show effective (merged) config
	cfg, err := config.Load()
	if err != nil {
		errExit(msg.ErrConfigLoad, err)
	}

	fmt.Println(msg.ConfigEffectiveHeader)
	printConfig(cfg)
}

// printConfig renders a Config struct to stdout
func printConfig(cfg *config.Config) {
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
