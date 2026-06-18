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
	"github.com/hyper0x/cmdguard/internal/msg"
	"github.com/hyper0x/cmdguard/internal/vault"
)

// appendLog records an audit entry, swallowing logger-construction
// errors so a degraded log directory cannot block the user's
// destructive operation. The previous code repeated this pattern
// (`if logger, err := log.New(); err == nil { logger.Append(e) }`)
// 9+ times in this file alone — extracting it dedupes the boilerplate
// and makes the call sites read as audit statements rather than
// nested error-handling.
//
// Any failure to write the audit entry is intentionally silent at
// this layer: log.Append already prints a warning on its own when
// the underlying file write fails, and we don't want to leak a
// second confusing error to stderr from the caller.
func appendLog(entry log.Entry) {
	if logger, err := log.New(); err == nil {
		_ = logger.Append(entry)
	}
}

// RunGuard handles rm/mv/chmod commands
func RunGuard(cmdName string, args []string) {
	dryRun := false
	verbose := false
	bypassID := ""
	filteredArgs := make([]string, 0, len(args))

	for _, a := range args {
		switch {
		case a == "--check":
			fmt.Printf(msg.GuardCheckOK+"\n", cmdName)
			os.Exit(0)
		case a == "--dry-run":
			dryRun = true
		case a == "--verbose":
			verbose = true
		case a == "--version":
			if Version == "dev" {
				fmt.Printf("cmdguard %s (commit: %s)\n\n", Version, Commit)
			} else {
				fmt.Printf("cmdguard %s\n\n", Version)
			}
			if realCmd, err := findRealCommand(cmdName); err == nil {
				if output, err := exec.Command(realCmd, "--version").Output(); err == nil {
					os.Stdout.Write(output)
				}
			}
			os.Exit(0)
		case a == "--help":
			fmt.Print(msg.GuardHelp(cmdName))
			fmt.Println()
			if realCmd, err := findRealCommand(cmdName); err == nil {
				if output, err := exec.Command(realCmd, "--help").Output(); err == nil {
					os.Stdout.Write(output)
				}
			}
			os.Exit(0)
		case strings.HasPrefix(a, msg.GuardBypassFlag+"="):
			bypassID = strings.TrimPrefix(a, msg.GuardBypassFlag+"=")
		default:
			filteredArgs = append(filteredArgs, a)
		}
	}
	args = filteredArgs

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, msg.FmtErr(msg.ErrConfigLoad)+"\n", err)
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
		if verbose {
			fmt.Println(msg.GuardNoTargets)
		}
		execOriginal(cmdName, args, verbose)
		return
	}

	// Existence check.
	//
	// For rm/chmod: any non-existent target is a user error. We refuse
	// to invoke the underlying command and report the missing path so
	// the operator can correct the typo. cmdguard never silently filters
	// the target list — the user's input is preserved as-is.
	//
	// For mv: the destination (last argument) typically does NOT exist
	// (mv creates a new file). In that case there is no file to overwrite,
	// so no protection is needed — fall through to the underlying mv.
	// If the destination DOES exist, we keep the normal protection flow.
	if cmdName == "mv" {
		// guard.ExtractTargets returns only the destination for mv
		dest := targets[0]
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			if verbose {
				fmt.Println(msg.GuardMvDestNew)
			}
			execOriginal(cmdName, args, verbose)
			return
		}
	} else {
		// rm / chmod: every target must exist
		missing := false
		for _, t := range targets {
			if _, err := os.Stat(t); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, msg.GuardTargetMissing+"\n", cmdName, t)
				missing = true
			}
		}
		if missing {
			os.Exit(1)
		}
	}

	// Check protection rules
	result := guard.Check(cfg, cmdName, targets)

	if verbose {
		if result.Rule != "" {
			fmt.Printf(msg.GuardRuleMatched+"\n", result.Rule, result.Action)
		} else {
			fmt.Println(msg.GuardNoRule)
		}
		if result.Message != "" {
			fmt.Printf("%s\n", result.Message)
		}
	}

	switch result.Action {
	case msg.LevelReject:
		guard.PrintWarning(cmdName, result)
		if verbose {
			fmt.Println(msg.GuardRejected)
		}
		logEntry := log.Entry{
			Command: cmdName,
			Action:  msg.LevelReject,
			Targets: strings.Join(targets, ", "),
			Rule:    result.Rule,
			Message: result.Message,
		}
		appendLog(logEntry)
		os.Exit(1)

	case msg.LevelConfirmDbl, msg.LevelConfirm:
		// --bypass overrides interactive confirmation.
		// We do NOT log here — the unified entry (with Bypass field) is
		// written later in the backup+execute flow to avoid duplication.
		if bypassID != "" {
			if !msg.ValidateBypass(bypassID) {
				fmt.Fprintf(os.Stderr, msg.GuardBypassInvalid+"\n",
					bypassID,
					cmdName+" "+strings.Join(args, " "))
				logEntry := log.Entry{
					Command: cmdName,
					Action:  msg.LevelReject,
					Targets: strings.Join(targets, ", "),
					Rule:    result.Rule,
					Message: fmt.Sprintf(msg.GuardBypassInvalidMsg, bypassID),
				}
				appendLog(logEntry)
				os.Exit(1)
			}
			guard.PrintWarning(cmdName, result)
			// Fall through to backup + execute
			break
		}

		// Non-TTY mode: reject with guidance.
		// Two ways to detect non-interactive:
		//   1. CMDGUARD_NONINTERACTIVE env var is set (explicit, preferred for agents)
		//   2. stdin is not a TTY (e.g. piped, redirected)
		// Either signal skips the wait entirely.
		if config.IsNonInteractive() {
			emitNonTTYRejection(cmdName, args, targets, result, reasonEnv)
			os.Exit(1)
		}
		if !isTerminal() {
			emitNonTTYRejection(cmdName, args, targets, result, reasonNonTTY)
			os.Exit(1)
		}

		// Interactive confirmation
		guard.PrintWarning(cmdName, result)
		if dryRun {
			fmt.Printf(msg.GuardDryRunBackup+"\n", cmdName)
			os.Exit(0)
		}

		// Pick the per-level timeout from config.
		timeout := cfg.Guard.ConfirmTimeout
		if result.Action == msg.LevelConfirmDbl {
			timeout = cfg.Guard.ConfirmDoubleTimeout
		}

		// Hint: which keys to press, including timeout duration
		if timeout <= 0 {
			fmt.Fprintln(os.Stderr, msg.ConfirmTimeoutDisabled)
		}
		if result.Action == msg.LevelConfirmDbl {
			fmt.Fprintf(os.Stderr, msg.ConfirmDoubleHint+"\n", timeout)
		} else {
			fmt.Fprintf(os.Stderr, msg.ConfirmHint+"\n", timeout)
		}

		// First confirmation (with timeout fallback to non-TTY rejection)
		fmt.Fprint(os.Stderr, msg.ConfirmPrompt)
		answer, timedOut := readLineWithTimeout(timeout)
		if timedOut {
			emitNonTTYRejectionTimeout(cmdName, args, targets, result, timeout)
			os.Exit(1)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, msg.ConfirmCancelled)
			logEntry := log.Entry{
				Command: cmdName,
				Action:  msg.LevelReject,
				Targets: strings.Join(targets, ", "),
				Rule:    result.Rule,
				Message: msg.ConfirmCancelledMsg,
			}
			appendLog(logEntry)
			os.Exit(1)
		}

		// Second confirmation (only for confirm_double)
		if result.Action == msg.LevelConfirmDbl {
			fmt.Fprint(os.Stderr, msg.ConfirmDoublePrompt2)
			answer2, timedOut2 := readLineWithTimeout(timeout)
			if timedOut2 {
				emitNonTTYRejectionTimeout(cmdName, args, targets, result, timeout)
				os.Exit(1)
			}
			answer2 = strings.TrimSpace(strings.ToLower(answer2))
			if answer2 != "yes" {
				fmt.Fprintln(os.Stderr, msg.ConfirmCancelled)
				logEntry := log.Entry{
					Command: cmdName,
					Action:  msg.LevelReject,
					Targets: strings.Join(targets, ", "),
					Rule:    result.Rule,
					Message: msg.ConfirmDoubleCancelledMsg,
				}
				appendLog(logEntry)
				os.Exit(1)
			}
		}
		// Confirmed, fall through to backup + execute

	case msg.LevelWarn:
		guard.PrintWarning(cmdName, result)
		if dryRun {
			fmt.Printf(msg.GuardDryRunBackup+"\n", cmdName)
			os.Exit(0)
		}
		// Fall through to backup + execute

	case msg.LevelAllow:
		if dryRun {
			fmt.Printf(msg.GuardNoRule+" — %s will execute directly\n", cmdName)
			os.Exit(0)
		}
		logEntry := log.Entry{
			Command: cmdName,
			Action:  msg.LevelAllow,
			Targets: strings.Join(targets, ", "),
			Message: "no matching rule, allowed",
		}
		appendLog(logEntry)
		execOriginal(cmdName, args, verbose)
		return
	}

	// For confirm_double/confirm/warn: backup to vault then execute
	if result.Action == msg.LevelConfirmDbl || result.Action == msg.LevelConfirm || result.Action == msg.LevelWarn {
		if dryRun {
			fmt.Printf(msg.DryRunWillBackup+"\n", cmdName)
			for _, t := range targets {
				fmt.Printf("  - %s\n", t)
			}
			os.Exit(0)
		}
		v, err := vault.New(&cfg.Vault)
		if err != nil {
			fmt.Fprintf(os.Stderr, msg.FmtErr(msg.ErrVaultNew)+"\n", err)
			execOriginal(cmdName, args, verbose)
			return
		}

		entry := log.Entry{
			Command: cmdName,
			Action:  result.Action,
			Targets: strings.Join(targets, ", "),
			Rule:    result.Rule,
			Message: result.Message,
			Bypass:  bypassID,
		}
		entry.ID = log.NewID()
		entry.Timestamp = time.Now().Format(time.RFC3339)

		// For mv, store source paths in message for undo reference
		if cmdName == "mv" {
			allTargets := guard.ExtractAllTargets(args)
			if len(allTargets) > 1 {
				sources := allTargets[:len(allTargets)-1]
				entry.Message = fmt.Sprintf("src: %s → dst: %s", strings.Join(sources, ", "), targets[0])
			}
		}

		// Backup files to vault
		backupDir := v.BackupDir(entry.ID)

		if verbose {
			fmt.Printf(msg.VerboseBackupDir+"\n", backupDir)
		}

		if cmdName == "mv" {
			// For mv, we need to back up everything that *would be
			// overwritten*. Two cases:
			//
			//   1. Dest is an existing FILE — `mv src dst` overwrites
			//      dst directly. Back up dst.
			//
			//   2. Dest is an existing DIRECTORY — `mv a b dest/` lands
			//      each source as `dest/<basename(src)>`. For every src
			//      whose target slot already holds a regular file, that
			//      file is about to be overwritten. Back it up.
			//
			//   The previous implementation only handled case (1), so
			//   `mv old.conf existing-dir/` could silently overwrite
			//   `existing-dir/old.conf` with no recovery path. This is
			//   a data-loss bug; we now cover both paths.
			dest := targets[len(targets)-1]
			info, err := os.Stat(dest)
			switch {
			case err != nil:
				// Dest doesn't exist — `mv` will create it. No backup needed.
			case !info.IsDir():
				// Case 1: file destination.
				if verbose {
					fmt.Printf(msg.VerboseBackupFile+"\n", dest)
				}
				if _, err := v.SaveFile(backupDir, dest); err != nil {
					fmt.Fprintf(os.Stderr, msg.FmtWarn(msg.ErrVaultBackup)+"\n", dest, err)
				}
			default:
				// Case 2: directory destination.
				// Sources are everything except the last positional arg.
				allTargets := guard.ExtractAllTargets(args)
				if len(allTargets) > 1 {
					sources := allTargets[:len(allTargets)-1]
					for _, src := range sources {
						victim := filepath.Join(dest, filepath.Base(src))
						vi, err := os.Stat(victim)
						if err != nil || vi.IsDir() {
							continue
						}
						if verbose {
							fmt.Printf(msg.VerboseBackupFile+"\n", victim)
						}
						if _, err := v.SaveFile(backupDir, victim); err != nil {
							fmt.Fprintf(os.Stderr, msg.FmtWarn(msg.ErrVaultBackup)+"\n", victim, err)
						}
					}
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
					if verbose {
						fmt.Printf(msg.VerboseBackupFile+"\n", t)
					}
					if _, err := v.SaveFile(backupDir, t); err != nil {
						fmt.Fprintf(os.Stderr, msg.FmtWarn(msg.ErrVaultBackup)+"\n", t, err)
					}
				}
			}
		}

		// Log the operation
		appendLog(entry)

		if verbose {
			fmt.Printf(msg.GuardExecuting+"\n", cmdName, strings.Join(args, " "))
		}

		execOriginal(cmdName, args, verbose)
	}
}

