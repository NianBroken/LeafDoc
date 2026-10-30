package restore

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	coreshared "github.com/NianBroken/LeafDoc/internal/core/shared"
)

// interventionError keeps maintenance active when resource rollback is incomplete.
type interventionError struct{ cause error }

func (e *interventionError) Error() string { return e.cause.Error() }
func (e *interventionError) Unwrap() error { return e.cause }

func rollbackResources(sw *swapper, cause error) error {
	if rollbackErr := sw.RollbackAll(); rollbackErr != nil {
		return &interventionError{errors.Join(cause, rollbackErr)}
	}
	return cause
}

// RestoreOffline uses the same validation, database transaction and resource swap as live restore.
func RestoreOffline(dataDir, zipPath string) (resultErr error) {
	staging, _, err := extractAndValidateWithLimits(zipPath, dataDir, coreshared.UnrestrictedExtractionLimits)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, os.RemoveAll(staging)) }()
	database.MaintenanceMu.Lock()
	defer database.MaintenanceMu.Unlock()
	return restoreStaged(dataDir, staging)
}

// restoreStaged commits MySQL only after all resource moves succeed.
func restoreStaged(dataDir, staging string) (resultErr error) {
	raw, err := os.ReadFile(filepath.Join(staging, "database.json"))
	if err != nil {
		return err
	}
	db, err := database.Open(dataDir)
	if err != nil {
		return err
	}
	defer database.CloseResource(db, &resultErr)
	tx, err := database.Import(context.Background(), db, raw)
	if err != nil {
		return err
	}
	defer database.Rollback(tx, &resultErr)
	sw := newSwapper(dataDir, staging)
	if err := sw.SwapAll(); err != nil {
		return rollbackResources(sw, err)
	}
	if err := tx.Commit(); err != nil {
		return rollbackResources(sw, err)
	}
	sw.CommitAll()
	return nil
}
