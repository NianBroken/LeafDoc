package shared

import (
	"fmt"
	"io"
	"log/slog"

	"os"
	"path/filepath"
	"strings"
)

const errFailedToCloseTempFile = "failed to close temp file"

func atomicWriteDir(filename string) string {
	normalized := strings.ReplaceAll(filename, `\`, `/`)
	return filepath.Dir(filepath.FromSlash(normalized))
}

// WriteFileAtomic writes data to filename atomically by writing to a temp file
// in the same directory and then renaming it over the target.
func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir := atomicWriteDir(filename)

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpName := tmpFile.Name()
	// Ensure the temp file is removed in case of an error
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if perm != 0 {
		if err := tmpFile.Chmod(perm); err != nil {
			chmodErr := fmt.Errorf("chmod temp file: %w", err)
			if closeErr := tmpFile.Close(); closeErr != nil {
				slog.Default().Error(errFailedToCloseTempFile, "operation", "chmod", "error", closeErr)
			}
			return chmodErr
		}
	}

	if _, err := tmpFile.Write(data); err != nil {
		writeErr := fmt.Errorf("write temp file: %w", err)
		if closeErr := tmpFile.Close(); closeErr != nil {
			slog.Default().Error(errFailedToCloseTempFile, "operation", "write", "error", closeErr)
		}
		return writeErr
	}

	if err := tmpFile.Sync(); err != nil {
		syncErr := fmt.Errorf("sync temp file: %w", err)
		if closeErr := tmpFile.Close(); closeErr != nil {
			slog.Default().Error(errFailedToCloseTempFile, "operation", "sync", "error", closeErr)
		}
		return syncErr
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := atomicReplace(tmpName, filename); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func WriteStreamAtomic(targetPath string, src io.Reader, maxBytes int64, perm os.FileMode) error {
	dir := atomicWriteDir(targetPath)

	out, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := out.Name()

	// Ensure cleanup on failure
	ok := false
	defer func() {
		if out == nil {
			if !ok {
				_ = os.Remove(tmp)
			}
			return
		}

		if err := out.Close(); err != nil {
			slog.Default().Error("Failed to close temp file", "file", tmp, "error", err)
			return
		}
		if !ok {
			_ = os.Remove(tmp)
		}
	}()

	if err := CopyWithLimit(out, src, maxBytes); err != nil {
		return err
	}

	// Best-effort durability
	if err := out.Sync(); err != nil {
		return err
	}

	// Chmod after writing so partial content is never world-readable.
	if perm != 0 {
		if err := out.Chmod(perm); err != nil {
			return fmt.Errorf("chmod temp file: %w", err)
		}
	}

	if err := out.Close(); err != nil {
		return err
	}
	out = nil

	if err := atomicReplace(tmp, targetPath); err != nil {
		return err
	}

	ok = true
	return nil
}
