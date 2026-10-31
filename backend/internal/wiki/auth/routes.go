package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	coreavatar "github.com/NianBroken/LeafDoc/internal/core/avatar"
	"github.com/NianBroken/LeafDoc/internal/core/preferences"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/utils"
	"github.com/gin-gonic/gin"
)

// 共享消息用于 Cookie 安全检查及载荷错误，文本作为响应与翻译模板保持独立。
const (
	httpsRequiredLogMsg        = "https required for auth cookies use allow insecure"
	httpsRequiredUserMsg       = "HTTPS is required for auth cookies. Use HTTPS or start LeafDoc with --allow-insecure for trusted plain HTTP setups."
	errInvalidRequestUserMsg   = "Invalid request"
	errInvalidRequestLogMsg    = "invalid request"
	errFailedToIssueCSRFCookie = "Failed to issue CSRF cookie"
)

// Routes 将登录、刷新、账号管理及邮件凭据用例接入 HTTP，并协调浏览器 Cookie。
// 用例执行领域操作，路由解析请求、选择权限、处理 Cookie 及构造响应。
type Routes struct {
	login             *LoginUseCase
	logout            *LogoutUseCase
	refreshToken      *RefreshTokenUseCase
	createUser        *CreateUserUseCase
	updateUser        *UpdateUserUseCase
	changeOwnPassword *ChangeOwnPasswordUseCase
	deleteUser        *DeleteUserUseCase
	getUsers          *GetUsersUseCase
	getUserByID       *GetUserByIDUseCase
	authService       *coreauth.AuthService

	requestPasswordReset *RequestPasswordResetUseCase
	confirmPasswordReset *ConfirmPasswordResetUseCase
	inviteUser           *InviteUserUseCase
	resendInvite         *ResendInviteUseCase
	confirmInvite        *ConfirmInviteUseCase
}

// RoutesConfig 提供全部账号用例和鉴权服务，邮件未启用时仍保留对应用例入口。
type RoutesConfig struct {
	Login             *LoginUseCase
	Logout            *LogoutUseCase
	RefreshToken      *RefreshTokenUseCase
	CreateUser        *CreateUserUseCase
	UpdateUser        *UpdateUserUseCase
	ChangeOwnPassword *ChangeOwnPasswordUseCase
	DeleteUser        *DeleteUserUseCase
	GetUsers          *GetUsersUseCase
	GetUserByID       *GetUserByIDUseCase
	AuthService       *coreauth.AuthService

	RequestPasswordReset *RequestPasswordResetUseCase
	ConfirmPasswordReset *ConfirmPasswordResetUseCase
	InviteUser           *InviteUserUseCase
	ResendInvite         *ResendInviteUseCase
	ConfirmInvite        *ConfirmInviteUseCase
}

// NewRoutes 保存依赖，不执行登录、邮件发送或路由注册。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		login:             cfg.Login,
		logout:            cfg.Logout,
		refreshToken:      cfg.RefreshToken,
		createUser:        cfg.CreateUser,
		updateUser:        cfg.UpdateUser,
		changeOwnPassword: cfg.ChangeOwnPassword,
		deleteUser:        cfg.DeleteUser,
		getUsers:          cfg.GetUsers,
		getUserByID:       cfg.GetUserByID,
		authService:       cfg.AuthService,

		requestPasswordReset: cfg.RequestPasswordReset,
		confirmPasswordReset: cfg.ConfirmPasswordReset,
		inviteUser:           cfg.InviteUser,
		resendInvite:         cfg.ResendInvite,
		confirmInvite:        cfg.ConfirmInvite,
	}
}

