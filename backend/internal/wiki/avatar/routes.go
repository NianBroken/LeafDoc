package avatar

import (
	"errors"
	"net/http"
	"strings"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	coreavatar "github.com/NianBroken/LeafDoc/internal/core/avatar"
	"github.com/NianBroken/LeafDoc/internal/core/brandassets"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes 为公开头像读取与当前账号头像管理提供接口，修改目标始终来自鉴权上下文。
type Routes struct {
	uploadAvatar  *UploadAvatarUseCase
	deleteAvatar  *DeleteAvatarUseCase
	avatarService *coreavatar.AvatarService
	authService   *coreauth.AuthService
	userService   func() *coreauth.UserService
}

// RoutesConfig 复用头像用例、文件服务及动态用户服务，公开读取也检查账号是否存在。
type RoutesConfig struct {
	UploadAvatar  *UploadAvatarUseCase
	DeleteAvatar  *DeleteAvatarUseCase
	AvatarService *coreavatar.AvatarService
	AuthService   *coreauth.AuthService
	UserService   func() *coreauth.UserService
}

// NewRoutes 保存现有依赖，不在构造时读取头像。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		uploadAvatar:  cfg.UploadAvatar,
		deleteAvatar:  cfg.DeleteAvatar,
		avatarService: cfg.AvatarService,
		authService:   cfg.AuthService,
		userService:   cfg.UserService,
	}
}

// RegisterRoutes 直接挂载公开头像 GET，并为当前账号的状态、上传和删除使用鉴权组。
// 管理接口不接受其他用户标识，也不提供管理员代改入口。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	base := ctx.Base

	// 公开读取不额外要求登录，存在的账号没有上传头像时返回内置默认图片。
	base.GET("/avatars/:userId", r.handleServeAvatar)

	authGroup := ctx.APIAuthGroup(r.authService)
	authGroup.GET("/user/avatar", r.handleAvatarStatus)
	authGroup.POST("/user/avatar", r.handleUploadAvatar)
	authGroup.DELETE("/user/avatar", r.handleDeleteAvatar)
}

// handleAvatarStatus 按当前用户检查自定义文件，并禁止缓存这个随上传变化的状态。
// 路由鉴权必须保证 MustGetUser 返回有效用户后才能继续访问其标识。
func (r *Routes) handleAvatarStatus(c *gin.Context) {
	user := authmw.MustGetUser(c)
	_, found := r.avatarService.AvatarPath(user.ID)
	disableClientCache(c)
	c.JSON(http.StatusOK, gin.H{"hasCustomAvatar": found})
}

// handleUploadAvatar 使用头像模块的大小限制解析 file 字段，再提交当前用户的文件。
// 解析超限返回 413，其他解析错误返回缺文件的 400；成功后无响应体并返回 200。
// 文件延迟关闭，当前实现不报告关闭错误。
func (r *Routes) handleUploadAvatar(c *gin.Context) {
	user := authmw.MustGetUser(c)

	file, header, err := httpinternal.ParseUploadedFile(c, coreavatar.MaxUploadSize, "file")
	if err != nil {
		if errors.Is(err, httpinternal.ErrUploadTooLarge) {
			respondWithAvatarStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeAvatarUploadFailed, "File too large", "file too large")
		} else {
			respondWithAvatarStatusError(c, http.StatusBadRequest, ErrCodeAvatarUploadFailed, "Missing file", "missing file")
		}
		return
	}
	defer func() { _ = file.Close() }()

	if err := r.uploadAvatar.Execute(c.Request.Context(), UploadAvatarInput{
		UserID:   user.ID,
		File:     file,
		Filename: header.Filename,
	}); err != nil {
		respondWithAvatarError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// handleDeleteAvatar 只删除当前账号头像，用例成功后返回 200，默认图片由公开读取路径回退提供。
func (r *Routes) handleDeleteAvatar(c *gin.Context) {
	user := authmw.MustGetUser(c)

	if err := r.deleteAvatar.Execute(c.Request.Context(), DeleteAvatarInput{UserID: user.ID}); err != nil {
		respondWithAvatarError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// handleServeAvatar 拒绝危险路径文本，检查账号存在，再选择上传文件或内置 WebP 默认图。
// 任意账号查找错误都返回 404，内置图片读取失败返回 500，成功图片响应禁止缓存。
func (r *Routes) handleServeAvatar(c *gin.Context) {
	userID := c.Param("userId")

	// 在调用服务前拒绝斜杠、反斜杠或双点，头像服务自身仍执行其标识检查。
	if strings.ContainsAny(userID, "/\\") || strings.Contains(userID, "..") {
		c.Status(http.StatusNotFound)
		return
	}

	if _, err := r.userService().GetUserByID(userID); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	path, found := r.avatarService.AvatarPath(userID)
	if !found {
		data, err := brandassets.Read(brandassets.AvatarFile)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		disableClientCache(c)
		c.Data(http.StatusOK, "image/webp", data)
		return
	}

	disableClientCache(c)
	c.File(path)
}

// disableClientCache 禁止浏览器及中间缓存保留头像或状态，避免同名文件更新后显示旧内容。
func disableClientCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))
}
