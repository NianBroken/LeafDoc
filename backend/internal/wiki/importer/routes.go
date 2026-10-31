package importer

import (
	"log/slog"
	"net/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	coreimporter "github.com/NianBroken/LeafDoc/internal/core/importer"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

const (
	// importMaxUploadSize 限制整个 ZIP 上传请求体，也作为 multipart 内存解析阈值。
	importMaxUploadSize = 500 << 20 // 500 MiB。
	// importPlanRoutePath 供创建、读取和清空当前计划的路由共用。
	importPlanRoutePath = "/import/plan"
)

// Routes 将当前导入计划的创建、轮询、启动及取消清理接入 HTTP。
type Routes struct {
	createPlan  *CreateImportPlanUseCase
	getPlan     *GetImportPlanUseCase
	execute     *ExecuteImportUseCase
	clearPlan   *ClearImportPlanUseCase
	authService *coreauth.AuthService
	svc         *coreimporter.ImporterService
	log         *slog.Logger
}

// RoutesConfig 保存用例及其共享服务，服务还用于接收路由层的资源上传限制。
type RoutesConfig struct {
	CreatePlan  *CreateImportPlanUseCase
	GetPlan     *GetImportPlanUseCase
	Execute     *ExecuteImportUseCase
	ClearPlan   *ClearImportPlanUseCase
	AuthService *coreauth.AuthService
	Svc         *coreimporter.ImporterService
	Log         *slog.Logger
}

// NewRoutes 保存依赖，不在构造时解压文件或启动导入。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		createPlan:  cfg.CreatePlan,
		getPlan:     cfg.GetPlan,
		execute:     cfg.Execute,
		clearPlan:   cfg.ClearPlan,
		authService: cfg.AuthService,
		svc:         cfg.Svc,
		log:         cfg.Log,
	}
}

// RegisterRoutes 为全部导入接口要求登录和编辑权限，读取当前计划也不采用公开读取豁免。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	opts := ctx.Opts

	// 正的路由资源限制传给导入服务，用于执行中的单个资源处理，不改变固定 ZIP 请求限制。
	if r.svc != nil && opts.MaxAssetUploadSizeBytes > 0 {
		r.svc.SetAssetMaxUploadSizeBytes(opts.MaxAssetUploadSizeBytes)
	}

	authGroup := ctx.APIAuthGroup(r.authService)

	authGroup.POST(importPlanRoutePath, authmw.RequireEditorOrAdmin(), r.handleCreatePlan)
	authGroup.GET(importPlanRoutePath, authmw.RequireEditorOrAdmin(), r.handleGetPlan)
	authGroup.POST("/import/execute", authmw.RequireEditorOrAdmin(), r.handleExecute)
	authGroup.DELETE(importPlanRoutePath, authmw.RequireEditorOrAdmin(), r.handleClearPlan)
}

// handleCreatePlan 验证操作者后解析 ZIP 文件与目标基础路径，创建计划但不执行内容导入。
// 表单解析错误统一映射为 413，文件缺失和打开失败分别映射为 400。
// 已打开的上传文件由本处理器关闭，关闭失败写错误日志。
func (r *Routes) handleCreatePlan(c *gin.Context) {
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, importMaxUploadSize)
	if err := c.Request.ParseMultipartForm(importMaxUploadSize); err != nil {
		respondWithImporterStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeImporterUploadTooLarge, "Upload exceeds maximum size limit of 500 MiB", "upload exceeds maximum size limit")
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		respondWithImporterStatusError(c, http.StatusBadRequest, ErrCodeImporterMissingFile, "Missing file", "missing file")
		return
	}
	file, err := fh.Open()
	if err != nil {
		respondWithImporterStatusError(c, http.StatusBadRequest, ErrCodeImporterFileOpenFailed, "Failed to open uploaded file", "failed to open uploaded file")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.log.Error("could not close uploaded import file", "error", err)
		}
	}()

	targetBasePath := c.PostForm("targetBasePath")
	out, err := r.createPlan.Execute(c.Request.Context(), CreateImportPlanInput{
		File: file, TargetBasePath: targetBasePath,
	})
	if err != nil {
		logRejectedZipExtraction(r.log, err)
		respondWithImporterError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Plan)
}

// logRejectedZipExtraction 仅对三种解压限制错误记录底层原因，保留字节数量及调用链信息。
// 对外消息由用例采用固定模板，内部原因通过日志单独保存，其他错误不在这里记录。
func logRejectedZipExtraction(log *slog.Logger, err error) {
	loc, ok := sharederrors.AsLocalizedError(err)
	if !ok {
		return
	}
	switch loc.Code {
	case ErrCodeImporterZipEntryTooLarge, ErrCodeImporterZipExtractedTooLarge, ErrCodeImporterZipRatioTooHigh:
		log.Warn("rejected import upload: zip extraction limit exceeded", "code", loc.Code, "error", loc.Cause)
	}
}

// handleGetPlan 返回当前计划快照，无计划经用例转换为空成功结果而不是 404。
func (r *Routes) handleGetPlan(c *gin.Context) {
	out, err := r.getPlan.Execute(c.Request.Context())
	if err != nil {
		respondWithImporterError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Plan)
}

// handleExecute 将当前用户作为导入作者提交启动请求，不等待后台导入结束。
// 刚启动或状态仍运行中时返回 202，其余成功状态返回 200。
func (r *Routes) handleExecute(c *gin.Context) {
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	out, err := r.execute.Execute(c.Request.Context(), ExecuteImportInput{UserID: user.ID})
	if err != nil {
		respondWithImporterError(c, err)
		return
	}

	statusCode := http.StatusOK
	if out.Started || out.State.ExecutionStatus == coreimporter.ExecutionStatusRunning {
		statusCode = http.StatusAccepted
	}
	c.JSON(statusCode, out.State)
}

// handleClearPlan 对未退出的取消任务返回 202 与状态，清理完成后返回 200 和空值。
func (r *Routes) handleClearPlan(c *gin.Context) {
	state, err := r.clearPlan.Execute(c.Request.Context())
	if err != nil {
		respondWithImporterError(c, err)
		return
	}
	if state != nil {
		// 非空状态表示取消标记已设置，执行尚未结束，前端继续轮询。
		c.JSON(http.StatusAccepted, state)
		return
	}
	c.JSON(http.StatusOK, nil)
}