// RegisterRoutes 区分无需预先登录、可选身份和必须登录的请求组。
// 账号写入与会话管理拒绝 API 密钥标记，用户列表读取仍允许具备管理员角色的密钥。
// 登录、刷新及邮件入口分别使用独立或共享的限流实例，所有计数按连接对端主机分组。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {

	// 登录五分钟最多累计十次请求，成功的 2xx 响应会清空该地址历史。
	loginRateLimiter := security.NewRateLimiter(10, 5*time.Minute, true)

	nonAuth := ctx.Base.Group("/api")
	nonAuth.POST("/auth/login", loginRateLimiter, r.handleLogin(ctx))
	// 刷新使用独立的一分钟三十次限制，成功也不清空，与登录计数不共用窗口。
	refreshRateLimiter := security.NewRateLimiter(30, time.Minute, false)
	nonAuth.POST("/auth/refresh-token", refreshRateLimiter, r.handleRefreshToken(ctx))

	// 配置接口不要求预先登录，响应前签发或复用 CSRF Cookie。
	nonAuth.GET("/config", r.handleConfig(ctx))

	// 当前身份接口采用可选鉴权，未取得用户时返回 200 与 null，而非要求浏览器重新认证。
	meGroup := ctx.Base.Group("/api")
	meGroup.Use(
		authmw.OptionalAuth(r.authService, ctx.AuthCookies),
	)
	meGroup.GET("/auth/me", r.handleMe)

	authGroup := ctx.APIAuthGroup(r.authService)

	// 该守卫拒绝上游已经解析的 API 密钥标记，不独立要求请求必须含 Cookie。
	// 代理注入的用户继续受普通鉴权、角色与 CSRF 规则约束。
	sessionOnly := authmw.RequireCookieSession()

	authGroup.POST("/auth/logout", sessionOnly, r.handleLogout(ctx))

	authGroup.POST("/users", sessionOnly, authmw.RequireAdmin(), r.handleCreateUser)
	authGroup.GET("/users", authmw.RequireAdmin(), r.handleGetUsers)
	authGroup.PUT("/users/:id", sessionOnly, authmw.RequireAdmin(), r.handleUpdateUser)
	authGroup.DELETE("/users/:id", sessionOnly, authmw.RequireAdmin(), r.handleDeleteUser)
	// 邀请路由始终注册，邮件服务为空时由用例返回功能关闭错误并映射为 403。
	authGroup.POST("/users/invite", sessionOnly, authmw.RequireAdmin(), r.handleInviteUser)
	authGroup.POST("/users/:id/invite/resend", sessionOnly, authmw.RequireAdmin(), r.handleResendInvite)

	authGroup.PUT("/users/me/password", sessionOnly, r.handleChangeOwnPassword(ctx))
	emailRateLimiter := security.NewRateLimiter(10, 5*time.Minute, true)
	// 三个邮件入口共用五分钟十次且成功清空的地址限制。
	// 忘记密码另外按原始标识每小时限制三次，不因成功响应清空该标识的记录。
	passwordResetIdentifierLimiter := security.NewKeyedLimiter(3, time.Hour, false)
	nonAuth.POST("/auth/password/forgot", emailRateLimiter, r.handleRequestPasswordReset(passwordResetIdentifierLimiter))
	nonAuth.POST("/auth/password/reset", emailRateLimiter, r.handleConfirmPasswordReset)
	nonAuth.POST("/auth/invite/accept", emailRateLimiter, r.handleConfirmInvite(ctx))
}

// writeAuthCookieError 将要求 HTTPS 的错误映射为 400，其他签发错误记日志并返回 500。
// 消息由调用方传入，不在日志中包含 Cookie 令牌值。
func writeAuthCookieError(c *gin.Context, err error, httpsMsg, internalMsg, logMsg string) {
	if errors.Is(err, utils.ErrHTTPSRequired) {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, httpsMsg, httpsRequiredLogMsg)
		return
	}
	slog.Default().Error(logMsg, "error", err)
	respondWithAuthStatusError(c, http.StatusInternalServerError, ErrCodeAuthInternalError, internalMsg, "failed to issue auth cookie")
}

