package subcmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hyper0x/cmdguard/internal/config"
	"github.com/hyper0x/cmdguard/internal/msg"
)

// RunPath handles the "path" command.
func RunPath(args []string) {
	// path takes no flags or positional args. Reject anything to keep
	// the contract honest — silent acceptance of garbage hides typos
	// such as `cmdguard path --raw` (user thought of `config --raw`).
	if len(args) > 0 {
		rejectUnknownArg(args[0], "path")
	}

	cfgDir := config.ConfigDir()
	cfgPath := config.ConfigPath()

	fmt.Println(msg.PathHeader)
	fmt.Println()

	// Config directory
	fmt.Printf(msg.PathConfigDir+"\n", cfgDir)

	// Config file
	cfgFileInfo := msg.PathFileNotExist
	if fi, err := os.Stat(cfgPath); err == nil {
		// Count lines
		lineCount := 0
		// #nosec G304 -- cfgPath is config.ConfigPath(); controlled.
		if data, err := os.ReadFile(cfgPath); err == nil {
			lineCount = bytes.Count(data, []byte{'\n'})
		}
		cfgFileInfo = fmt.Sprintf("%s, %d lines", formatFileSize(fi.Size()), lineCount)
	}
	fmt.Printf(msg.PathConfigFile+"\n", cfgPath, cfgFileInfo)

	// Log directory
	logDir := filepath.Join(cfgDir, "log")
	fmt.Println()
	fmt.Printf(msg.PathLogDir+"\n", logDir)
	printDirFiles(logDir, ".log")

	// Vault directory
	vaultDir := filepath.Join(cfgDir, "vault")
	fmt.Println()
	fmt.Printf(msg.PathVaultDir+"\n", vaultDir)
	printVaultInfo(vaultDir)

	// Bin directory
	binDir := config.BinDir()
	fmt.Println()
	fmt.Printf(msg.PathBinDir+"\n", binDir)
	printDirFiles(binDir, "")
}

// printDirFiles lists files in a directory, newest first, with a count summary.
// extFilter filters by extension (e.g. ".log"), empty means show all.
func printDirFiles(dir string, extFilter string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  (%s)\n", msg.PathDirNotExist)
		} else {
			fmt.Printf("  (%s: %v)\n", msg.PathDirError, err)
		}
		return
	}

	// Filter and collect file info
	type fileInfo struct {
		name    string
		modTime int64 // for sorting
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if extFilter != "" && !strings.HasSuffix(e.Name(), extFilter) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: e.Name(), modTime: info.ModTime().Unix()})
	}

	if len(files) == 0 {
		fmt.Printf("  (%s)\n", msg.PathDirEmpty)
		return
	}

	// Sort newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	// Show up to 5 files
	showCount := min(len(files), 5)
	for i := range showCount {
		prefix := "├─ "
		if i == showCount-1 {
			prefix = "└─ "
		}
		fmt.Printf("  %s%s\n", prefix, files[i].name)
	}

	if len(files) > 5 {
		fmt.Printf("  ...\n")
	}

	// Summary
	summary := fmt.Sprintf(msg.PathFileCount, len(files))
	fmt.Printf("  (%s)\n", summary)
}

// printVaultInfo shows vault summary: backup count and disk usage.
func printVaultInfo(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  (%s)\n", msg.PathDirNotExist)
		} else {
			fmt.Printf("  (%s: %v)\n", msg.PathDirError, err)
		}
		return
	}

	var backupDirs []os.DirEntry
	var totalSize int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		backupDirs = append(backupDirs, e)
		// Estimate disk usage by walking files/ recursively. Backups
		// now use a nested layout (files/etc/foo.conf) so a single
		// ReadDir would miss most of the bytes.
		filesDir := filepath.Join(dir, e.Name(), "files")
		_ = filepath.WalkDir(filesDir, func(_ string, d os.DirEntry, err error) error {
			// On per-entry walk errors (permission denied, broken
			// symlink, race with concurrent vault clean) we skip
			// the entry and keep walking. This is a best-effort
			// size estimate for `cmdguard path` — an underestimate
			// is acceptable, but refusing to print anything because
			// of one unreadable file would be a worse experience.
			//
			// nilerr flags this because we receive a non-nil err
			// and return nil; that's intentional here per the
			// filepath.WalkDir contract (return nil = skip this
			// entry, keep walking).
			if err != nil {
				return nil //nolint:nilerr // best-effort size walk
			}
			if d.IsDir() {
				return nil
			}
			if info, err := d.Info(); err == nil {
				totalSize += info.Size()
			}
			return nil
		})
	}

	if len(backupDirs) == 0 {
		fmt.Printf("  (%s)\n", msg.PathDirEmpty)
		return
	}

	fmt.Printf("  (%s)\n", fmt.Sprintf(msg.PathVaultSummary, len(backupDirs), formatFileSize(totalSize)))
}

// formatFileSize returns a human-readable file size.
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
