package branding

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/brandassets"
	corebanding "github.com/NianBroken/LeafDoc/internal/core/branding"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes 为公开品牌读取、图标文件响应及管理员品牌写入提供处理器。
type Routes struct {
	getBranding     *GetBrandingUseCase
	updateBranding  *UpdateBrandingUseCase
	uploadLogo      *UploadLogoUseCase
	deleteLogo      *DeleteLogoUseCase
	uploadFavicon   *UploadFaviconUseCase
	deleteFavicon   *DeleteFaviconUseCase
	brandingService *corebanding.BrandingService
	authService     *coreauth.AuthService
	log             *slog.Logger
}

// RoutesConfig 让各用例与文件响应共享同一品牌服务和鉴权配置。
type RoutesConfig struct {
	GetBranding     *GetBrandingUseCase
	UpdateBranding  *UpdateBrandingUseCase
	UploadLogo      *UploadLogoUseCase
	DeleteLogo      *DeleteLogoUseCase
	UploadFavicon   *UploadFaviconUseCase
	DeleteFavicon   *DeleteFaviconUseCase
	BrandingService *corebanding.BrandingService
	AuthService     *coreauth.AuthService
	Log             *slog.Logger
}

// NewRoutes 保存现有依赖，不执行文件或品牌配置读写。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		getBranding:     cfg.GetBranding,
		updateBranding:  cfg.UpdateBranding,
		uploadLogo:      cfg.UploadLogo,
		deleteLogo:      cfg.DeleteLogo,
		uploadFavicon:   cfg.UploadFavicon,
		deleteFavicon:   cfg.DeleteFavicon,
		brandingService: cfg.BrandingService,
		authService:     cfg.AuthService,
		log:             cfg.Log,
	}
}

// RegisterRoutes 将品牌配置及图片读取直接挂在基础组，写接口另要求登录与管理员角色。
// 读取路由不额外检查公开模式，但仍继承基础组先前已有的中间件。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	base := ctx.Base

	// 品牌读取不另加登录守卫，供登录页面与首屏显示使用。
	nonAuth := base.Group("/api")
	nonAuth.GET("/branding", r.handleGetBranding)

	// 按名称读取与当前 favicon 入口共享路径检查，后者另有内置图片回退。
	base.GET("/branding/:filename", r.handleServeBrandingAsset)
	base.GET("/favicon.ico", r.handleServeCurrentFavicon)

	// 写操作经通用鉴权及 CSRF 处理，再要求有效管理员角色。
	authGroup := ctx.APIAuthGroup(r.authService)
	authGroup.PUT("/branding", authmw.RequireAdmin(), r.handleUpdateBranding)
	authGroup.POST("/branding/logo", authmw.RequireAdmin(), r.handleUploadLogo)
	authGroup.POST("/branding/favicon", authmw.RequireAdmin(), r.handleUploadFavicon)
	authGroup.DELETE("/branding/logo", authmw.RequireAdmin(), r.handleDeleteLogo)
	authGroup.DELETE("/branding/favicon", authmw.RequireAdmin(), r.handleDeleteFavicon)
}

// handleGetBranding 返回服务提供的品牌配置和约束，错误按品牌领域规则编码。
func (r *Routes) handleGetBranding(c *gin.Context) {
	out, err := r.getBranding.Execute(c.Request.Context())
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Config)
}

