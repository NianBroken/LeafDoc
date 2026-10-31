package revisions

import (
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/NianBroken/LeafDoc/internal/http/dto"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// 页面及版本必填消息由历史读取和恢复处理器共享，不作为状态判断依据。
const (
	errPageIDRequiredUserMsg   = "Page ID is required"
	errPageIDRequiredLogMsg    = "page id is required"
	errRevisionNotFoundUserMsg = "Revision not found"
)

// Routes 持有历史查询、资源读取与恢复用例，作者解析器用于响应中的名称标签。
// 完整性用例保存在配置中，本文件的注册列表没有为它挂载独立 HTTP 路由。
type Routes struct {
	listRevisions    *ListRevisionsUseCase
	getRevision      *GetRevisionUseCase
	compareRevisions *CompareRevisionsUseCase
	getRevisionAsset *GetRevisionAssetUseCase
	getLatest        *GetLatestRevisionUseCase
	restoreRevision  *RestoreRevisionUseCase
	checkIntegrity   *CheckIntegrityUseCase
	userResolver     *coreauth.UserResolver
	authService      *coreauth.AuthService
}

// RoutesConfig 提供历史用例与鉴权、作者解析依赖。
type RoutesConfig struct {
	ListRevisions    *ListRevisionsUseCase
	GetRevision      *GetRevisionUseCase
	CompareRevisions *CompareRevisionsUseCase
	GetRevisionAsset *GetRevisionAssetUseCase
	GetLatest        *GetLatestRevisionUseCase
	RestoreRevision  *RestoreRevisionUseCase
	CheckIntegrity   *CheckIntegrityUseCase
	UserResolver     *coreauth.UserResolver
	AuthService      *coreauth.AuthService
}

// NewRoutes 保存依赖，功能是否挂载在 RegisterRoutes 中读取开关。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		listRevisions:    cfg.ListRevisions,
		getRevision:      cfg.GetRevision,
		compareRevisions: cfg.CompareRevisions,
		getRevisionAsset: cfg.GetRevisionAsset,
		getLatest:        cfg.GetLatest,
		restoreRevision:  cfg.RestoreRevision,
		checkIntegrity:   cfg.CheckIntegrity,
		userResolver:     cfg.UserResolver,
		authService:      cfg.AuthService,
	}
}

// RegisterRoutes 只在历史功能启用时挂载接口，所有历史读取都要求鉴权，不使用公开读取组。
// 恢复接口另外要求编辑者或管理员角色。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	opts := ctx.Opts

	authGroup := ctx.APIAuthGroup(r.authService)

	// 功能关闭时不注册这些 URL，而不是注册后返回空历史。
	if opts.EnableRevision {
		authGroup.GET("/pages/:id/revisions", r.handleListRevisions)
		authGroup.GET("/pages/:id/revisions/latest", r.handleGetLatestRevision)
		authGroup.GET("/pages/:id/revisions/compare", r.handleCompareRevisions)
		authGroup.GET("/pages/:id/revisions/:revisionId/assets/*name", r.handleGetRevisionAsset)
		authGroup.GET("/pages/:id/revisions/:revisionId", r.handleGetRevision)
		authGroup.POST("/pages/:id/revisions/:revisionId/restore", authmw.RequireEditorOrAdmin(), r.handleRestoreRevision)
	}

}

// handleListRevisions 去除页面、游标与条数字符串的边缘空白，条数默认五十且限定一至二百。
// 成功响应转换作者标签并返回下一页游标，服务错误走统一历史错误映射。
func (r *Routes) handleListRevisions(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}

	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 200 {
			respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidLimit, "Revision list limit is invalid", "revision list limit for page %s is invalid", pageID)
			return
		}
		limit = parsed
	}

	out, err := r.listRevisions.Execute(c.Request.Context(), ListRevisionsInput{
		PageID: pageID,
		Cursor: strings.TrimSpace(c.Query("cursor")),
		Limit:  limit,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}

	result := make([]*RevisionResponse, 0, len(out.Revisions))
	for _, rev := range out.Revisions {
		result = append(result, toRevisionResponse(rev, r.userResolver))
	}
	c.JSON(http.StatusOK, gin.H{
		"revisions":  result,
		"nextCursor": out.NextCursor,
	})
}

// handleGetRevision 要求页面与版本标识，快照或其版本记录为空时直接返回 404。
func (r *Routes) handleGetRevision(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	revisionID := strings.TrimSpace(c.Param("revisionId"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}
	if revisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidRevisionID, "Revision ID is required", "revision id is required")
		return
	}

	out, err := r.getRevision.Execute(c.Request.Context(), GetRevisionInput{PageID: pageID, RevisionID: revisionID})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Snapshot == nil || out.Snapshot.Revision == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision %s for page %s not found", revisionID, pageID)
		return
	}
	c.JSON(http.StatusOK, toSnapshotResponse(out.Snapshot, r.userResolver))
}

