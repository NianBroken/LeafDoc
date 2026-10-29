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

// Routes is the RouteRegistrar for the avatar domain.
type Routes struct {
	uploadAvatar  *UploadAvatarUseCase
	deleteAvatar  *DeleteAvatarUseCase
	avatarService *coreavatar.AvatarService
	authService   *coreauth.AuthService
	userService   func() *coreauth.UserService
}

// RoutesConfig holds the dependencies required to build a Routes instance.
type RoutesConfig struct {
	UploadAvatar  *UploadAvatarUseCase
	DeleteAvatar  *DeleteAvatarUseCase
	AvatarService *coreavatar.AvatarService
	AuthService   *coreauth.AuthService
	UserService   func() *coreauth.UserService
}

// NewRoutes constructs the avatar RouteRegistrar.
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		uploadAvatar:  cfg.UploadAvatar,
		deleteAvatar:  cfg.DeleteAvatar,
		avatarService: cfg.AvatarService,
		authService:   cfg.AuthService,
		userService:   cfg.UserService,
	}
}

// RegisterRoutes implements RouteRegistrar. Any authenticated user manages
// only their own avatar — there is no admin override (self-service only).
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	base := ctx.Base

	// Public avatar static file server — unauthenticated, path-traversal
	// protected. Existing accounts without uploads receive the built-in avatar.
	base.GET("/avatars/:userId", r.handleServeAvatar)

	authGroup := ctx.APIAuthGroup(r.authService)
	authGroup.POST("/user/avatar", r.handleUploadAvatar)
	authGroup.DELETE("/user/avatar", r.handleDeleteAvatar)
}

// ─── Handlers ───────────────────────────────────────────────────────────────

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

func (r *Routes) handleDeleteAvatar(c *gin.Context) {
	user := authmw.MustGetUser(c)

	if err := r.deleteAvatar.Execute(c.Request.Context(), DeleteAvatarInput{UserID: user.ID}); err != nil {
		respondWithAvatarError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func (r *Routes) handleServeAvatar(c *gin.Context) {
	userID := c.Param("userId")

	// Defense in depth against path traversal — AvatarService.AvatarPath
	// already rejects unsafe ids internally, but reject obviously unsafe
	// values before even asking.
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

func disableClientCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))
}
