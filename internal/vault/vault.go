package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hyper0x/cmdguard/internal/config"
)

// Backup represents a single vault backup entry
type Backup struct {
	ID        string
	Timestamp time.Time
	Dir       string // full path to the backup directory
	Command   string
	Source    string // original path
	Target    string // for mv: destination path
	Files     []string
}

// ManifestVersion is bumped when the manifest schema changes in a way
// that older code cannot read. v1 is the initial release.
const ManifestVersion = 1

// manifestFileName is the per-backup metadata file. Stored alongside
// the files/ tree inside each backup directory.
const manifestFileName = "manifest.json"

// ManifestEntry records one saved file: its original absolute path
// and where it lives inside the backup's files/ tree (a forward-slash
// path relative to files/).
type ManifestEntry struct {
	OriginalPath string `json:"original"`
	StoredAs     string `json:"stored"`
}

// Manifest is the structure persisted as manifest.json in each
// backup directory. It is the authoritative record of what was
// backed up; the on-disk layout under files/ is just a content
// store keyed by manifest entries.
type Manifest struct {
	Version int             `json:"version"`
	Files   []ManifestEntry `json:"files"`
}

// Vault manages file backups
type Vault struct {
	dir string
	cfg *config.VaultConfig
}

// New creates a new Vault instance
func New(cfg *config.VaultConfig) (*Vault, error) {
	dir := filepath.Join(config.ConfigDir(), "vault")
	// 0700: vault holds copies of files the user just deleted. Many
	// of those originals (~/.ssh/*, /etc/shadow, etc.) were 0600 or
	// stricter; the backup must not be world-readable. Restricting
	// the top-level directory means even if a backup file slips
	// through with permissive mode, no other principal can list or
	// stat its way to it.
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create vault directory: %w", err)
	}
	return &Vault{dir: dir, cfg: cfg}, nil
}

// BackupDir returns the path for a new backup
func (v *Vault) BackupDir(id string) string {
	ts := time.Now().Format("20060102_150405")
	return filepath.Join(v.dir, fmt.Sprintf("%s_%s", ts, id))
}

// SaveFile copies a file into the vault, preserving the source file's
// permission bits, and records the original path in the backup's
// manifest. The on-disk layout uses the original absolute path
// (with a leading "/" stripped) so files with the same basename
// from different directories no longer collide:
//
//   /etc/foo.conf  →  files/etc/foo.conf
//   /opt/foo.conf  →  files/opt/foo.conf
//
// Preserving mode matters because restoring a file like ~/.ssh/id_rsa
// with default 0644 would silently weaken its security posture
// (sshd refuses keys with permissive modes).
func (v *Vault) SaveFile(backupDir, srcPath string) (string, error) {
	// 0700 on per-backup directory: same rationale as the vault root
	// — only the owner needs to traverse this. The directory mode
	// does not need to relax for files inside (they keep their
	// original mode via info.Mode().Perm() below).
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Resolve to an absolute, cleaned path so the manifest record is
	// unambiguous. Falling back to srcPath keeps the call working even
	// if Abs fails (e.g. on a deleted CWD), at the cost of a less
	// canonical record.
	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		absSrc = filepath.Clean(srcPath)
	}

	stored := storedPathFor(absSrc)
	destPath := filepath.Join(backupDir, "files", stored)
	// 0700 again: parent dirs of files/etc/foo.conf must not leak
	// the names of backed-up files via directory listing.
	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return "", fmt.Errorf("failed to create files directory: %w", err)
	}

	// Read source file along with its mode.
	info, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat source file: %w", err)
	}
	// #nosec G304 G703 -- srcPath is the file the user is about to
	// destroy via rm/mv/chmod; the whole point of the vault is to
	// read its contents before destruction. Refusing variable paths
	// would defeat the tool. Path-traversal isn't a concern: srcPath
	// originates from argv, which the user has both intent and the
	// fs permission to access.
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	// #nosec G703 -- destPath is computed by joining the vault root
	// with storedPathFor(absSrc); cmdguard controls both halves of
	// that path and the resulting write target stays inside the
	// vault by construction. Same trust model as the ReadFile above.
	if err := os.WriteFile(destPath, data, info.Mode().Perm()); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	if err := v.appendManifest(backupDir, ManifestEntry{
		OriginalPath: absSrc,
		StoredAs:     filepath.ToSlash(stored),
	}); err != nil {
		return "", fmt.Errorf("failed to update manifest: %w", err)
	}

	return destPath, nil
}

