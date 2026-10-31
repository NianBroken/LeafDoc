// 本文件提供已有快照恢复、上传恢复、进度查询与介入重启的 HTTP 入口。
// 恢复执行交给管理器，接口层区分管理员身份和仅能查看当前任务的专用凭据。
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

// uploadParseMaxMemory 控制分段表单解析时文件内容的内存缓冲量，超过后写入临时文件。
// 它与整个请求体的上传限额分开设置，避免将接近上传限额的归档全部留在内存。
const uploadParseMaxMemory = 32 << 20 // 文件部分使用 32 MiB 缓冲，与 net/http 默认值相同。

// Routes 保存恢复管理器与身份服务，管理器为空时各处理入口返回 503。
type Routes struct {
	manager     *restore.Manager
	authService *coreauth.AuthService
}

// NewRoutes 绑定依赖，不启动恢复或替换运行数据。
func NewRoutes(manager *restore.Manager, authService *coreauth.AuthService) *Routes {
	return &Routes{
		manager:     manager,
		authService: authService,
	}
}

// RegisterRoutes 为管理操作安装认证和管理员限制，进度查询单独校验任务凭据。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {

	// 进度入口不要求账号会话，只允许专用凭据读取当前任务，不授予管理操作权限。
	ctx.Base.GET("/api/restore/progress", r.handleProgress)

	authGroup := ctx.APIAuthGroup(r.authService)

	adminGroup := authGroup.Group("/admin")
	adminGroup.Use(authmw.RequireAdmin())

	// 恢复会替换数据库中的账号和 API 密钥记录，写操作另行拒绝 API 密钥身份，
	// 避免凭密钥将其他备份中的账号凭据装入系统。
	sessionOnly := authmw.RequireCookieSession()

	adminGroup.POST("/restore/upload", sessionOnly, r.handleTriggerUpload)
	adminGroup.POST("/restore/:id", sessionOnly, r.handleTrigger)
	adminGroup.GET("/restore/status", r.handleStatus)
	adminGroup.POST("/restore/self-restart", sessionOnly, r.handleSelfRestart)
}

// respondNotEnabled 为缺失恢复管理器的操作输出统一的服务不可用错误。
func (r *Routes) respondNotEnabled(c *gin.Context) {
	respondWithRestoreStatusError(c, http.StatusServiceUnavailable, ErrCodeRestoreNotEnabled, "Restore is not enabled", "restore not enabled")
}

// handleTrigger 将路径中的快照标识交给管理器启动任务，返回 202 与专用进度凭据。
// 归档读取和内容验证在后台执行，接受请求不表示恢复已经成功。
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

// handleTriggerUpload 接收 file 表单项，同步解析上传并将文件暂存到数据目录。
// 暂存成功后由管理器后台校验归档与执行恢复，返回 202 和任务进度凭据。
// 表单解析、打开文件与暂存失败直接响应错误，归档内容失败通过任务状态报告。
func (r *Routes) handleTriggerUpload(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}

	// 限额覆盖整个请求体，包括表单边界和字段，并非仅限制 ZIP 文件字节。
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
	// 解析成功后登记清理表单临时文件，后续任一返回分支都会执行。
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
	// 管理器在返回前读完上传流，因此处理器返回时关闭句柄不影响后台恢复。
	defer func() { _ = file.Close() }()

	statusToken, err := r.manager.TriggerRestoreFromUploadWithStatusToken(file)
	if err != nil {
		respondWithRestoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "statusToken": statusToken})
}

// handleStatus 为管理员返回当前恢复快照，执行失败通过状态字段表示。
func (r *Routes) handleStatus(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	c.JSON(http.StatusOK, r.manager.Status())
}

// handleSelfRestart 只在任务标记需要介入时调用平台重启实现，其余状态响应 409。
// 成功重启会替换或退出当前进程，不走后续响应分支；调用返回时记录失败原因并响应 500。
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
	// 平台实现成功时不返回，返回的错误留在服务日志中供排查。
	err := r.manager.SelfRestart()
	slog.Default().Error("restore: self-restart failed", "error", err)
	respondWithRestoreStatusError(c, http.StatusInternalServerError, ErrCodeRestoreInternalError, "Failed to restart server", "failed to restart server")
}

// handleProgress 从 X-Restore-Token 请求头取得凭据，校验后仅返回当前任务状态。
// 无效或过期凭据响应 401，所有响应禁止缓存，凭据不放入 URL。
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
