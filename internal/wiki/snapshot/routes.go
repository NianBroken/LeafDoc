package wikisnapshot

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	coreauth "github.com/perber/wiki/internal/core/auth"
	snapshotSvc "github.com/perber/wiki/internal/core/snapshot"
	httpinternal "github.com/perber/wiki/internal/http"
	authmw "github.com/perber/wiki/internal/http/middleware/auth"
)

// Routes is the RouteRegistrar for the full-backup (snapshot) admin endpoints.
type Routes struct {
	manager        *snapshotSvc.Manager
	scheduler      *snapshotSvc.Scheduler
	authService    *coreauth.AuthService
	retentionCount int
	settings       *snapshotSvc.SettingsStore
}

// NewRoutes constructs the snapshot RouteRegistrar.
func NewRoutes(manager *snapshotSvc.Manager, scheduler *snapshotSvc.Scheduler, authService *coreauth.AuthService, retentionCount int, stores ...*snapshotSvc.SettingsStore) *Routes {
	var store *snapshotSvc.SettingsStore
	if len(stores) > 0 {
		store = stores[0]
	}
	return &Routes{
		manager:        manager,
		scheduler:      scheduler,
		authService:    authService,
		retentionCount: retentionCount,
		settings:       store,
	}
}

// RegisterRoutes implements RouteRegistrar.
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	opts := ctx.Opts

	authGroup := ctx.APIAuthGroup(r.authService)

	adminGroup := authGroup.Group("/admin")
	adminGroup.Use(authmw.RequireAdmin(opts.AuthDisabled))

	adminGroup.GET("/snapshot/status", r.handleStatus)
	adminGroup.PUT("/snapshot/settings", r.handleUpdateSettings)
	adminGroup.GET("/snapshot", r.handleList)
	adminGroup.POST("/snapshot", r.handleTrigger)
	adminGroup.GET("/snapshot/:id/download", authmw.RequireCookieSession(), r.handleDownload)
	adminGroup.DELETE("/snapshot/:id", r.handleDelete)
}

// handleStatus returns whether the feature is enabled, the configured
// retention count, and the current run status.
func (r *Routes) handleStatus(c *gin.Context) {
	if r.manager == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":        true,
		"settings":       r.currentSettings(),
		"retentionCount": r.currentSettings().RetentionCount,
		"status":         r.manager.Status(),
	})
}

func (r *Routes) currentSettings() snapshotSvc.Settings {
	if r.settings != nil {
		return r.settings.Get()
	}
	value := r.manager.Settings()
	value.RetentionCount = r.retentionCount
	return value
}

func (r *Routes) handleUpdateSettings(c *gin.Context) {
	if r.manager == nil || r.settings == nil {
		r.respondNotEnabled(c)
		return
	}
	var value snapshotSvc.Settings
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot settings"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot settings"})
		return
	}
	if err := snapshotSvc.ValidateSettings(value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := r.settings.Put(value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save snapshot settings"})
		return
	}
	r.manager.UpdateSettings(value)
	if r.scheduler != nil {
		r.scheduler.UpdateSettings()
	}
	c.JSON(http.StatusOK, gin.H{"settings": value})
}

// respondNotEnabled writes the standard 503 response for handlers that need
// a non-nil manager/scheduler, which is only nil if snapshots are disabled.
func (r *Routes) respondNotEnabled(c *gin.Context) {
	respondWithSnapshotStatusError(c, http.StatusServiceUnavailable, ErrCodeSnapshotNotEnabled, "Snapshot backup is not enabled", "snapshot backup not enabled")
}

// handleList returns all finished snapshots, newest first.
func (r *Routes) handleList(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	entries, err := r.manager.List()
	if err != nil {
		respondWithSnapshotError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": entries})
}

// handleTrigger triggers an immediate snapshot and returns 202 Accepted.
func (r *Routes) handleTrigger(c *gin.Context) {
	if r.scheduler == nil {
		r.respondNotEnabled(c)
		return
	}
	if !r.scheduler.TriggerNow() {
		respondWithSnapshotError(c, snapshotSvc.ErrAlreadyRunning)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"triggered": true})
}

// handleDownload streams the ZIP file for a given snapshot id as an attachment.
func (r *Routes) handleDownload(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	id := c.Param("id")
	zipPath, err := r.manager.SnapshotZipPath(id)
	if err != nil {
		respondWithSnapshotError(c, err)
		return
	}

	f, err := os.Open(zipPath)
	if err != nil {
		respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Failed to open snapshot", "failed to open snapshot")
		return
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Failed to read snapshot", "failed to read snapshot")
		return
	}

	filename := filepath.Base(zipPath)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if disposition == "" {
		disposition = "attachment"
	}
	c.Header("Content-Disposition", disposition)
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Writer.Header().Set("Content-Type", "application/zip")
	http.ServeContent(c.Writer, c.Request, filename, stat.ModTime(), f)
}

// handleDelete removes a snapshot's ZIP and sidecar metadata.
func (r *Routes) handleDelete(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	id := c.Param("id")
	if err := r.manager.Delete(id); err != nil {
		respondWithSnapshotError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