// RestoreFile restores a file from vault to its original location,
// preserving the mode captured at backup time. The lookup is by the
// destination's absolute path (matched against the manifest), so
// same-basename files in different directories are restored to the
// correct location.
//
// Falls back to the legacy basename layout (files/<basename>) for
// backups created before manifest support, so existing vaults remain
// usable after upgrade.
func (v *Vault) RestoreFile(backupDir, destPath string) error {
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		absDest = filepath.Clean(destPath)
	}

	srcPath := ""

	// Preferred: look up via manifest.
	if m, err := v.readManifest(backupDir); err == nil {
		for _, fe := range m.Files {
			if fe.OriginalPath == absDest {
				srcPath = filepath.Join(backupDir, "files", filepath.FromSlash(fe.StoredAs))
				break
			}
		}
	}

	// Legacy fallback: basename layout.
	if srcPath == "" {
		legacy := filepath.Join(backupDir, "files", filepath.Base(destPath))
		if _, err := os.Stat(legacy); err == nil {
			srcPath = legacy
		}
	}

	if srcPath == "" {
		return fmt.Errorf("no backup found for %s", destPath)
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}
	// #nosec G304 G703 -- restore path. srcPath is a vault-internal
	// file under backupDir/files/<storedPathFor(...)>; destPath is
	// the original absolute path we read from the manifest at
	// backup time. Both come from cmdguard-controlled bookkeeping,
	// not user argv at restore time, so taint analysis flagging
	// "variable" here is a false positive.
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// #nosec G703 -- destPath is the OriginalPath recorded in the
	// manifest at backup time; restoring to it is the entire point
	// of `cmdguard undo`. Same trust model as ReadFile above.
	if err := os.WriteFile(destPath, data, info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// storedPathFor returns the relative path under <backup>/files/
// where the given original absolute path is stored. We strip the
// leading "/" so the path stays relative; on Windows we'd also
// strip the drive letter, but cmdguard targets Unix.
func storedPathFor(absSrc string) string {
	rel := absSrc
	if strings.HasPrefix(rel, string(filepath.Separator)) {
		rel = rel[1:]
	}
	if rel == "" {
		// Pathological: someone tried to back up the root. Use a
		// placeholder name; the manifest still has the real path.
		rel = "_root_"
	}
	return rel
}

// readManifest loads the manifest.json from backupDir, returning an
// empty manifest if the file does not exist (so callers can treat
// "no manifest" as the legacy/basename layout).
func (v *Vault) readManifest(backupDir string) (*Manifest, error) {
	path := filepath.Join(backupDir, manifestFileName)
	// #nosec G304 -- backupDir is the vault-controlled directory
	// for this backup ID; we're reading the cmdguard-written
	// manifest.json inside it. Not user-supplied path data.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Version: ManifestVersion}, nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	return &m, nil
}

// appendManifest adds a single entry to backupDir/manifest.json,
// preserving any existing entries. We rewrite the whole file each
// time because manifests are tiny (a few entries per backup) and
// the simpler write-rewrite avoids partial-write corruption that
// pure append would expose.
func (v *Vault) appendManifest(backupDir string, entry ManifestEntry) error {
	m, err := v.readManifest(backupDir)
	if err != nil {
		return err
	}
	if m.Version == 0 {
		m.Version = ManifestVersion
	}

	// Replace any prior entry for the same OriginalPath so a single
	// backup that re-saves the same file (unlikely but defensible)
	// does not accumulate duplicates.
	replaced := false
	for i, fe := range m.Files {
		if fe.OriginalPath == entry.OriginalPath {
			m.Files[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		m.Files = append(m.Files, entry)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	// 0600: the manifest records every original absolute path that
	// was backed up. Treat that index with the same privacy as the
	// audit log — if a user deletes ~/.ssh/id_rsa, we don't want a
	// world-readable manifest at $vault/$backup/manifest.json
	// announcing that fact.
	return os.WriteFile(filepath.Join(backupDir, manifestFileName), data, 0600)
}

// PurgeExpired removes backups older than retention_days
// Returns the list of purged backup IDs
func (v *Vault) PurgeExpired() ([]string, error) {
	if v.cfg.RetentionDays <= 0 {
		return nil, nil
	}

	cutoff := time.Now().AddDate(0, 0, -v.cfg.RetentionDays)
	var purged []string

	entries, err := os.ReadDir(v.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read vault directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse timestamp from directory name: 20260606_103005_abc123
		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) < 3 {
			continue
		}

		t, err := time.Parse("20060102_150405", parts[0]+"_"+parts[1])
		if err != nil {
			continue
		}

		if t.Before(cutoff) {
			backupPath := filepath.Join(v.dir, entry.Name())
			if err := os.RemoveAll(backupPath); err != nil {
				fmt.Fprintf(os.Stderr, "[cmdguard] warning: failed to delete expired backup %s: %v\n", entry.Name(), err)
				continue
			}
			purged = append(purged, parts[2]) // the ID
		}
	}

	return purged, nil
}

// ListExpired returns backup directories older than retention_days (without deleting)
func (v *Vault) ListExpired() ([]string, error) {
	if v.cfg.RetentionDays <= 0 {
		return nil, nil
	}

	cutoff := time.Now().AddDate(0, 0, -v.cfg.RetentionDays)
	var expired []string

	entries, err := os.ReadDir(v.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read vault directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) < 3 {
			continue
		}

		t, err := time.Parse("20060102_150405", parts[0]+"_"+parts[1])
		if err != nil {
			continue
		}

		if t.Before(cutoff) {
			expired = append(expired, entry.Name())
		}
	}

	sort.Strings(expired)
	return expired, nil
}

// BackupInfo holds display info for a vault backup entry
type BackupInfo struct {
	ID        string
	Timestamp time.Time
	Dir       string // full path to the backup directory
	Files     []string
	Expired   bool
}

// ListAll returns all backup entries sorted by timestamp (newest first).
func (v *Vault) ListAll() ([]BackupInfo, error) {
	entries, err := os.ReadDir(v.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read vault directory: %w", err)
	}

	var cutoff time.Time
	if v.cfg.RetentionDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -v.cfg.RetentionDays)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) < 3 {
			continue
		}

		t, err := time.Parse("20060102_150405", parts[0]+"_"+parts[1])
		if err != nil {
			continue
		}

		backupDir := filepath.Join(v.dir, entry.Name())
		fileNames := listBackupFiles(backupDir)

		expired := false
		if !cutoff.IsZero() && t.Before(cutoff) {
			expired = true
		}

		backups = append(backups, BackupInfo{
			ID:        parts[2],
			Timestamp: t,
			Dir:       backupDir,
			Files:     fileNames,
			Expired:   expired,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// BackupExists checks if a backup directory exists for the given ID (supports prefix)
func (v *Vault) BackupExists(id string) bool {
	entries, err := os.ReadDir(v.dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) == 3 && (parts[2] == id || strings.HasPrefix(parts[2], id)) {
			return true
		}
	}
	return false
}

// FindBackupDir finds the backup directory for a given ID (supports prefix)
func (v *Vault) FindBackupDir(id string) string {
	entries, err := os.ReadDir(v.dir)
	if err != nil {
		return ""
	}

	var match string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 3)
		if len(parts) == 3 && (parts[2] == id || strings.HasPrefix(parts[2], id)) {
			if match != "" {
				// Ambiguous prefix
				return ""
			}
			match = filepath.Join(v.dir, entry.Name())
		}
	}
	return match
}

