package restore

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	coreshared "github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// ErrRestoreAlreadyRunning is returned by TriggerRestore when a restore is
// already in progress. Fixed package-level LocalizedError, matching
// wikiresync.ErrResyncAlreadyRunning / snapshot.ErrAlreadyRunning.
var ErrRestoreAlreadyRunning = sharederrors.NewLocalizedError(
	"restore_already_running",
	"A restore is already in progress",
	"a restore is already in progress",
	nil,
)

// ErrRestoreNeedsIntervention is returned by TriggerRestore when a previous
// restore left the instance in a NeedsIntervention state. Starting a new
// restore on top of a half-swapped filesystem and orphaned .pre-restore-*
// copies would compound the inconsistency instead of resolving it — the only
// supported way out is self-restart (see Manager.SelfRestart / ADR-0009).
var ErrRestoreNeedsIntervention = sharederrors.NewLocalizedError(
	"restore_needs_intervention",
	"This instance needs attention before a new restore can be started — restart the server first",
	"a previous restore needs intervention before a new restore can be started",
	nil,
)

// ErrWritesDisabled is returned by the write-gate HTTP middleware
// (internal/http/middleware/maintenance) when a mutating request arrives
// while a restore is swapping files. Defined here, not hand-rolled in the
// middleware, so that response follows the same *sharederrors.LocalizedError
// convention as every other error this feature returns.
var ErrWritesDisabled = sharederrors.NewLocalizedError(
	"restore_writes_disabled",
	"A restore is in progress; writes are temporarily disabled",
	"a restore is in progress; writes are temporarily disabled",
	nil,
)

// gateDrainTimeout bounds how long the restore sequence waits, once the
// write gate is engaged, for requests already in flight (started just before
// Engage()) to finish before files are swapped out from under them. A
// timeout here is logged and the restore proceeds anyway rather than failing
// the whole operation over a slow request.
const gateDrainTimeout = 10 * time.Second

// progressTokenTTL bounds the capability to a single restore observation window.
const progressTokenTTL = time.Hour

// DefaultMaxUploadSizeBytes is used by TriggerRestoreFromUpload's HTTP
// handler (POST /restore/upload) when Config.MaxUploadSizeBytes is unset.
const DefaultMaxUploadSizeBytes int64 = 500 * 1024 * 1024 // 500 MiB

type Manager struct {
	progressMu        sync.Mutex
	progressTokenHash [sha256.Size]byte
	hasProgressToken  bool
	progressExpiresAt time.Time
	cfg               Config
	job               *Job
	// wg tracks the in-flight runLocked goroutine (if any), so callers that
	// need the process to shut down cleanly (main.go) can wait for a restore
	// to finish before closing the services it depends on (AuthService,
	// BrandingService) out from under it.
	wg sync.WaitGroup
}

func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg, job: NewJob()}
}

// TriggerRestore starts a restore job asynchronously for the given snapshot
// id. Returns ErrRestoreAlreadyRunning if a restore is already in progress,
// or ErrRestoreNeedsIntervention if a previous restore left the instance in
// a state where a new one must not be started (see ErrRestoreNeedsIntervention).
func (m *Manager) TriggerRestore(id string) error {
	_, err := m.TriggerRestoreWithStatusToken(id)
	return err
}

// TriggerRestoreWithStatusToken grants read-only access to this job's progress.
func (m *Manager) TriggerRestoreWithStatusToken(id string) (string, error) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	token, err := m.startProgressJob()
	if err != nil {
		return "", err
	}
	m.wg.Go(func() { m.runLocked(id) })
	return token, nil
}

// startProgressJob runs under progressMu so token rotation and job start are atomic.
func (m *Manager) startProgressJob() (string, error) {
	if m.job.Status().NeedsIntervention {
		return "", ErrRestoreNeedsIntervention
	}
	if m.job.Status().Running {
		return "", ErrRestoreAlreadyRunning
	}
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate restore progress credential: %w", err)
	}
	if !m.job.Start() {
		return "", ErrRestoreAlreadyRunning
	}
	token := hex.EncodeToString(random[:])
	m.progressTokenHash = sha256.Sum256([]byte(token))
	m.hasProgressToken = true
	m.progressExpiresAt = time.Now().Add(progressTokenTTL)
	return token, nil
}

