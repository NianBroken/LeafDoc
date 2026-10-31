package assets

import (
	"log/slog"
	"net/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// maxMultipartMemory 是 multipart 解析保留文件内容的内存阈值，不是请求总大小限制。
// 总大小由每次上传使用的 MaxBytesReader 控制。
const maxMultipartMemory = 32 << 20 // 32 MiB。

// Routes 将资源查询、修改及静态文件响应绑定到现有用例和文件目录。
type Routes struct {
	upload      *UploadAssetUseCase
	list        *ListAssetsUseCase
	rename      *RenameAssetUseCase
	delete      *DeleteAssetUseCase
	authService *coreauth.AuthService
	assetsDir   string
	log         *slog.Logger
}

// RoutesConfig 提供用例、鉴权、静态目录及日志，构造函数直接保存而不填充默认服务。
type RoutesConfig struct {
	Upload      *UploadAssetUseCase
	List        *ListAssetsUseCase
	Rename      *RenameAssetUseCase
	Delete      *DeleteAssetUseCase
	AuthService *coreauth.AuthService
	AssetsDir   string
	Log         *slog.Logger
}

// NewRoutes 保存资源路由依赖，不在构造阶段读取磁盘或挂载 URL。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		upload:      cfg.Upload,
		list:        cfg.List,
		rename:      cfg.Rename,
		delete:      cfg.Delete,
		authService: cfg.AuthService,
		assetsDir:   cfg.AssetsDir,
		log:         cfg.Log,
	}
}

// RegisterRoutes 分别挂载资源文件、页面资源列表与编辑者写接口。
// 静态读取和列表使用公开读取门禁，写接口要求登录及管理员或编辑者角色。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	opts := ctx.Opts

	// 静态文件组禁止目录列表，先设置不可信文件响应头，再执行公开或登录读取鉴权。
	if r.assetsDir != "" {
		assetsFS := gin.Dir(r.assetsDir, false)
		assetsGroup := ctx.Base.Group("/assets")
		assetsGroup.Use(
			assetServeHeaders(),
			authmw.RequireAuthOrPublicRead(r.authService, ctx.AuthCookies, opts.PublicAccess),
		)
		assetsGroup.StaticFS("/", assetsFS)
	}

	// 列表只注册一次，每次请求读取当前公开访问设置。
	readGroup := ctx.APIReadGroup(r.authService)
	readGroup.GET("/pages/:id/assets", r.handleList)

	// 写操作通过通用鉴权及 CSRF 组，再检查有效用户的编辑权限。
	authGroup := ctx.APIAuthGroup(r.authService)
	authGroup.POST("/pages/:id/assets", authmw.RequireEditorOrAdmin(), r.handleUpload(opts.MaxAssetUploadSizeBytes))
	authGroup.PUT("/pages/:id/assets/rename", authmw.RequireEditorOrAdmin(), r.handleRename)
	authGroup.DELETE("/pages/:id/assets/:name", authmw.RequireEditorOrAdmin(), r.handleDelete)
}

// handleUpload 限制整个请求体的字节数，再解析名为 file 的 multipart 文件。
// 所有表单解析错误均返回 413，所有 FormFile 错误均返回缺少文件的 400。
// 成功打开的文件由本处理器延迟关闭，用例成功后返回 201 与资源地址。
func (r *Routes) handleUpload(maxUploadSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 限制包括 multipart 边界与字段开销，不只计算单个文件的内容字节。
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

		if err := c.Request.ParseMultipartForm(maxMultipartMemory); err != nil {
			respondWithAssetStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeAssetFileTooLarge, "File too large", "file too large")
			return
		}

		pageID := c.Param("id")
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetMissingFile, "Missing file", "missing file")
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				r.log.Error("could not close uploaded file", "error", err)
			}
		}()

		user := authmw.MustGetUser(c)
		if user == nil {
			return
		}

		out, err := r.upload.Execute(c.Request.Context(), UploadAssetInput{
			UserID: user.ID, PageID: pageID, File: file, Filename: header.Filename, MaxBytes: maxUploadSize,
		})
		if err != nil {
			respondWithAssetError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"file": out.URL})
	}
}

// handleList 按页面标识取得资源地址列表，以 files 字段返回，失败交给资源错误映射。
func (r *Routes) handleList(c *gin.Context) {
	pageID := c.Param("id")
	out, err := r.list.Execute(c.Request.Context(), ListAssetsInput{PageID: pageID})
	if err != nil {
		respondWithAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": out.Files})
}

// handleRename 要求两个非空文件名，从鉴权上下文取得操作者后交给重命名用例。
// 返回的 url 是资源服务形成的访问地址，不是磁盘文件路径。
func (r *Routes) handleRename(c *gin.Context) {
	pageID := c.Param("id")
	var req struct {
		OldFilename string `json:"old_filename" binding:"required"`
		NewFilename string `json:"new_filename" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetInvalidPayload, "Invalid payload", "invalid payload")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	out, err := r.rename.Execute(c.Request.Context(), RenameAssetInput{
		UserID: user.ID, PageID: pageID, OldFilename: req.OldFilename, NewFilename: req.NewFilename,
	})
	if err != nil {
		respondWithAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": out.URL})
}

// handleDelete 拒绝空文件名，再按页面与文件名执行删除，成功返回固定确认消息。
func (r *Routes) handleDelete(c *gin.Context) {
	pageID := c.Param("id")
	filename := c.Param("name")
	if filename == "" {
		respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetMissingName, "Missing filename", "missing filename")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.delete.Execute(c.Request.Context(), DeleteAssetInput{
		UserID: user.ID, PageID: pageID, Filename: filename,
	}); err != nil {
		respondWithAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
}