// listBackupFiles returns the original paths of all files saved in a
// backup directory. It prefers manifest.json (one entry per save call,
// recorded with the file's original absolute path) and falls back to
// scanning files/ for legacy backups created before manifest support.
//
// Display callers (path / vault list) only need *names* to show, but
// returning the original paths keeps the data more useful — and the
// undo flow uses this to know which files belong to a backup without
// guessing from basenames.
func listBackupFiles(backupDir string) []string {
	manifestPath := filepath.Join(backupDir, manifestFileName)
	// #nosec G304 -- manifestPath is constructed from the
	// vault-controlled backupDir + a fixed filename, not user input.
	if data, err := os.ReadFile(manifestPath); err == nil {
		var m Manifest
		if json.Unmarshal(data, &m) == nil && len(m.Files) > 0 {
			out := make([]string, 0, len(m.Files))
			for _, fe := range m.Files {
				out = append(out, fe.OriginalPath)
			}
			return out
		}
	}

	// Legacy fallback: list whatever filenames live under files/.
	filesDir := filepath.Join(backupDir, "files")
	var names []string
	if fEntries, err := os.ReadDir(filesDir); err == nil {
		for _, fe := range fEntries {
			if !fe.IsDir() {
				names = append(names, fe.Name())
			}
		}
	}
	return names
}
