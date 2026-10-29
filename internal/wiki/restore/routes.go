package wikirestore

import (
	"errors"
	"log/slog"
	"net/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/restore"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// uploadParseMaxMemory bounds how much of an uploaded restore ZIP
// ParseMultipartForm is allowed to buffer in memory before mime/multipart
// spills the rest to its own temp file on disk. Deliberately independent of
// the (much larger) MaxUploadSizeBytes limit: reusing that limit here would
// let ParseMultipartForm hold an entire near-max-size upload in RAM, which on
// a resource-constrained deployment (e.g. a Raspberry Pi, an explicit
// LeafDoc deployment target) risks OOM.
const uploadParseMaxMemory = 32 << 20 // 32 MiB, matching net/http's own default

// Routes is the RouteRegistrar for the live restore admin endpoints.
type Routes struct {
	manager     *restore.Manager
	authService *coreauth.AuthService
}

// NewRoutes constructs the restore RouteRegistrar.
func NewRoutes(manager *restore.Manager, authService *coreauth.AuthService) *Routes {
	return &Routes{
		manager:     manager,
		authService: authService,
	}
}

// RegisterRoutes implements RouteRegistrar.
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	opts := ctx.Opts

	// Progress uses its own job-scoped credential and never grants admin access.
	ctx.Base.GET("/api/restore/progress", r.handleProgress)

	authGroup := ctx.APIAuthGroup(r.authService)

	adminGroup := authGroup.Group("/admin")
	adminGroup.Use(authmw.RequireAdmin(opts.AuthDisabled))

	// A restore replaces MySQL account and API key records, so it is browser-session
	// only: an API key must not be able to swap in its own credentials.
	sessionOnly := authmw.RequireCookieSession()

	adminGroup.POST("/restore/upload", sessionOnly, r.handleTriggerUpload)
	adminGroup.POST("/restore/:id", sessionOnly, r.handleTrigger)
	adminGroup.GET("/restore/status", r.handleStatus)
	adminGroup.POST("/restore/self-restart", sessionOnly, r.handleSelfRestart)
}

// respondNotEnabled writes the standard 503 response for handlers that need
// a non-nil manager, which is only nil if snapshots (and therefore restore)
// are disabled.
func (r *Routes) respondNotEnabled(c *gin.Context) {
	respondWithRestoreStatusError(c, http.StatusServiceUnavailable, ErrCodeRestoreNotEnabled, "Restore is not enabled", "restore not enabled")
}

// handleTrigger starts a restore from snapshot :id and returns 202 Accepted.
func (r *Routes) handleTrigger(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	id := c.Param("id")
	statusToken, err := r.manager.TriggerRestoreWithStatusToken(id)
	if err != nil {
		respondWithRestoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "statusToken": statusToken})
}

// handleTriggerUpload starts a restore from an uploaded backup ZIP and
// returns 202 Accepted. Progress and any validation failure (e.g. a
// malformed or non-LeafDoc ZIP) surface asynchronously via handleStatus,
// exactly as they do for handleTrigger.
func (r *Routes) handleTriggerUpload(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}

	maxSize := r.manager.MaxUploadSizeBytes()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
	if err := c.Request.ParseMultipartForm(uploadParseMaxMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondWithRestoreStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeRestoreUploadTooLarge, "Upload exceeds the maximum allowed size", "upload exceeds the maximum allowed size")
			return
		}
		respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreUploadInvalid, "Failed to parse the uploaded file", "failed to parse the uploaded file")
		return
	}
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()

	fh, err := c.FormFile("file")
	if err != nil {
		respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreMissingFile, "Missing file", "missing file")
		return
	}
	file, err := fh.Open()
	if err != nil {
		respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreFileOpenFailed, "Failed to open uploaded file", "failed to open uploaded file")
		return
	}
	defer func() { _ = file.Close() }()

	statusToken, err := r.manager.TriggerRestoreFromUploadWithStatusToken(file)
	if err != nil {
		respondWithRestoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "statusToken": statusToken})
}

// handleStatus returns the current restore job state for client polling.
func (r *Routes) handleStatus(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	c.JSON(http.StatusOK, r.manager.Status())
}

// handleSelfRestart re-execs the server process. Only permitted once the
// restore job has reported NeedsIntervention — this is the documented
// recovery path out of a stuck restore, not a general-purpose restart button.
// The HTTP response may never actually reach the client (the process image
// is replaced/exits before the handler can finish writing) — that's expected,
// the frontend treats a dropped connection here as success.
func (r *Routes) handleSelfRestart(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	if !r.manager.Status().NeedsIntervention {
		respondWithRestoreStatusError(c, http.StatusConflict, ErrCodeRestoreNotIntervenable,
			"Self-restart is only available after a restore reports it needs intervention",
			"self-restart is only available after a restore reports it needs intervention")
		return
	}
	// A successful restart replaces or exits the process. Returning here means
	// the restart failed, so retain the cause in the server log.
	err := r.manager.SelfRestart()
	slog.Default().Error("restore: self-restart failed", "error", err)
	respondWithRestoreStatusError(c, http.StatusInternalServerError, ErrCodeRestoreInternalError, "Failed to restart server", "failed to restart server")
}

// handleProgress exposes only the current restore job to its bearer credential.
// Credentials travel in headers and responses carry no-store to prevent caching.
func (r *Routes) handleProgress(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	status, valid := r.manager.ProgressStatus(c.GetHeader("X-Restore-Token"))
	if !valid {
		respondWithRestoreStatusError(c, http.StatusUnauthorized, "restore_progress_unauthorized", "Invalid restore progress credential", "invalid restore progress credential")
		return
	}
	c.JSON(http.StatusOK, status)
}