// handleConfig 返回功能设置、动态公开与目录状态及偏好数据，并禁止客户端存储响应。
// 先完成 CSRF 签发，失败时不继续返回配置；偏好不存在时使用默认偏好。
// defaultLanguage 在构造映射后由偏好的 Language 覆盖，因此最终值以偏好为准。
func (r *Routes) handleConfig(ctx httpinternal.RouterContext) gin.HandlerFunc {
	opts := ctx.Opts
	return func(c *gin.Context) {
		if _, err := ctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				httpsRequiredUserMsg,
				errFailedToIssueCSRFCookie,
				"failed to issue config CSRF cookie",
			)
			return
		}
		config := gin.H{
			"publicAccess":            opts.PublicAccess.Enabled(),
			"publicAccessEnvManaged":  opts.PublicAccess.EnvManaged(),
			"alwaysShowToc":           opts.AlwaysShowToc.AlwaysShow(),
			"editorLimit":             opts.EditorLimit,
			"hideLinkMetadataSection": opts.HideLinkMetadataSection,
			"basePath":                opts.BasePath,
			"maxAssetUploadSizeBytes": opts.MaxAssetUploadSizeBytes,
			// 头像限制直接使用头像模块的权威常量与扩展名列表，不从资源上传限制推导。
			"maxAvatarUploadSizeBytes": coreavatar.MaxUploadSize,
			"avatarAllowedExts":        coreavatar.AllowedExts(),
			"enableRevision":           opts.EnableRevision,
			"enableLinkRefactor":       opts.EnableLinkRefactor,
			"enableApiKeyManagement":   opts.EnableAPIKeyManagement,
			"snapshotEnabled":          opts.SnapshotEnabled,
			"smtpEnabled":              opts.SMTPEnabled,
			"httpRemoteUserEnabled":    opts.HTTPRemoteUser.Enabled,
			"loginUrl":                 opts.LoginURL,
			"logoutUrl":                opts.LogoutURL,
			"userManagementUrl":        opts.UserManagementURL,
			"defaultLanguage":          opts.DefaultLanguage,
		}
		prefs := preferences.Defaults()
		if opts.Preferences != nil {
			prefs = opts.Preferences.Get()
		}
		config["preferences"] = prefs
		if opts.SettingsVersion != nil {
			config["settingsVersion"] = opts.SettingsVersion()
			config["settingsInstance"] = opts.SettingsInstance
		}
		config["defaultLanguage"] = prefs.Language
		config["tocMode"] = opts.AlwaysShowToc.Mode()
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, config)
	}
}

// handleMe 设置禁止缓存的响应头，从已有上下文返回有限的用户身份字段。
// 无用户时返回 200 与 null，不在这里触发必须登录的错误响应。
func (r *Routes) handleMe(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))

	user := authmw.TryGetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mustChangePassword": user.MustChangePassword,
		"id":                 user.ID,
		"username":           user.Username,
		"email":              user.Email,
		"role":               user.Role,
	})
}

// handleLogin 要求标识和密码，通过用例签发会话后写入 CSRF 与认证 Cookie。
// Cookie 失败时不撤销已经签发的服务端会话，成功响应不直接返回令牌明文。
func (r *Routes) handleLogin(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Identifier string `json:"identifier" binding:"required"`
			Password   string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidPayload, "Invalid login payload", "invalid login payload")
			return
		}
		out, err := r.login.Execute(c.Request.Context(), LoginInput{
			Identifier: req.Identifier, Password: req.Password,
		})
		if err != nil {
			respondWithAuthError(c, err)
			return
		}
		if _, err := rctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				"HTTPS is required for login cookies. Use HTTPS or start LeafDoc with --allow-insecure for trusted plain HTTP setups.",
				errFailedToIssueCSRFCookie,
				"failed to issue login CSRF cookie",
			)
			return
		}
		if err := rctx.AuthCookies.Set(c, out.Token.Token, out.Token.RefreshToken); err != nil {
			if errors.Is(err, utils.ErrHTTPSRequired) {
				respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,
					httpsRequiredUserMsg,
					httpsRequiredLogMsg)
				return
			}
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":              "Login successful",
			"user":                 out.Token.User,
			"accessTokenExpiresAt": out.Token.AccessTokenExpiresAt,
		})
	}
}

