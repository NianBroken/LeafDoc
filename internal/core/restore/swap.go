package restore

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// backupMeta mirrors internal/core/snapshot's unexported backupMeta struct — the
// shape written to backup-meta.json inside every snapshot ZIP.
type backupMeta struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Version   string    `json:"version"`
}

// Required entries distinguish the versioned MySQL format from older archives.
var requiredZipEntries = []string{"backup-meta.json", "database.json", "checksums.json"}

// extractAndValidate verifies paths, size budgets, checksums and the database format.
// The returned staging directory is owned and removed by the caller.
func extractAndValidate(zipPath, dataDir string) (stagingDir string, meta backupMeta, err error) {
	return extractAndValidateWithLimits(zipPath, dataDir, shared.DefaultZipExtractionLimits)
}

func extractAndValidateWithLimits(zipPath, dataDir string, limits shared.ExtractionLimits) (stagingDir string, meta backupMeta, err error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", backupMeta{}, fmt.Errorf("failed to open snapshot zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	present := map[string]bool{}
	for _, f := range r.File {
		name := strings.ToLower(filepath.ToSlash(filepath.Clean(f.Name)))
		if present[name] {
			return "", backupMeta{}, fmt.Errorf("duplicate backup entry: %s", f.Name)
		}
		present[name] = true
	}
	for _, want := range requiredZipEntries {
		if !present[want] {
			return "", backupMeta{}, fmt.Errorf("unsupported backup format: required MySQL entry %q is missing", want)
		}
	}

	stagingDir, err = os.MkdirTemp(dataDir, ".leafdoc-restore-*")
	if err != nil {
		return "", backupMeta{}, fmt.Errorf("failed to create staging directory: %w", err)
	}

	budget := shared.NewSizeBudget(limits.MaxTotalBytes)
	for _, f := range r.File {
		if err := extractZipEntry(f, stagingDir, limits, budget); err != nil {
			_ = os.RemoveAll(stagingDir)
			return "", backupMeta{}, fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
	}

	metaBytes, err := os.ReadFile(filepath.Join(stagingDir, "backup-meta.json"))
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", backupMeta{}, fmt.Errorf("failed to read backup-meta.json: %w", err)
	}
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", backupMeta{}, fmt.Errorf("failed to parse backup-meta.json: %w", err)
	}

	if err := validateArchive(stagingDir); err != nil {
		return "", backupMeta{}, errors.Join(err, os.RemoveAll(stagingDir))
	}
	for _, name := range swapNames {
		if err := os.MkdirAll(filepath.Join(stagingDir, name), 0o755); err != nil {
			return "", backupMeta{}, errors.Join(err, os.RemoveAll(stagingDir))
		}
	}

	return stagingDir, meta, nil
}

