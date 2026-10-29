package snapshot

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// Config supplies filesystem resources and a shared write barrier for MySQL snapshots.
type Config struct {
	BackupsDir     string
	DataDir        string
	AssetsDir      string
	BrandingDir    string
	AvatarsDir     string
	WikiVersion    string
	Interval       time.Duration
	RetentionCount int
	Quiesce        func() (func(), error)
}

type backupMeta struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Version   string    `json:"version"`
}

// createSnapshot builds the ZIP and sidecar JSON, returns the snapshot ID.
func createSnapshot(ctx context.Context, cfg Config) (string, error) {
	if err := os.MkdirAll(cfg.BackupsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backups directory: %w", err)
	}

	id, err := uniqueSnapshotID(cfg.BackupsDir, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("failed to allocate snapshot id: %w", err)
	}

	database.MaintenanceMu.Lock()
	defer database.MaintenanceMu.Unlock()
	if cfg.Quiesce != nil {
		release, err := cfg.Quiesce()
		if err != nil {
			return "", err
		}
		defer release()
	}
	databaseBytes, err := database.Export(ctx, cfg.DataDir)
	if err != nil {
		return "", err
	}
	createdAt := time.Now().UTC()
	zipPath := filepath.Join(cfg.BackupsDir, id+".zip")
	if err := writeSnapshotZip(zipPath, cfg, id, createdAt, databaseBytes); err != nil {
		_ = os.Remove(zipPath) // best-effort cleanup of a partial/invalid zip left by the failed write
		return "", fmt.Errorf("failed to write snapshot zip: %w", err)
	}

	info, err := os.Stat(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat snapshot zip: %w", err)
	}

	entry := SnapshotEntry{ID: id, CreatedAt: createdAt, SizeBytes: info.Size()}
	sidecarPath := filepath.Join(cfg.BackupsDir, id+".json")
	if err := writeJSONFile(sidecarPath, entry); err != nil {
		return "", fmt.Errorf("failed to write snapshot sidecar: %w", err)
	}

	return id, nil
}

// uniqueSnapshotID returns a snapshot ID based on t that has no existing
// .zip or .json file in backupsDir yet. IDs are second-precision timestamps,
// so two runs completing within the same second would otherwise collide and
// silently overwrite each other's snapshot; this appends a "-2", "-3", ...
// suffix until a free slot is found.
func uniqueSnapshotID(backupsDir string, t time.Time) (string, error) {
	base := "snapshot-" + t.Format("20060102-150405")
	id := base
	for i := 2; ; i++ {
		_, zipErr := os.Stat(filepath.Join(backupsDir, id+".zip"))
		_, jsonErr := os.Stat(filepath.Join(backupsDir, id+".json"))
		if os.IsNotExist(zipErr) && os.IsNotExist(jsonErr) {
			return id, nil
		}
		if zipErr != nil && !os.IsNotExist(zipErr) {
			return "", zipErr
		}
		if jsonErr != nil && !os.IsNotExist(jsonErr) {
			return "", jsonErr
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
}

// writeSnapshotZip combines a database snapshot and checked resource files.
func writeSnapshotZip(zipPath string, cfg Config, id string, createdAt time.Time, databaseBytes []byte) error {
	file, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(file)
	checksums := map[string]string{}
	defer file.Close()
	defer writer.Close()
	if err := addBytesToZip(writer, "database.json", databaseBytes, checksums); err != nil {
		return err
	}
	directories := map[string]string{"assets": cfg.AssetsDir, "branding": cfg.BrandingDir, "avatars": cfg.AvatarsDir, "revision-assets": filepath.Join(cfg.DataDir, ".leafdoc", "blobs", "assets")}
	for name, directory := range directories {
		if err := addDirToZip(writer, directory, name+"/", checksums); err != nil {
			return err
		}
	}
	meta, err := json.Marshal(backupMeta{ID: id, CreatedAt: createdAt, Version: cfg.WikiVersion})
	if err != nil {
		return err
	}
	if err := addBytesToZip(writer, "backup-meta.json", meta, checksums); err != nil {
		return err
	}
	manifest, err := json.Marshal(checksums)
	if err != nil {
		return err
	}
	if err := addBytesToZip(writer, "checksums.json", manifest); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

// addDirToZip walks srcDir and adds every file under it to the zip, prefixed
// with prefix. A missing srcDir (e.g. optional BrandingDir) is not an error.
func addDirToZip(w *zip.Writer, srcDir, prefix string, checksums ...map[string]string) error {
	if srcDir == "" {
		return nil
	}
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat %s: %w", srcDir, err)
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}
		return addFileToZip(w, path, prefix+filepath.ToSlash(rel), checksums...)
	})
}

// addFileToZip adds a single file to the zip under name. A missing srcFile
// (e.g. optional SchemaFile) is not an error.
func addFileToZip(w *zip.Writer, srcFile, name string, checksums ...map[string]string) error {
	if srcFile == "" {
		return nil
	}
	f, err := os.Open(srcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open %s: %w", srcFile, err)
	}
	defer func() { _ = f.Close() }()

	zw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", name, err)
	}
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(zw, hasher), f); err != nil {
		return fmt.Errorf("failed to write zip entry %s: %w", name, err)
	}
	if len(checksums) > 0 {
		checksums[0][name] = hex.EncodeToString(hasher.Sum(nil))
	}
	return nil
}

func addBytesToZip(w *zip.Writer, name string, data []byte, checksums ...map[string]string) error {
	zw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", name, err)
	}
	if _, err := zw.Write(data); err != nil {
		return fmt.Errorf("failed to write zip entry %s: %w", name, err)
	}
	if len(checksums) > 0 {
		sum := sha256.Sum256(data)
		checksums[0][name] = hex.EncodeToString(sum[:])
	}
	return nil
}

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