// handleLogout 尝试读取并撤销刷新会话，再依次清理认证及 CSRF Cookie。
// 读取或撤销失败只记录警告，Cookie 清理失败才在该路径返回错误。
// 清理完成返回成功，不据此保证已签发的访问 JWT 失效。
func (r *Routes) handleLogout(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := rctx.AuthCookies.ReadRefresh(c)
		if err != nil {
			slog.Default().Warn("could not read refresh token during logout; refresh token will not be revoked", "error", err)
		}
		if refreshToken != "" {
			if err := r.logout.Execute(c.Request.Context(), LogoutInput{RefreshToken: refreshToken}); err != nil {
				slog.Default().Warn("failed to revoke refresh token during logout", "error", err)
			}
		}
		if err := rctx.AuthCookies.Clear(c); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to clear authentication cookies", "failed to clear authentication cookies")
			return
		}
		if err := rctx.CSRFCookie.Clear(c); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCsrfFailed, "Failed to clear CSRF cookie", "failed to clear csrf cookie")
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
	}
}

// handleRefreshToken 从 Cookie 读取刷新令牌，缺失或读取错误统一返回 422。
// 刷新成功后先签发或复用 CSRF，再写认证 Cookie，Cookie 失败不回滚令牌轮换。
func (r *Routes) handleRefreshToken(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		rt, err := rctx.AuthCookies.ReadRefresh(c)
		if err != nil || rt == "" {
			respondWithAuthStatusError(c, http.StatusUnprocessableEntity, ErrCodeAuthInvalidRefreshToken, "Missing or invalid refresh token", "missing or invalid refresh token")
			return
		}
		out, err := r.refreshToken.Execute(c.Request.Context(), RefreshTokenInput{RefreshToken: rt})
		if err != nil {
			respondWithAuthError(c, err)
			return
		}
		if _, err := rctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				httpsRequiredUserMsg,
				errFailedToIssueCSRFCookie,
				"failed to issue refresh CSRF cookie",
			)
			return
		}
		if err := rctx.AuthCookies.Set(c, out.Token.Token, out.Token.RefreshToken); err != nil {
			if errors.Is(err, utils.ErrHTTPSRequired) {
				respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,
					httpsRequiredUserMsg,
					httpsRequiredLogMsg)
				return
			}
			respondWithAuthStatusError(c, http.StatusInternalServerError, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":              "Token refreshed",
			"user":                 out.Token.User,
			"accessTokenExpiresAt": out.Token.AccessTokenExpiresAt,
		})
	}
}

// handleCreateUser 解析四个必填字段，将进一步验证和创建交给用例，成功返回 201 公开账号。
func (r *Routes) handleCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.createUser.Execute(c.Request.Context(), CreateUserInput{
		Username: req.Username, Email: req.Email, Password: req.Password, Role: req.Role,
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out.User)
}

// handleGetUsers 返回公开账号数组，不将用户存储的密码摘要编码到响应。
func (r *Routes) handleGetUsers(c *gin.Context) {
	out, err := r.getUsers.Execute(c.Request.Context())
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Users)
}

