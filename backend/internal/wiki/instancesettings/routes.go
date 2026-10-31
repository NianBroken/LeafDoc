// Package instancesettings 通过管理员接口管理运行期偏好、匿名读取及目录显示设置。
package instancesettings

import (
	"log/slog"
	"net/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tocdisplay"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes 持有公开访问与目录服务，以及鉴权和日志入口，不复制设置本身。
type Routes struct {
	publicAccess *publicaccess.Service
	tocDisplay   *tocdisplay.Service
	authService  *coreauth.AuthService
	log          *slog.Logger
}

// NewRoutes 保存服务依赖，日志为空时使用默认日志器，不在构造时读写设置。
func NewRoutes(publicAccess *publicaccess.Service, tocDisplay *tocdisplay.Service, authService *coreauth.AuthService, log *slog.Logger) *Routes {
	if log == nil {
		log = slog.Default()
	}
	return &Routes{publicAccess: publicAccess, tocDisplay: tocDisplay, authService: authService, log: log}
}

// RegisterRoutes 在登录鉴权组下为各接口要求管理员角色。
// 偏好接口只在偏好服务非空时注册，偏好写入与公开模式写入额外拒绝 API 密钥身份。
// 目录设置写入没有该额外限制，仍受鉴权组的 CSRF 和管理员规则约束。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	adminGroup := ctx.APIAuthGroup(r.authService).Group("/admin/settings")
	if ctx.Opts.Preferences != nil {
		adminGroup.GET("/preferences", authmw.RequireAdmin(), func(c *gin.Context) { c.JSON(200, ctx.Opts.Preferences.Get()) })
		// JSON 对象作为局部设置交给服务校验并保存，本地化校验错误返回 400。
		adminGroup.PUT("/preferences", authmw.RequireAdmin(), authmw.RequireCookieSession(), func(c *gin.Context) {
			var patch map[string]any
			if err := c.ShouldBindJSON(&patch); err != nil {
				c.JSON(400, gin.H{"error": localization.Message("invalid payload")})
				return
			}
			value, err := ctx.Opts.Preferences.Patch(patch)
			if err != nil {
				if loc, ok := sharederrors.AsLocalizedError(err); ok {
					httpinternal.RespondError(c, http.StatusBadRequest, loc.Code, loc.Message, loc.Template, loc.Args...)
				} else {
					r.log.Error("failed to save preferences", "error", err)
					httpinternal.RespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to save preferences", "failed to save preferences")
				}
				return
			}
			c.JSON(200, value)
		})
	}
	// 公开读取影响整个实例，RequireCookieSession 通过密钥标记拒绝 API 密钥调用。
	// 该守卫不自行读取 Cookie，其他已注入用户身份仍遵循上游鉴权规则。
	adminGroup.PUT("/public-access",
		authmw.RequireAdmin(),
		authmw.RequireCookieSession(),
		r.handleSetPublicAccess,
	)
	adminGroup.PUT("/toc-display",
		authmw.RequireAdmin(),
		r.handleSetTocDisplay,
	)
}

// handleSetPublicAccess 使用布尔指针区分缺少字段与明确关闭，缺失或非法 JSON 返回 400。
// 服务写入成功后重新读取实际值，并将操作者标识写入日志。
func (r *Routes) handleSetPublicAccess(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")
		return
	}

	if err := r.publicAccess.SetEnabled(*req.Enabled); err != nil {
		// 环境变量管理的实例由服务返回冲突代码，经领域错误映射返回 409。
		respondWithError(c, err)
		return
	}

	r.log.Info("public access mode changed",
		"enabled", r.publicAccess.Enabled(),
		"actor_id", actorID(c),
	)
	c.JSON(http.StatusOK, gin.H{"enabled": r.publicAccess.Enabled()})
}

// handleSetTocDisplay 优先采用非空 mode；仅未提供模式时将 alwaysShow 转为 always 或 auto。
// 两种表示都未提供则拒绝，模式合法性由目录服务校验，响应返回服务实际保存的两种表示。
func (r *Routes) handleSetTocDisplay(c *gin.Context) {
	var req struct {
		AlwaysShow *bool  `json:"alwaysShow"`
		Mode       string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.AlwaysShow == nil && req.Mode == "") {
		respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "auto"
		if *req.AlwaysShow {
			mode = "always"
		}
	}
	if err := r.tocDisplay.SetMode(mode); err != nil {
		respondWithError(c, err)
		return
	}

	r.log.Info("toc always-show setting changed",
		"always_show", r.tocDisplay.AlwaysShow(),
		"actor_id", actorID(c),
	)
	c.JSON(http.StatusOK, gin.H{"alwaysShow": r.tocDisplay.AlwaysShow(), "mode": r.tocDisplay.Mode()})
}

// actorID 从已有鉴权上下文取用户标识，无法取得用户时返回空文本用于日志字段。
func actorID(c *gin.Context) string {
	if u := authmw.TryGetUser(c); u != nil {
		return u.ID
	}
	return ""
}
