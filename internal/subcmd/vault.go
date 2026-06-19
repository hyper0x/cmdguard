package subcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
	"github.com/hyper0x/cmdguard/internal/vault"
)

// RunVault handles the "vault" command.
func RunVault(args []string) {
	// Require an explicit subcommand. Earlier this defaulted to
	// "clean", which meant `cmdguard vault` (no args) silently ran a
	// destructive operation. Even though clean is currently a no-op
	// when nothing is expired, the contract should not bury a delete
	// behind a bare verb. Sweep finding (P2-1).
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, msg.VaultUsage)
		os.Exit(1)
	}
	if strings.HasPrefix(args[0], "-") {
		// e.g. `cmdguard vault --dry-run`. Refuse rather than guess
		// which subcommand the flag belongs to.
		errExit("vault subcommand required (got flag %q)", args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		errExit(msg.ErrConfigLoad, err)
	}

	v, err := vault.New(&cfg.Vault)
	if err != nil {
		errExit(msg.ErrVaultNew, err)
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "clean":
		runVaultClean(v, subArgs)
	case "list":
		runVaultList(v, subArgs)
	default:
		fmt.Fprintln(os.Stderr, msg.VaultUsage)
		errExit("unknown vault subcommand: %s", sub)
	}
}

// runVaultClean handles `vault clean [--dry-run]`.
func runVaultClean(v *vault.Vault, args []string) {
	dryRun := false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		default:
			rejectUnknownArg(a, "vault clean")
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
		switch a {
		case "--json":
			jsonOutput = true
		default:
			rejectUnknownArg(a, "vault list")
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