// extractZipEntry writes a single zip entry under destDir, rejecting any
// entry whose path would escape destDir (zip slip).
func extractZipEntry(f *zip.File, destDir string, limits shared.ExtractionLimits, budget *shared.SizeBudget) error {
	cleanName := filepath.Clean(f.Name)
	if cleanName == "." || strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
		return fmt.Errorf("unsafe entry path %q", f.Name)
	}
	destPath := filepath.Join(destDir, cleanName)
	rel, err := filepath.Rel(destDir, destPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("unsafe entry path %q", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if err := shared.CopyWithBudget(out, rc, f.CompressedSize64, limits, budget); err != nil {
		return err
	}
	return nil
}

// swapNames contains only file resources; business data is restored transactionally.
var swapNames = []string{"assets", "branding", "avatars", "revision-assets"}

// swapItem is one live path <-> staged path pair.
type swapItem struct {
	name       string
	livePath   string
	stagedPath string
	preRestore string
	// movedAside is true once SwapAll has passed the move-aside step for this
	// item — i.e. either the live path existed and was successfully renamed
	// to preRestore, or it didn't exist at all (nothing to move aside).
	// Tracked independently of swapped so that if the *second* rename
	// (staged -> live) then fails, RollbackAll still knows to check for and
	// restore a preRestore copy instead of silently leaving the item missing.
	movedAside bool
	// swapped is true once this item's staged replacement was fully moved
	// into place (implies movedAside; only items present in the staging dir
	// get swapped at all).
	swapped bool
}

// swapper drives the rename-aside/rename-in dance for every restorable item,
// on both POSIX and Windows (os.Rename isn't atomic-over-an-existing-target
// on Windows, so this two-step sequence is used uniformly rather than having
// a separate Windows code path).
type swapper struct {
	items []*swapItem
	// rename permits deterministic filesystem failure testing without permission assumptions.
	rename func(string, string) error
}

// newSwapper prepares the live<->staged path pairs. Items not present in
// stagingDir (e.g. an empty assets/ dir at snapshot time, or an older
// snapshot taken before branding.json was captured) are left untouched by
// SwapAll rather than cleared — restore only ever brings back what the
// snapshot actually captured, it never deletes live content the snapshot
// simply didn't record.
func newSwapper(dataDir, stagingDir string) *swapper {
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	items := make([]*swapItem, 0, len(swapNames))
	for _, name := range swapNames {
		live := filepath.Join(dataDir, name)
		if name == "revision-assets" {
			live = filepath.Join(dataDir, ".leafdoc", "blobs", "assets")
		}
		items = append(items, &swapItem{
			name:       name,
			livePath:   live,
			stagedPath: filepath.Join(stagingDir, name),
			preRestore: live + ".pre-restore-" + stamp,
		})
	}
	return &swapper{items: items}
}

// SwapAll performs rename-aside then rename-in for every item present in the
// staging dir, stopping at the first failure. It does not roll back on
// failure itself — the caller decides how to handle a partial swap (see
// Manager.rollbackOrIntervene), since the same rollback path is shared with
// failures from later phases (auth reopen, branding reload).
func (sw *swapper) SwapAll() error {
	rename := sw.rename
	if rename == nil {
		rename = os.Rename
	}
	for _, item := range sw.items {
		if _, err := os.Stat(item.stagedPath); err != nil {
			continue // not captured by this snapshot; leave the live item alone
		}
		if err := os.MkdirAll(filepath.Dir(item.livePath), 0o755); err != nil {
			return fmt.Errorf("prepare restored resource directory: %w", err)
		}

		if _, err := os.Stat(item.livePath); err == nil {
			if err := rename(item.livePath, item.preRestore); err != nil {
				return fmt.Errorf("failed to move aside %s: %w", item.name, err)
			}
		}
		item.movedAside = true

		if err := rename(item.stagedPath, item.livePath); err != nil {
			return fmt.Errorf("failed to move in restored %s: %w", item.name, err)
		}
		item.swapped = true
	}
	return nil
}

// RollbackAll reverses every item SwapAll touched: removes any restored
// content currently at the live path and moves each .pre-restore-* copy back
// into place. Driven by movedAside, not swapped — an item whose move-aside
// succeeded but whose move-in then failed (SwapAll's second rename) has no
// live content of its own at all right now, but still has a pre-restore copy
// that must be put back; checking only swapped would skip it and leave the
// item permanently missing. Best-effort — accumulates every error rather than
// stopping at the first, so every item still gets attempted even if one fails.
func (sw *swapper) RollbackAll() error {
	var errs []error
	for _, item := range sw.items {
		// swapped always implies movedAside (SwapAll only ever sets swapped
		// after movedAside, in the same iteration), so movedAside alone is
		// the correct "did SwapAll touch this item at all" check.
		if !item.movedAside {
			continue
		}
		// Safe no-op if the move-in step never ran (nothing to remove).
		if err := os.RemoveAll(item.livePath); err != nil {
			errs = append(errs, fmt.Errorf("%s: failed to remove restored content: %w", item.name, err))
			continue
		}
		if _, statErr := os.Stat(item.preRestore); statErr != nil {
			// Nothing was renamed aside (the live item didn't exist before
			// the swap) — nothing to restore, the item is now correctly absent.
			item.movedAside = false
			item.swapped = false
			continue
		}
		if err := os.Rename(item.preRestore, item.livePath); err != nil {
			errs = append(errs, fmt.Errorf("%s: failed to restore pre-restore copy: %w", item.name, err))
			continue
		}
		item.movedAside = false
		item.swapped = false
	}
	return errors.Join(errs...)
}

// CommitAll deletes every .pre-restore-* backup copy. Only call once the
// entire restore sequence (swap + auth reopen + branding reload) has
// succeeded — see Manager.runLocked.
func (sw *swapper) CommitAll() {
	for _, item := range sw.items {
		if !item.swapped {
			continue
		}
		if err := os.RemoveAll(item.preRestore); err != nil {
			slog.Default().Warn("restore: failed to clean up pre-restore backup copy", "item", item.name, "path", item.preRestore, "error", err)
		}
	}
}

// validateArchive checks all extracted bytes before any live resource changes.
func validateArchive(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "checksums.json"))
	if err != nil {
		return err
	}
	checksums := map[string]string{}
	if err := json.Unmarshal(raw, &checksums); err != nil {
		return err
	}
	for _, required := range []string{"database.json", "backup-meta.json"} {
		if checksums[required] == "" {
			return fmt.Errorf("missing required checksum")
		}
	}
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if name == "checksums.json" {
			return nil
		}
		expected, ok := checksums[name]
		if !ok {
			return fmt.Errorf("unverified backup entry: %s", name)
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(bytes)
		if hex.EncodeToString(sum[:]) != expected {
			return fmt.Errorf("backup checksum mismatch: %s", name)
		}
		delete(checksums, name)
		return nil
	})
	if err != nil {
		return err
	}
	if len(checksums) != 0 {
		return fmt.Errorf("backup resource is missing")
	}
	raw, err = os.ReadFile(filepath.Join(dir, "database.json"))
	if err != nil {
		return err
	}
	_, err = database.ValidateDump(raw)
	return err
}