// execOriginal executes the original system command
func execOriginal(cmdName string, args []string, verbose bool) {
	realCmd, err := findRealCommand(cmdName)
	if err != nil {
		fmt.Fprintf(os.Stderr, msg.FmtErr(msg.ErrCmdNotFound)+"\n", cmdName)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf(msg.GuardExecuting+"\n", realCmd, strings.Join(args, " "))
	}

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

// findRealCommand finds the real command, skipping cmdguard itself and
// its wrapper scripts. The lookup walks PATH in declaration order and
// returns the first match that is:
//
//   - a regular, executable file
//   - not the running cmdguard binary (so we don't recurse)
//   - not located inside cmdguard's own bin/ wrapper directory
//
// We use filepath.SplitList instead of strings.Split(":") because
// SplitList honours the OS path separator (Unix=':', Windows=';') and
// — crucially — preserves empty entries as ".", matching the shell's
// own PATH semantics. A subtle correctness fix more than a portability
// one: cmdguard targets Unix, but inheriting a PATH like ":/usr/bin"
// would otherwise silently skip the implicit CWD.
//
// We also skip our own wrapper dir BEFORE the os.SameFile dance: that
// prefix check is a cheap string compare, while SameFile costs two
// stat calls. Same correctness, less I/O when invoked in tight loops.
func findRealCommand(name string) (string, error) {
	pathEnv := os.Getenv("PATH")
	dirs := filepath.SplitList(pathEnv)

	self, _ := os.Executable()
	var selfInfo os.FileInfo
	if self != "" {
		selfInfo, _ = os.Stat(self)
	}
	cfgDir := config.ConfigDir()
	binDir := filepath.Join(cfgDir, "bin")
	binPrefix := binDir + string(filepath.Separator)

	for _, dir := range dirs {
		fullPath := filepath.Join(dir, name)

		// Cheap: skip our own wrapper directory without stat'ing.
		if strings.HasPrefix(fullPath, binPrefix) {
			continue
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			continue
		}

		// Skip if this match is the running cmdguard binary itself —
		// otherwise an unguarded `rm` invocation could recurse forever.
		if selfInfo != nil && os.SameFile(selfInfo, info) {
			continue
		}

		return fullPath, nil
	}
	return "", fmt.Errorf(msg.ErrCmdNotFound, name)
}

// isTerminal checks whether stdin is a terminal (TTY)
func isTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// readLineWithTimeout reads a single line from stdin, returning early
// if no input arrives within the given number of seconds.
// A seconds <= 0 disables the timeout (waits forever).
//
// Why timeout exists: isTerminal() can return true in environments
// where stdin behaves like a TTY but no human is actually present
// (some agent sandboxes, pseudo-TTY allocations, terminals where the
// user wandered off). A timeout fallback prevents the process from
// hanging forever — we treat the silence as non-interactive and
// reject the operation with the standard bypass guidance.
func readLineWithTimeout(seconds int) (string, bool) {
	ch := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		ch <- line
	}()
	if seconds <= 0 {
		// Timeout disabled — block until input arrives or the
		// process is killed (Ctrl+C).
		return <-ch, false
	}
	select {
	case line := <-ch:
		return line, false
	case <-time.After(time.Duration(seconds) * time.Second):
		// Print a newline so the timeout message appears on its own line,
		// not on the same line as the prompt.
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, msg.ConfirmTimeoutMsg+"\n", seconds)
		return "", true
	}
}