// handleUpdateBranding 解析 siteName 后交给用例验证与保存，不在路由中使用必填绑定规则。
func (r *Routes) handleUpdateBranding(c *gin.Context) {
	var req struct {
		SiteName string `json:"siteName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingInvalidPayload, "Invalid payload", "invalid payload")
		return
	}
	out, err := r.updateBranding.Execute(c.Request.Context(), UpdateBrandingInput{SiteName: req.SiteName})
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Config)
}

// handleUploadLogo 先读取当前品牌约束，以标识文件大小限制解析上传，再调用保存用例。
// 超限返回 413，其他文件解析错误归为缺文件；文件关闭失败单独记录日志。
func (r *Routes) handleUploadLogo(c *gin.Context) {
	constraints, err := r.brandingService.GetBranding()
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	maxSize := constraints.BrandingConstraints.MaxLogoSize
	file, header, err := httpinternal.ParseUploadedFile(c, maxSize, "file")
	if err != nil {
		if errors.Is(err, httpinternal.ErrUploadTooLarge) {
			respondWithBrandingStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeBrandingLogoTooLarge, "File too large", "file too large")
		} else {
			respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingLogoMissing, "Missing file", "missing file")
		}
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.log.Error("could not close logo file", "error", err)
		}
	}()
	out, err := r.uploadLogo.Execute(c.Request.Context(), UploadLogoInput{File: file, Filename: header.Filename})
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": out.Path, "branding": out.Config})
}

// handleDeleteLogo 调用删除用例后返回当前品牌配置，不在该路由删除内置默认资源。
func (r *Routes) handleDeleteLogo(c *gin.Context) {
	out, err := r.deleteLogo.Execute(c.Request.Context())
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"branding": out.Config})
}

// handleUploadFavicon 使用图标专属大小限制，成功返回存储服务给出的路径及最新配置。
// 上传文件由处理器延迟关闭，错误分别经过上传或品牌领域映射。
func (r *Routes) handleUploadFavicon(c *gin.Context) {
	constraints, err := r.brandingService.GetBranding()
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	maxSize := constraints.BrandingConstraints.MaxFaviconSize
	file, header, err := httpinternal.ParseUploadedFile(c, maxSize, "file")
	if err != nil {
		if errors.Is(err, httpinternal.ErrUploadTooLarge) {
			respondWithBrandingStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeBrandingFaviconTooLarge, "File too large", "file too large")
		} else {
			respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingFaviconMissing, "Missing file", "missing file")
		}
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.log.Error("could not close favicon file", "error", err)
		}
	}()
	out, err := r.uploadFavicon.Execute(c.Request.Context(), UploadFaviconInput{File: file, Filename: header.Filename})
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": out.Path, "branding": out.Config})
}

// handleDeleteFavicon 删除用户图标并返回配置，后续当前图标读取可使用默认图片回退。
func (r *Routes) handleDeleteFavicon(c *gin.Context) {
	out, err := r.deleteFavicon.Execute(c.Request.Context())
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"branding": out.Config})
}

// handleServeBrandingAsset 读取约束并验证请求文件名，成功后禁止缓存并发送磁盘文件。
// 路径校验状态原样返回，不将所有文件问题统一当作不存在。
func (r *Routes) handleServeBrandingAsset(c *gin.Context) {
	cfg, err := r.brandingService.GetBranding()
	if err != nil {
		r.log.Error("failed to get branding constraints", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	cleanPath, status := r.resolveBrandingAssetPath(c.Param("filename"), cfg)
	if status != http.StatusOK {
		c.Status(status)
		return
	}

	disableClientCache(c)
	c.File(cleanPath)
}

// handleServeCurrentFavicon 优先读取配置中的图标，路径禁止或文件不存在时回退内置 ICO。
// 品牌读取或文件系统内部错误返回 500，不在这些分支用默认图掩盖错误。
func (r *Routes) handleServeCurrentFavicon(c *gin.Context) {
	cfg, err := r.brandingService.GetBranding()
	if err != nil {
		r.log.Error("failed to get branding config", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if cfg.FaviconFile != "" {
		cleanPath, status := r.resolveBrandingAssetPath(cfg.FaviconFile, cfg)
		if status == http.StatusOK {
			disableClientCache(c)
			c.File(cleanPath)
			return
		}
		if status == http.StatusInternalServerError {
			c.Status(status)
			return
		}
	}

	disableClientCache(c)
	data, err := brandassets.Read(brandassets.FaviconFile)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "image/x-icon", data)
}

// resolveBrandingAssetPath 按安全名称、标识及图标扩展名并集、目录相对关系和文件状态逐步检查。
// 采用路径清理与 Stat，不解析符号链接目标，也不验证文件内容或确认目标一定为普通文件。
func (r *Routes) resolveBrandingAssetPath(filename string, cfg *corebanding.BrandingConfigResponse) (string, int) {
	// 同一入口同时检查请求名称和当前配置中的文件名。
	if corebanding.ContainsUnsafePath(filename) {
		return "", http.StatusForbidden
	}

	// 两类允许扩展名合并后查询，读取时不要求文件属于配置当前选中的那个名称。
	allowedExts := make(map[string]bool)
	for _, ext := range cfg.BrandingConstraints.LogoExts {
		allowedExts[ext] = true
	}
	for _, ext := range cfg.BrandingConstraints.FaviconExts {
		allowedExts[ext] = true
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExts[ext] {
		return "", http.StatusForbidden
	}

	brandingDir := r.brandingService.GetBrandingAssetsDir()
	filePath := filepath.Join(brandingDir, filename)
	cleanPath := filepath.Clean(filePath)
	cleanBrandingDir := filepath.Clean(brandingDir)

	rel, err := filepath.Rel(cleanBrandingDir, cleanPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", http.StatusForbidden
	}

	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return "", http.StatusNotFound
	} else if err != nil {
		r.log.Error("error checking branding file", "error", err, "path", cleanPath)
		return "", http.StatusInternalServerError
	}

	return cleanPath, http.StatusOK
}

// disableClientCache 禁止同名品牌资源更新后仍被浏览器或代理复用旧响应。
func disableClientCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))
}
