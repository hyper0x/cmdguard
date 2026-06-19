package subcmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyper0x/cmdguard/internal/log"
	"github.com/hyper0x/cmdguard/internal/msg"
	"github.com/hyper0x/cmdguard/internal/vault"
)

// RunUndo handles the "undo" command
func RunUndo(args []string) {
	dryRun := false
	interactive := false
	var id string

	// Strict flag parser: every branch consumes exactly what it
	// expects, missing values error out, and unknown tokens hit
	// rejectUnknownArg. The earlier loop dropped unknown flags
	// silently (P2-2 was fixed for list/config/init/path/vault but
	// missed undo) — `cmdguard undo --it` would silently behave like
	// `cmdguard undo` and either pipe-read or print usage, hiding
	// the typo of `--interactive`.
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--id":
			if i+1 >= len(args) {
				errExit(msg.ErrFlagMissingValue, a)
			}
			id = args[i+1]
			i++
		case strings.HasPrefix(a, "--id="):
			id = strings.TrimPrefix(a, "--id=")
			if id == "" {
				errExit(msg.ErrFlagMissingValue, "--id")
			}
		case a == "--interactive":
			interactive = true
		case a == "--dry-run":
			dryRun = true
		default:
			rejectUnknownArg(a, "undo")
		}
	}

	// If no --id provided, try to read from pipe
	if id == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "ID") || strings.HasPrefix(line, "--------") {
					continue
				}
				if strings.HasPrefix(line, `[{"id":`) || strings.HasPrefix(line, `{"id":`) {
					// Try as a single object or first object in an array
					trimmed := strings.TrimPrefix(line, "[")
					end := strings.Index(trimmed, "}")
					if end > 0 {
						obj := trimmed[:end+1]
						var entry struct {
							ID string `json:"id"`
						}
						if err := json.Unmarshal([]byte(obj), &entry); err == nil && entry.ID != "" {
							id = entry.ID
							break
						}
					}
				}
				fields := strings.Fields(line)
				if len(fields) > 0 {
					id = fields[0]
					break
				}
			}
		}
	}

	if id == "" && !interactive {
		// Usage block on a non-zero exit: route to stderr so callers
		// piping stdout (e.g. `undo | something`) don't accidentally
		// inhale the usage text. Sweep finding (P3-2).
		fmt.Fprintln(os.Stderr, msg.UndoUsage)
		fmt.Fprintln(os.Stderr, msg.UndoUsagePipe)
		fmt.Fprintln(os.Stderr, msg.UndoUsageInteractive)
		os.Exit(1)
	}

	logger, err := log.New()
	if err != nil {
		errExit(msg.ErrLogLoad, err)
	}

	// Interactive mode: list recent operations and let user choose
	if id == "" && interactive {
		entries := logger.Search(log.Query{Recent: 20})
		if len(entries) == 0 {
			fmt.Println(msg.UndoNoRecords)
			return
		}

		fmt.Println(msg.UndoSelectPrompt)
		displayIdx := 0
		var displayed []log.Entry
		for _, e := range entries {
			if e.Action == msg.LevelReject || e.Action == msg.LevelUndo {
				continue
			}
			displayIdx++
			displayed = append(displayed, e)
			ts := e.Timestamp
			if len(ts) > 19 {
				ts = ts[:19]
			}
			fmt.Printf(msg.UndoSelectItem+"\n", displayIdx, ts, e.Command, e.Targets)
		}

		if len(displayed) == 0 {
			fmt.Println(msg.UndoNoRecords)
			return
		}

		fmt.Fprint(os.Stderr, msg.UndoSelectInput)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "0" || input == "" {
			fmt.Println(msg.UndoCancelled)
			return
		}

		for i, e := range displayed {
			if fmt.Sprintf("%d", i+1) == input {
				id = e.ID
				break
			}
		}

		if id == "" {
			fmt.Println(msg.UndoCancelled)
			return
		}
	}

	// Find the log entry
	entry := logger.FindByID(id)
	if entry == nil {
		// Failure path → stderr + exit 1. Earlier this went to stdout,
		// which broke `cmd 2>/dev/null` pipelines that try to silence
		// errors. Sweep finding (P3-1). Same below for Expired,
		// Rejected, BackupNotFound.
		fmt.Fprintf(os.Stderr, msg.UndoIDNotFound+"\n", id)
		os.Exit(1)
	}

	if entry.Expired {
		fmt.Fprintln(os.Stderr, msg.UndoExpired)
		os.Exit(1)
	}

	if entry.Action == msg.LevelReject {
		fmt.Fprintln(os.Stderr, msg.UndoRejected)
		os.Exit(1)
	}

	// Find vault backup
	v, err := vault.New(nil)
	if err != nil {
		errExit(msg.ErrVaultNew, err)
	}

	backupDir := v.FindBackupDir(entry.ID)
	if backupDir == "" {
		fmt.Fprintf(os.Stderr, msg.UndoBackupNotFound+"\n", entry.ID)
		os.Exit(1)
	}

	// Determine which paths to restore.
	//
	// Preferred: read the backup's manifest, which records each saved
	// file's *original absolute path*. This avoids the basename-based
	// matching that previously caused two files with the same name
	// (e.g. /etc/foo.conf vs /opt/foo.conf) to be confused at restore.
	//
	// Fallback: legacy backups (created before manifest support) only
	// have files/<basename>. We list them and try to match by basename
	// against the entry.Targets list — same limitation as before, but
	// preserved for backward compatibility.
	manifestEntries, hasManifest := readBackupManifest(backupDir)

	if !hasManifest {
		// Legacy fallback: list backed-up files.
		filesDir := filepath.Join(backupDir, "files")
		if _, err := os.ReadDir(filesDir); err != nil {
			fmt.Fprintf(os.Stderr, msg.UndoBackupNotFound+"\n", entry.ID)
			os.Exit(1)
		}
	}

	// entry.Targets is "/path/a, /path/b, ..." — split for display only.
	logTargets := strings.Split(entry.Targets, ", ")

	if dryRun {
		fmt.Println(msg.UndoDryRunHeader)
		if hasManifest {
			for _, p := range manifestEntries {
				fmt.Printf("  - %s\n", p)
			}
		} else {
			for _, t := range logTargets {
				fmt.Printf("  - %s\n", t)
			}
		}
		return
	}

	// Restore.
	restored := 0
	if hasManifest {
		// Manifest path → absolute original path is exactly where we restore.
		for _, p := range manifestEntries {
			if err := v.RestoreFile(backupDir, p); err != nil {
				// Per-file restore failure is a warning, not fatal:
				// we still try the rest. Stderr because it's an error
				// condition (sweep finding P3-1).
				fmt.Fprintf(os.Stderr, msg.UndoRestoreFailed+"\n", p, err)
			} else {
				restored++
			}
		}
	} else {
		// Legacy: match by basename against targets in the log entry.
		filesDir := filepath.Join(backupDir, "files")
		files, _ := os.ReadDir(filesDir)
		for _, f := range files {
			name := f.Name()
			for _, t := range logTargets {
				if filepath.Base(t) == name {
					if err := v.RestoreFile(backupDir, t); err != nil {
						fmt.Fprintf(os.Stderr, msg.UndoRestoreFailed+"\n", t, err)
					} else {
						restored++
					}
					break
				}
			}
		}
	}

	if restored > 0 {
		fmt.Printf(msg.UndoRestored+"\n", restored)
	} else {
		fmt.Println(msg.UndoNoFilesRestored)
	}

	// Log the undo operation. A failure to write the audit entry is
	// surfaced as a warning to stderr — the restore itself already
	// succeeded, so this is degraded-but-still-correct, not fatal.
	// Previously the error was silently dropped, which meant a
	// failed audit-log write left no trace of the undo at all.
	logEntry := log.Entry{
		Command: entry.Command,
		Action:  msg.LevelUndo,
		Targets: entry.Targets,
		Message: fmt.Sprintf("undo of %s", entry.ID),
	}
	if logger, err := log.New(); err == nil {
		if err := logger.Append(logEntry); err != nil {
			// ErrLogWrite uses %w which is only valid inside
			// fmt.Errorf — going through warn → FmtWarn → Sprintf
			// would render it as "%!w(...)". Use a plain template
			// here. The cause is included verbatim.
			warn("failed to write undo entry to audit log: %v", err)
		}
	}
}

// readBackupManifest tries to read manifest.json under backupDir and
// returns the list of original absolute paths. The second return value
// is true only when a non-empty manifest was found, signalling callers
// to use the manifest-aware restore path. We deliberately keep this
// loose (any error → "no manifest") so a corrupt manifest falls back
// gracefully to the legacy basename-matching path.
func readBackupManifest(backupDir string) ([]string, bool) {
	data, err := os.ReadFile(filepath.Join(backupDir, "manifest.json"))
	if err != nil {
		return nil, false
	}
	var m struct {
		Files []struct {
			Original string `json:"original"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &m); err != nil || len(m.Files) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(m.Files))
	for _, fe := range m.Files {
		out = append(out, fe.Original)
	}
	return out, true
}