// handleGetLatestRevision 读取单页最新版本，服务返回空记录时作为版本不存在处理。
func (r *Routes) handleGetLatestRevision(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}

	out, err := r.getLatest.Execute(c.Request.Context(), GetLatestRevisionInput{PageID: pageID})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Revision == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision for page %s not found", pageID)
		return
	}
	c.JSON(http.StatusOK, toRevisionResponse(out.Revision, r.userResolver))
}

// handleCompareRevisions 要求页面、基准和目标版本，任一比较快照缺失时返回 404。
// 差异计算由用例及历史服务完成，路由只转换响应结构。
func (r *Routes) handleCompareRevisions(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	baseRevisionID := strings.TrimSpace(c.Query("base"))
	targetRevisionID := strings.TrimSpace(c.Query("target"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}
	if baseRevisionID == "" || targetRevisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionCompareInvalidRequest, "Revision compare request is invalid", "revision compare request for page %s is invalid", pageID)
		return
	}

	out, err := r.compareRevisions.Execute(c.Request.Context(), CompareRevisionsInput{
		PageID: pageID, BaseRevisionID: baseRevisionID, TargetRevisionID: targetRevisionID,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Comparison == nil || out.Comparison.Base == nil || out.Comparison.Target == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision compare resource for page %s not found", pageID)
		return
	}
	c.JSON(http.StatusOK, toComparisonResponse(out.Comparison, r.userResolver))
}

// handleGetRevisionAsset 从历史快照定位资源字节，打开文件并设置不可信内容响应头。
// 类型优先用记录中的 MIME，其次按扩展名查询，再回退二进制流类型。
// ServeContent 使用可定位文件及修改时间处理内容响应，文件关闭错误在当前实现中不报告。
func (r *Routes) handleGetRevisionAsset(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	revisionID := strings.TrimSpace(c.Param("revisionId"))
	assetName := strings.TrimSpace(strings.TrimPrefix(c.Param("name"), "/"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}
	if revisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidRevisionID, "Revision ID is required", "revision id is required")
		return
	}
	if assetName == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionPreviewAssetInvalidName, "Revision asset name is invalid", "revision asset name for page %s revision %s is invalid", pageID, revisionID)
		return
	}

	out, err := r.getRevisionAsset.Execute(c.Request.Context(), GetRevisionAssetInput{
		PageID: pageID, RevisionID: revisionID, AssetName: assetName,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Asset == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionPreviewAssetNotFound, "Revision asset not found", "revision asset %s for page %s revision %s not found", assetName, pageID, revisionID)
		return
	}

	// 快照索引存在而实际字节文件无法打开时返回内部不可用错误，不当作空文件发送。
	f, err := os.Open(out.Asset.Path)
	if err != nil {
		respondWithRevisionStatusError(c, http.StatusInternalServerError, ErrCodeRevisionPreviewAssetBlobUnavailable, "Revision asset blob is unavailable", "revision asset blob %s for page %s revision %s is unavailable", assetName, pageID, revisionID)
		return
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		respondWithRevisionStatusError(c, http.StatusInternalServerError, ErrCodeRevisionInternalError, "Failed to load revision asset", "failed to load revision asset")
		return
	}

	contentType := out.Asset.Asset.MIMEType
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(path.Ext(assetName)))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	security.SetUntrustedFileHeaders(c.Writer.Header(), assetName)
	c.Writer.Header().Set("Content-Type", contentType)
	http.ServeContent(c.Writer, c.Request, path.Base(assetName), stat.ModTime(), f)
}

// handleRestoreRevision 验证标识后使用当前用户作为恢复作者，成功返回当前页面公开结构。
func (r *Routes) handleRestoreRevision(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	revisionID := strings.TrimSpace(c.Param("revisionId"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionRestoreInvalidPageID, "Failed to restore page", "failed to restore page %s", pageID)
		return
	}
	if revisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionRestoreInvalidRevision, "Restore revision is invalid", "restore revision %s for page %s is invalid", revisionID, pageID)
		return
	}

	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	out, err := r.restoreRevision.Execute(c.Request.Context(), RestoreRevisionInput{
		UserID: user.ID, PageID: pageID, RevisionID: revisionID,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToAPIPage(out.Page, r.userResolver))
}