// handleUpdateUser 从路由取目标标识，从上下文取实际请求者权限，避免请求体伪造管理员标记。
// 用户名与邮箱必填，密码与角色可省略，保留规则由更新用例处理。
func (r *Routes) handleUpdateUser(c *gin.Context) {
	id := c.Param("id")
	requester := authmw.MustGetUser(c)
	if requester == nil {
		return
	}
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.updateUser.Execute(c.Request.Context(), UpdateUserInput{
		ID: id, Username: req.Username, Email: req.Email, Password: req.Password, Role: req.Role,
		RequesterID: requester.ID, RequesterIsAdmin: requester.HasRole(coreauth.RoleAdmin),
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.User)
}

// handleDeleteUser 将路由标识交给删除用例，成功返回无响应体的 204。
func (r *Routes) handleDeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := r.deleteUser.Execute(c.Request.Context(), DeleteUserInput{ID: id}); err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleChangeOwnPassword 以当前用户标识提交新旧密码，不允许请求体选择其他账号。
// 当前刷新 Cookie 交给用例决定保留哪个持久化会话，读取失败仍继续改密流程。
func (r *Routes) handleChangeOwnPassword(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := authmw.MustGetUser(c)
		if user == nil {
			return
		}
		var req struct {
			OldPassword string `json:"oldPassword" binding:"required"`
			NewPassword string `json:"newPassword" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
			return
		}
		refreshToken, err := rctx.AuthCookies.ReadRefresh(c)
		if err != nil {
			// 空令牌使服务采用撤销全部刷新会话的回退，警告便于定位后续重新登录的原因。
			// 该操作不直接撤销访问 JWT，撤销失败的处理由用例负责。
			slog.Default().Warn("could not read refresh token while changing password; will revoke all sessions", "userID", user.ID, "error", err)
		}
		if err := r.changeOwnPassword.Execute(c.Request.Context(), ChangeOwnPasswordInput{
			UserID: user.ID, OldPassword: req.OldPassword, NewPassword: req.NewPassword, CurrentRefreshToken: refreshToken,
		}); err != nil {
			respondWithAuthError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// handleRequestPasswordReset 按原始标识检查独立额度，允许时提交异步重置请求。
// 标识额度耗尽时不调用用例，仍返回相同通用成功体，不透露是否执行发送。
func (r *Routes) handleRequestPasswordReset(identifierLimiter *security.KeyedLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Identifier string `json:"identifier" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
			return
		}

		// 额度按输入文本本身区分，这里不统一大小写或裁空白；耗尽只跳过发送入口。
		if identifierLimiter.Allow(req.Identifier) {
			if err := r.requestPasswordReset.Execute(c.Request.Context(), RequestPasswordResetInput{Identifier: req.Identifier}); err != nil {
				respondWithAuthError(c, err)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "If an account exists for that identifier, a password reset email has been sent.",
		})
	}
}

// handleConfirmPasswordReset 要求令牌和新密码，成功只返回公开用户，不建立登录 Cookie。
func (r *Routes) handleConfirmPasswordReset(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.confirmPasswordReset.Execute(c.Request.Context(), ConfirmPasswordResetInput{
		Token: req.Token, NewPassword: req.NewPassword,
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": out.User})
}

// handleInviteUser 返回 201 与账号和发送标记，邮件发送失败不等于账号创建失败。
func (r *Routes) handleInviteUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.inviteUser.Execute(c.Request.Context(), InviteUserInput{
		Username: req.Username, Email: req.Email, Role: req.Role,
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user":      out.User,
		"emailSent": out.EmailSent,
	})
}

// handleResendInvite 仅按现有账号标识重新发送，成功返回 204，失败按鉴权领域错误映射。
func (r *Routes) handleResendInvite(c *gin.Context) {
	id := c.Param("id")
	if err := r.resendInvite.Execute(c.Request.Context(), ResendInviteInput{UserID: id}); err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleConfirmInvite 完成邀请与会话签发后写入 CSRF 和认证 Cookie，使成功响应具备登录状态。
// Cookie 写入失败不撤销已经完成的邀请或会话，响应只返回用户与访问到期时刻。
func (r *Routes) handleConfirmInvite(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token       string `json:"token" binding:"required"`
			NewPassword string `json:"newPassword" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
			return
		}
		out, err := r.confirmInvite.Execute(c.Request.Context(), ConfirmInviteInput{
			Token: req.Token, NewPassword: req.NewPassword,
		})
		if err != nil {
			respondWithAuthError(c, err)
			return
		}
		if _, err := rctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				"HTTPS is required for login cookies. Use HTTPS or start LeafDoc with --allow-insecure for trusted plain HTTP setups.",
				errFailedToIssueCSRFCookie,
				"failed to issue invite-accept CSRF cookie",
			)
			return
		}
		if err := rctx.AuthCookies.Set(c, out.Token.Token, out.Token.RefreshToken); err != nil {
			if errors.Is(err, utils.ErrHTTPSRequired) {
				respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,
					httpsRequiredUserMsg,
					httpsRequiredLogMsg)
				return
			}
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":              "Invite accepted",
			"user":                 out.Token.User,
			"accessTokenExpiresAt": out.Token.AccessTokenExpiresAt,
		})
	}
}