// ProgressStatus accepts only the current job's credential. Fixed-size hash
// comparison is constant time; neither credential nor hash enters job JSON.
func (m *Manager) ProgressStatus(token string) (JobStatus, bool) {
	candidate := sha256.Sum256([]byte(token))
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	matches := subtle.ConstantTimeCompare(candidate[:], m.progressTokenHash[:])
	if !m.hasProgressToken || matches != 1 || !time.Now().Before(m.progressExpiresAt) {
		return JobStatus{}, false
	}
	return m.job.Status(), true
}

// TriggerRestoreFromUpload starts a restore job asynchronously from an
// uploaded ZIP, read in full from r. Returns the same errors as
// TriggerRestore (ErrRestoreAlreadyRunning / ErrRestoreNeedsIntervention),
// plus any error encountered while staging the upload to disk.
//
// Unlike TriggerRestore(id), staging happens synchronously before this
// method returns: r is expected to be backed by the inbound HTTP request's
// multipart body, which is only valid for the lifetime of that request — it
// must be fully drained into a durable temp file before control returns to
// the HTTP handler, since the actual restore then runs in a background
// goroutine after the handler (and its response) has already completed.
func (m *Manager) TriggerRestoreFromUpload(r io.Reader) error {
	_, err := m.TriggerRestoreFromUploadWithStatusToken(r)
	return err
}

// TriggerRestoreFromUploadWithStatusToken stages the request before granting
// a credential, and retains that credential after sessions are invalidated.
func (m *Manager) TriggerRestoreFromUploadWithStatusToken(r io.Reader) (string, error) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	token, err := m.startProgressJob()
	if err != nil {
		return "", err
	}
	zipPath, cleanup, err := m.stageUpload(r)
	if err != nil {
		m.job.Finish(err)
		return "", err
	}
	m.wg.Go(func() {
		m.runGuarded(func() { defer cleanup(); m.runFromZipPath(zipPath, coreshared.DefaultZipExtractionLimits, cleanup) })
	})
	return token, nil
}

// stageUpload drains r into a fresh temp file inside dataDir (the same
// filesystem extractAndValidate's staging directory uses, and for the same
// reason — so nothing downstream needs a cross-filesystem copy). The
// returned cleanup func removes the temp file; callers must invoke it
// exactly once, whether or not the restore that follows succeeds.
func (m *Manager) stageUpload(r io.Reader) (zipPath string, cleanup func(), err error) {
	f, err := os.CreateTemp(m.cfg.DataDir, ".leafdoc-restore-upload-*.zip")
	if err != nil {
		return "", nil, errStageUploadFailed(err)
	}
	var release sync.Once
	cleanup = func() { release.Do(func() { _ = os.Remove(f.Name()) }) }

	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, errStageUploadFailed(err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, errStageUploadFailed(err)
	}
	return f.Name(), cleanup, nil
}

// errStageUploadFailed wraps any failure to drain an uploaded backup ZIP to a
// durable temp file (disk full, permission error, ...) as a LocalizedError,
// per this repo's convention that domain errors reaching an HTTP handler must
// be *sharederrors.LocalizedError, not a bare fmt.Errorf/errors.New. Unlike
// this file's other fmt.Errorf calls (which only ever feed the async
// job-status string), stageUpload's errors are returned synchronously to
// TriggerRestoreFromUpload's HTTP handler, which only renders a specific
// message for LocalizedError — anything else falls back to a generic
// "Restore request failed".
func errStageUploadFailed(cause error) error {
	return sharederrors.NewLocalizedError(
		"restore_upload_staging_failed",
		"Failed to save the uploaded backup to disk",
		"failed to stage uploaded backup: %s",
		cause,
		cause.Error(),
	)
}

// Status returns the current restore job state (thread-safe).
func (m *Manager) Status() JobStatus {
	return m.job.Status()
}

// MaxUploadSizeBytes returns the configured limit for POST /restore/upload,
// falling back to DefaultMaxUploadSizeBytes when Config.MaxUploadSizeBytes is
// unset (<= 0).
func (m *Manager) MaxUploadSizeBytes() int64 {
	if m.cfg.MaxUploadSizeBytes > 0 {
		return m.cfg.MaxUploadSizeBytes
	}
	return DefaultMaxUploadSizeBytes
}

