// 本文件提供管理员快照配置、列表、手动排队、下载与删除接口。
// 归档生成由快照管理器和调度器执行，HTTP 层负责权限、输入与响应。
package wikisnapshot

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	snapshotSvc "github.com/NianBroken/LeafDoc/internal/core/snapshot"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes 保存归档管理器、串行调度器与可持久化设置。依赖为空时，
// 状态接口报告未启用，需要对应依赖的操作返回服务不可用。
type Routes struct {
	manager        *snapshotSvc.Manager
	scheduler      *snapshotSvc.Scheduler
	authService    *coreauth.AuthService
	retentionCount int
	settings       *snapshotSvc.SettingsStore
}

// NewRoutes 只使用可选设置参数的第一项。没有设置存储时仍可查询与操作归档，
// 但不能通过设置接口写入配置，保留数量取构造参数。
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

// RegisterRoutes 将接口置于已认证的管理员分组，下载入口另外拒绝 API 密钥身份。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {

	authGroup := ctx.APIAuthGroup(r.authService)

	adminGroup := authGroup.Group("/admin")
	adminGroup.Use(authmw.RequireAdmin())

	adminGroup.GET("/snapshot/status", r.handleStatus)
	adminGroup.PUT("/snapshot/settings", r.handleUpdateSettings)
	adminGroup.GET("/snapshot", r.handleList)
	adminGroup.POST("/snapshot", r.handleTrigger)
	adminGroup.GET("/snapshot/:id/download", authmw.RequireCookieSession(), r.handleDownload)
	adminGroup.DELETE("/snapshot/:id", r.handleDelete)
}

// handleStatus 用管理器是否存在表示功能可用性，而非周期调度的 Enabled 设置。
// 可用时返回设置、保留数量与执行状态，未启用也以 200 返回 enabled=false。
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

// currentSettings 优先读取设置存储的副本，没有存储时使用管理器配置，
// 并以路由构造参数覆盖保留数量。调用此方法前需确认管理器存在。
func (r *Routes) currentSettings() snapshotSvc.Settings {
	if r.settings != nil {
		return r.settings.Get()
	}
	value := r.manager.Settings()
	value.RetentionCount = r.retentionCount
	return value
}

// handleUpdateSettings 只接受一个完整 JSON 值，拒绝未知字段、尾随内容与无效设置。
// 持久化成功后才更新管理器并通知调度器重算时间，不在保存失败时改变运行设置。
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
	// 第二次解码必须直接到达结尾，避免接受两个相连对象或非法尾部。
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot settings"})
		return
	}
	if err := snapshotSvc.ValidateSettings(value); err != nil {
		if loc, ok := sharederrors.AsLocalizedError(err); ok {
			httpinternal.RespondError(c, http.StatusBadRequest, loc.Code, loc.Message, loc.Template, loc.Args...)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	if err := r.settings.Put(value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save snapshot settings"})
		return
	}
	// 通知只是登记重算事件，不取消当前正在生成的归档。
	r.manager.UpdateSettings(value)
	if r.scheduler != nil {
		r.scheduler.UpdateSettings()
	}
	c.JSON(http.StatusOK, gin.H{"settings": value})
}

// respondNotEnabled 在操作所需依赖缺失时返回统一的 503 错误。
func (r *Routes) respondNotEnabled(c *gin.Context) {
	respondWithSnapshotStatusError(c, http.StatusServiceUnavailable, ErrCodeSnapshotNotEnabled, "Snapshot backup is not enabled", "snapshot backup not enabled")
}

// handleList 返回管理器读取的元数据列表，按记录的创建时间降序排列。
// 列表读取不验证对应 ZIP 是否存在，任一元数据读取或解析失败都返回错误。
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

// handleTrigger 尝试向容量为一的手动事件通道排入任务，成功即返回 202。
// 已有待处理事件时返回 409；正在归档不等于通道已满，执行中仍能排队一次。
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

// handleDownload 校验快照标识并打开 ZIP，通过 ServeContent 发送附件。
// 文件句柄在处理结束时关闭，响应禁止缓存并使用磁盘文件的修改时间。
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

	// 只将路径末段作为下载名称，由标准库编码附件头，编码失败时保留附件类型。
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

// handleDelete 委托管理器删除 ZIP 和元数据。成功后返回 ok，
// 两项删除不构成事务，失败响应不代表此前删除的文件已被恢复。
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
