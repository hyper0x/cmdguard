package subcmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
	"github.com/hyper0x/cmdguard/internal/vault"
)

// RunVault handles the "vault" command
func RunVault(args []string) {
	cfg, err := config.Load()
	if err != nil {
		errExit(msg.ErrConfigLoad, err)
	}

	v, err := vault.New(&cfg.Vault)
	if err != nil {
		errExit(msg.ErrVaultNew, err)
	}

	// Dispatch on subcommand. Each branch parses its own flags so flag
	// scoping is explicit (e.g. --json belongs to `list`, not `clean`)
	// and unknown flags can be reported per-subcommand if needed later.
	sub := "clean"
	subArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		subArgs = args[1:]
	}

	switch sub {
	case "clean":
		runVaultClean(v, subArgs)
	case "list":
		runVaultList(v, subArgs)
	default:
		errExit("unknown vault subcommand: %s", sub)
	}
}

// runVaultClean handles `vault clean [--dry-run]`.
func runVaultClean(v *vault.Vault, args []string) {
	dryRun := false
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	if dryRun {
		expired, err := v.ListExpired()
		if err != nil {
			errExit(msg.ErrListExpired, err)
		}
		if len(expired) == 0 {
			fmt.Println(msg.VaultNoExpired)
			return
		}
		fmt.Printf(msg.VaultWillPurge+"\n", len(expired))
		for _, e := range expired {
			fmt.Printf("  - %s\n", e)
		}
		return
	}

	purged, err := v.PurgeExpired()
	if err != nil {
		errExit(msg.ErrPurgeExpired, err)
	}
	if len(purged) == 0 {
		fmt.Println(msg.VaultNoExpired)
		return
	}
	fmt.Printf(msg.VaultPurged+"\n", len(purged))
}

// runVaultList handles `vault list [--json]`.
func runVaultList(v *vault.Vault, args []string) {
	jsonOutput := false
	for _, a := range args {
		if a == "--json" {
			jsonOutput = true
		}
	}

	backups, err := v.ListAll()
	if err != nil {
		errExit("failed to list vault backups: %v", err)
	}

	if len(backups) == 0 {
		fmt.Println(msg.VaultListNoBackups)
		return
	}

	if jsonOutput {
		type jsonEntry struct {
			ID        string   `json:"id"`
			Timestamp string   `json:"timestamp"`
			Files     []string `json:"files"`
			Expired   bool     `json:"expired"`
		}
		entries := make([]jsonEntry, 0, len(backups))
		for _, b := range backups {
			entries = append(entries, jsonEntry{
				ID:        b.ID,
				Timestamp: b.Timestamp.Format(time.RFC3339),
				Files:     b.Files,
				Expired:   b.Expired,
			})
		}
		data, err := json.Marshal(entries)
		if err != nil {
			errExit("failed to encode vault list: %v", err)
		}
		fmt.Println(string(data))
		return
	}

	fmt.Printf(msg.VaultListTableHeader, "ID", "Time", "Files", "Status")
	fmt.Println(msg.VaultListTableSeparator)
	for _, b := range backups {
		ts := b.Timestamp.Format("2006-01-02 15:04:05")
		shortID := b.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		files := strings.Join(b.Files, ", ")
		if files == "" {
			files = "-"
		}
		status := ""
		if b.Expired {
			status = msg.VaultListStatusExpired
		}
		fmt.Printf(msg.VaultListTableRow, shortID, ts, files, status)
	}
	fmt.Printf(msg.VaultListSummary+"\n", len(backups))
}