// Wait blocks until any in-flight restore triggered via TriggerRestore has
// fully finished. Intended to be called during process shutdown, before
// closing services (AuthService, BrandingService) a running restore depends
// on — see cmd/leafdoc/main.go.
func (m *Manager) Wait() {
	m.wg.Wait()
}

// SelfRestart re-execs the current process. Callers (the HTTP handler) are
// expected to only allow this once Status().NeedsIntervention is true.
func (m *Manager) SelfRestart() error {
	return SelfRestart()
}

// runLocked resolves the given snapshot id to a zip path and runs the
// restore sequence against it (see runFromZipPath). The snapshot itself was
// produced by this server's own snapshot machinery, not attacker-controlled
// input, so extraction runs unrestricted rather than under
// DefaultZipExtractionLimits — a legitimately large wiki must still be able
// to restore its own backup.
func (m *Manager) runLocked(id string) {
	m.runGuarded(func() {
		zipPath, err := m.cfg.SnapshotManager.SnapshotZipPath(id)
		if err != nil {
			m.job.SetPhase(PhaseValidating)
			m.job.Finish(err)
			return
		}
		m.runFromZipPath(zipPath, coreshared.UnrestrictedExtractionLimits, nil)
	})
}

// runGuarded recovers from any panic raised by fn, treating it as
// NeedsIntervention: a panic mid-sequence means we don't know which phases
// actually completed, so failing safe (gate stays engaged, admin must
// self-restart) is the only sound response. Shared by both the by-id
// (runLocked) and by-upload (TriggerRestoreFromUpload) entrypoints.
func (m *Manager) runGuarded(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("panic during restore", "panic", r)
			m.job.FinishNeedsIntervention(fmt.Errorf("panic during restore: %v", r))
		}
	}()
	fn()
}

// runFromZipPath validates the archive before the shared maintenance barrier.
func (m *Manager) runFromZipPath(zipPath string, limits coreshared.ExtractionLimits, releaseUpload func()) {
	m.job.SetPhase(PhaseValidating)
	staging, _, err := extractAndValidateWithLimits(zipPath, m.cfg.DataDir, limits)
	if releaseUpload != nil {
		releaseUpload()
	}
	if err != nil {
		m.job.Finish(err)
		return
	}
	defer func() {
		if err := os.RemoveAll(staging); err != nil {
			m.job.recordCleanupError(err)
		}
	}()
	database.MaintenanceMu.Lock()
	defer database.MaintenanceMu.Unlock()
	m.cfg.WriteGate.Engage()
	if !m.cfg.WriteGate.WaitForDrain(gateDrainTimeout) {
		m.cfg.WriteGate.Disengage()
		m.job.Finish(fmt.Errorf("maintenance drain timed out"))
		return
	}
	m.job.SetPhase(PhaseSwapping)
	if err := restoreStaged(m.cfg.DataDir, staging); err != nil {
		var intervention *interventionError
		if errors.As(err, &intervention) {
			m.job.FinishNeedsIntervention(err)
			return
		}
		m.cfg.WriteGate.Disengage()
		m.job.Finish(err)
		return
	}
	if err := m.reloadAll(); err != nil {
		m.job.FinishNeedsIntervention(err)
		return
	}
	if m.cfg.UserResolver != nil {
		if err := m.cfg.UserResolver.Reload(); err != nil {
			m.job.FinishNeedsIntervention(err)
			return
		}
	}
	if m.cfg.ReloadRuntime != nil {
		if err := m.cfg.ReloadRuntime(); err != nil {
			m.job.FinishNeedsIntervention(err)
			return
		}
	} else if m.cfg.TriggerResync != nil {
		m.cfg.TriggerResync()
	}
	m.cfg.WriteGate.Disengage()
	m.job.Finish(nil)
}

// reloadAll re-reads every configured settings.Reloadable (nil-safe, one per
// entry) from the restored files, accumulating every failure rather than
// stopping at the first — mirroring reopenAllStores' error-accumulation
// style below.
func (m *Manager) reloadAll() error {
	var errs []error
	for _, r := range m.cfg.Reloadables {
		if r == nil {
			continue
		}
		if err := r.Reload(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
