package subcmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// backupFiles creates a zip archive of all files to be overwritten
func backupFiles(files []string) (string, error) {
	backupDir := filepath.Join(config.ConfigDir(), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	ts := time.Now().Format("20060102_150405")
	zipPath := filepath.Join(backupDir, fmt.Sprintf("init-%s.zip", ts))

	z, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	// Don't `defer z.Close()` here: this is a write path, and a
	// deferred Close would swallow any error from flushing the file's
	// final bytes to disk — which is exactly the kind of failure that
	// produces a silently-truncated backup zip. We close explicitly
	// after the zip writer has flushed its central directory, and
	// surface either error to the caller. See the explicit close
	// block at the end of this function.

	w := zip.NewWriter(z)
	// Same reasoning for w: zip.Writer.Close() writes the central
	// directory record. If that fails the archive is corrupt and
	// unreadable. Must be reported, not deferred-and-dropped.

	cfgDir := config.ConfigDir()

	// addOne keeps the per-file open/close lifetime scoped to a single
	// iteration. The previous implementation used `defer in.Close()`
	// directly inside the loop, which kept every file handle open until
	// the function returned — a leak for large `files` lists.
	addOne := func(src string) error {
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		// Read-only handle on the source: best-effort close is fine
		// because we only consumed bytes; a Close error here can't
		// corrupt anything we care about.
		defer func() { _ = in.Close() }()

		info, err := in.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		// Preserve relative path from ~/.cmdguard/
		rel, err := filepath.Rel(cfgDir, src)
		if err != nil {
			rel = filepath.Base(src)
		}
		header.Name = rel
		header.Method = zip.Deflate

		out, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		_, err = io.Copy(out, in)
		return err
	}

	for _, src := range files {
		if err := addOne(src); err != nil {
			// On any per-file error, still try to close both writers
			// so we don't leak fds; ignore their close errors because
			// the original cause is more informative.
			_ = w.Close()
			_ = z.Close()
			return "", err
		}
	}

	// Flush the zip central directory FIRST, then close the underlying
	// file. If either fails, the archive is unreliable — return the
	// error so callers know the backup didn't actually complete.
	if err := w.Close(); err != nil {
		_ = z.Close()
		return "", fmt.Errorf("finalize backup zip: %w", err)
	}
	if err := z.Close(); err != nil {
		return "", fmt.Errorf("close backup zip: %w", err)
	}

	return zipPath, nil
}

// RunInit handles the "init" command
func RunInit(args []string) {
	force := false
	dryRun := false
	for _, a := range args {
		switch a {
		case "--force":
			force = true
		case "--dry-run":
			dryRun = true
		default:
			rejectUnknownArg(a, "init")
		}
	}

	cfgDir := config.ConfigDir()
	binDir := config.BinDir()
	logDir := filepath.Join(cfgDir, "log")
	vaultDir := filepath.Join(cfgDir, "vault")

	if dryRun {
		overwrite := msg.OverwriteLabel(force)
		fmt.Println(msg.InitDryRunHeader)

		// 1. Directory creation — mirrors real-run step 1.
		fmt.Printf(msg.InitDryRunCreateDir+"\n", cfgDir)
		fmt.Printf(msg.InitDryRunCreateDir+"\n", binDir)
		fmt.Printf(msg.InitDryRunCreateDir+"\n", logDir)
		fmt.Printf(msg.InitDryRunCreateDir+"\n", vaultDir)

		// 2. Backup — mirrors real-run step 3 (which runs before
		//    config/scripts are written). Previously this line was
		//    printed AT THE END of dry-run, so users got a misleading
		//    preview: dry-run said "config will be overwritten, then
		//    scripts, then a backup will be made", but the real flow
		//    backs up FIRST. Move it up so the preview matches reality.
		cfgPath := config.ConfigPath()
		_, cfgStatErr := os.Stat(cfgPath)
		cfgExists := !os.IsNotExist(cfgStatErr)
		willBackup := false
		if force && cfgExists {
			willBackup = true
		}
		if !willBackup {
			for _, cmd := range GuardedCommands() {
				scriptPath := filepath.Join(binDir, cmd)
				if _, err := os.Stat(scriptPath); force && !os.IsNotExist(err) {
					willBackup = true
					break
				}
			}
		}
		if willBackup {
			fmt.Println(msg.InitDryRunBackup)
		}

		// 3. Config file.
		if !cfgExists {
			fmt.Printf(msg.InitDryRunCreateFile+"\n", cfgPath)
		} else {
			fmt.Printf(msg.InitDryRunExists+"\n", cfgPath, overwrite)
		}

		// 4. Wrapper scripts.
		for _, cmd := range GuardedCommands() {
			scriptPath := filepath.Join(binDir, cmd)
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				fmt.Printf(msg.InitDryRunCreateScript+"\n", scriptPath)
			} else {
				fmt.Printf(msg.InitDryRunExistsScript+"\n", scriptPath, overwrite)
			}
		}
		os.Exit(0)
	}

	// 1. Create directory structure
	dirs := []string{cfgDir, binDir, logDir, vaultDir}
	for _, d := range dirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			fmt.Printf(msg.InitDirExists+"\n", d)
		} else {
			if err := os.MkdirAll(d, 0755); err != nil {
				errExit(msg.ErrMkdir, d, err)
			}
			fmt.Printf(msg.InitDirCreated+"\n", d)
		}
	}

	// 2. Collect files to backup (only when --force and they exist)
	var toBackup []string
	cfgPath := config.ConfigPath()
	_, cfgStatErr := os.Stat(cfgPath)
	cfgExists := !os.IsNotExist(cfgStatErr)

	if force && cfgExists {
		toBackup = append(toBackup, cfgPath)
	}

	for _, cmd := range GuardedCommands() {
		scriptPath := filepath.Join(binDir, cmd)
		_, err := os.Stat(scriptPath)
		if force && !os.IsNotExist(err) {
			toBackup = append(toBackup, scriptPath)
		}
	}

	// 3. Backup if needed
	if len(toBackup) > 0 {
		zipPath, err := backupFiles(toBackup)
		if err != nil {
			errExit("backup failed: %v", err)
		}
		fmt.Printf(msg.InitBackupCreated+"\n", zipPath)
	}

	// 4. Generate default config
	needWriteConfig := !cfgExists
	if force && cfgExists {
		needWriteConfig = true
	}

	if needWriteConfig {
		defaultCfg := `# cmdguard configuration file
# Path protection rules using glob patterns (** matches any depth)
# Organized by protection level (strict to lenient):
#
#   reject           🚫   directly rejected, not executed
#   confirm_double   🔒   warning + double confirmation (type 'yes') → backup → execute
#   confirm          ❓   warning + single confirmation (press 'y') → backup → execute
#   warn             ⚠️   warning + backup → execute
#
# Rule matching order: reject → confirm_double → confirm → warn
# First match wins, so stricter rules should come first.
#
# ⚠️  IMPORTANT: Source of truth
# This file IS the source of truth for protection rules when it defines
# a [protect] section — built-in defaults are NOT merged on top.
#
#   • File has [protect] → file's rules are used as-is (no defaults added).
#     If you want NO reject rules, write "reject = []".
#   • File has NO [protect] section → built-in defaults apply.
#   • Vault and Guard use field-level merge: only the keys you write
#     override defaults; omitted keys keep their default values.
#
# The file generated by 'cmdguard init' includes the full default set.
# To customise, edit or remove the sections you want to change.

[protect]
reject = [
  # macOS/Linux system directories
  "/bin/**",
  "/boot/**",
  "/dev/**",
  "/etc/**",
  "/lib/**",
  "/lib64/**",
  "/proc/**",
  "/sbin/**",
  "/sys/**",
  "/usr/**",
  "/var/**",
  "/opt/**",
  "/private/**",
  "/System/**",
  "/Library/**",
  "/Applications/**",
  # Key files — non-recoverable
  "*.key",
  "*.pem",
  "*.crt",
  "*.p12",
  "*.pfx",
  "*.asc",
  # Home directory critical config — non-recoverable
  "~/.ssh/**",
  "~/.gnupg/**",
  "~/.aws/**",
]

confirm_double = [
  # Home directory app data — cleanable but requires double confirmation
  "~/.config/**",
  "~/.local/share/**",
]

confirm = [
  # Documents directory — confirm
  "~/Documents/**",
  # Desktop files — confirm
  "~/Desktop/**",
]

warn = [
  # Downloads directory — warn + backup
  "~/Downloads/**",
]

# Command-level overrides — apply stricter protection for specific commands
# Example: global rules set ~/.config/** to confirm_double,
# but rm is more dangerous, so override to reject for rm
[protect.command.rm]
reject = [
  "~/.config/**",
]

# mv protects the destination path (last argument).
# Reject mv into ~/Downloads/** to prevent accidental overwrites
[protect.command.mv]
reject = [
  "~/Downloads/**",
]

# chmod changes permissions. Reject for app config directories
# (global rules set ~/.config/** to confirm_double)
[protect.command.chmod]
reject = [
  "~/.config/**",
]

# Vault backup settings
[vault]
retention_days = 7
auto_purge = true

# Interactive prompt settings.
#
# Seconds to wait at each interactive confirmation prompt before
# falling back to non-interactive rejection (with --bypass guidance).
# Set to 0 to disable the timeout — only do this on personal machines
# where no automation will ever hit a confirm-level path, otherwise
# stray invocations may hang forever.
#
# AI agents and automation should set the env var instead:
#   export CMDGUARD_NONINTERACTIVE=1
# which skips the wait entirely (rejection is immediate).
[guard]
confirm_timeout = 5          # seconds; 'confirm' prompt
confirm_double_timeout = 10  # seconds per step; 'confirm_double' prompt
`
		if err := os.WriteFile(cfgPath, []byte(defaultCfg), 0644); err != nil {
			errExit(msg.ErrWriteFile, cfgPath, err)
		}
		if force {
			fmt.Printf(msg.InitConfigOverwritten+"\n", cfgPath)
		} else {
			fmt.Printf(msg.InitConfigCreated+"\n", cfgPath)
		}
	} else {
		fmt.Printf(msg.InitConfigExists+"\n", cfgPath)
	}

	// 5. Create wrapper scripts in ~/.cmdguard/bin/
	selfPath, err := os.Executable()
	if err != nil {
		errExit("failed to get executable path: %v", err)
	}

	for _, cmd := range GuardedCommands() {
		scriptPath := filepath.Join(binDir, cmd)
		_, err := os.Stat(scriptPath)
		needWriteScript := os.IsNotExist(err)

		if force && !os.IsNotExist(err) {
			needWriteScript = true
		}

		if needWriteScript {
			// Sentinel line lets findRealCommand recognise this file
			// as a cmdguard wrapper and skip it during PATH lookup —
			// crucial when multiple cmdguard installs coexist or when
			// a sandboxed CMDGUARD_CONFIG_DIR is layered on top of an
			// existing ~/.cmdguard/bin/. Without it, recursion ensues.
			script := fmt.Sprintf(`#!/bin/bash
%s
exec %s %s "$@"
`, WrapperSentinel, selfPath, cmd)
			if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
				errExit(msg.ErrWriteFile, scriptPath, err)
			}
			if force {
				fmt.Printf(msg.InitScriptOverwritten+"\n", scriptPath)
			} else {
				fmt.Printf(msg.InitScriptCreated+"\n", scriptPath)
			}
		} else {
			fmt.Printf(msg.InitScriptExists+"\n", scriptPath)
		}
	}

	// 6. Print shell integration guide
	fmt.Println()
	fmt.Print(msg.InitIntegrationGuide)
}