// nonTTYReason identifies why cmdguard fell into the non-interactive
// rejection path. It controls only the log message — the user-facing
// guidance is the same in all cases.
type nonTTYReason int

const (
	reasonNonTTY  nonTTYReason = iota // stdin is not a TTY (or unspecified)
	reasonEnv                         // CMDGUARD_NONINTERACTIVE is set
	reasonTimeout                     // interactive prompt timed out
)

// emitNonTTYRejection prints the standard non-interactive rejection
// block and writes the corresponding log entry.
func emitNonTTYRejection(cmdName string, args, targets []string, result *guard.Result, reason nonTTYReason) {
	fmt.Fprintln(os.Stderr, msg.GuardNonTTYRejected)
	guard.PrintWarning(cmdName, result)
	fmt.Fprintf(os.Stderr, msg.GuardBypassHelp+"\n",
		result.Action,
		cmdName+" "+strings.Join(args, " "))

	var logMsg string
	switch reason {
	case reasonEnv:
		logMsg = msg.GuardEnvNonInteractiveMsg
	case reasonNonTTY:
		logMsg = msg.GuardNonTTYMsg
	}
	logEntry := log.Entry{
		Command: cmdName,
		Action:  msg.LevelReject,
		Targets: strings.Join(targets, ", "),
		Rule:    result.Rule,
		Message: logMsg,
	}
	appendLog(logEntry)
}

// emitNonTTYRejectionTimeout is the timeout-specific variant: it
// records the actual timeout that elapsed so the audit log captures
// the operative setting at the time.
func emitNonTTYRejectionTimeout(cmdName string, args, targets []string, result *guard.Result, seconds int) {
	fmt.Fprintln(os.Stderr, msg.GuardNonTTYRejected)
	guard.PrintWarning(cmdName, result)
	fmt.Fprintf(os.Stderr, msg.GuardBypassHelp+"\n",
		result.Action,
		cmdName+" "+strings.Join(args, " "))

	logEntry := log.Entry{
		Command: cmdName,
		Action:  msg.LevelReject,
		Targets: strings.Join(targets, ", "),
		Rule:    result.Rule,
		Message: fmt.Sprintf(msg.ConfirmTimeoutLogMsg, seconds),
	}
	appendLog(logEntry)
}

func printGuardHelp(cmdName string) {
	fmt.Print(msg.GuardHelp(cmdName))
}
