package apikeys

import (
	"net/http"
	"strings"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes 为管理员提供密钥创建、列表和撤销接口，不承担 Bearer 请求的密钥解析。
type Routes struct {
	createAPIKey *CreateAPIKeyUseCase
	listAPIKeys  *ListAPIKeysUseCase
	revokeAPIKey *RevokeAPIKeyUseCase
	authService  *coreauth.AuthService
}

// RoutesConfig 保存管理用例和普通登录鉴权服务。
type RoutesConfig struct {
	CreateAPIKey *CreateAPIKeyUseCase
	ListAPIKeys  *ListAPIKeysUseCase
	RevokeAPIKey *RevokeAPIKeyUseCase
	AuthService  *coreauth.AuthService
}

// NewRoutes 保存管理路由依赖，不依据功能开关在构造时删除用例。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		createAPIKey: cfg.CreateAPIKey,
		listAPIKeys:  cfg.ListAPIKeys,
		revokeAPIKey: cfg.RevokeAPIKey,
		authService:  cfg.AuthService,
	}
}

// RegisterRoutes 为三个接口共同要求登录、拒绝已解析 API 密钥标记并检查管理员角色。
// RequireCookieSession 只检查密钥标记，不自行读取 Cookie，因此上游代理身份仍按鉴权规则处理。
// 此处总会挂载三个路由，未配置密钥服务时由用例返回功能关闭错误。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	authGroup := ctx.APIAuthGroup(r.authService)

	authGroup.POST("/api-keys", authmw.RequireCookieSession(), authmw.RequireAdmin(), r.handleCreateAPIKey)
	authGroup.GET("/api-keys", authmw.RequireCookieSession(), authmw.RequireAdmin(), r.handleListAPIKeys)
	authGroup.DELETE("/api-keys/:id", authmw.RequireCookieSession(), authmw.RequireAdmin(), r.handleRevokeAPIKey)
}

// handleCreateAPIKey 要求名称与所属用户，按 RFC3339 解析非空到期时间，再交给用例验证。
// 创建者来自当前管理员，不接受请求体指定；成功响应含一次创建返回的明文 secret。
func (r *Routes) handleCreateAPIKey(c *gin.Context) {
	admin := authmw.MustGetUser(c)
	if admin == nil {
		return
	}

	var req struct {
		Name      string `json:"name" binding:"required"`
		UserID    string `json:"userId" binding:"required"`
		Role      string `json:"role"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyInvalidRequest, "Invalid request", "invalid request")
		return
	}

	var expiresAt *time.Time
	// 空白判断不改变解析输入，非空值仍用原字符串解析，两端多余空白不会被去掉。
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyInvalidExpiry, "Invalid expiry date", "invalid expiry date")
			return
		}
		expiresAt = &parsed
	}

	out, err := r.createAPIKey.Execute(c.Request.Context(), CreateAPIKeyInput{
		Name: req.Name, UserID: req.UserID, Role: req.Role, ExpiresAt: expiresAt, CreatedBy: admin.ID,
	})
	if err != nil {
		respondWithAPIKeyError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":    apiKeyResponse(out.Key),
		"secret": out.Secret,
	})
}

// handleListAPIKeys 将每条领域记录转换为公开字段，输出数组而非直接编码含哈希的结构体。
func (r *Routes) handleListAPIKeys(c *gin.Context) {
	out, err := r.listAPIKeys.Execute(c.Request.Context())
	if err != nil {
		respondWithAPIKeyError(c, err)
		return
	}
	keys := make([]gin.H, len(out.Keys))
	for i, k := range out.Keys {
		keys[i] = apiKeyResponse(k)
	}
	c.JSON(http.StatusOK, keys)
}

// handleRevokeAPIKey 按 URL 记录标识撤销，成功返回无响应体的 204。
func (r *Routes) handleRevokeAPIKey(c *gin.Context) {
	id := c.Param("id")
	if err := r.revokeAPIKey.Execute(c.Request.Context(), RevokeAPIKeyInput{ID: id}); err != nil {
		respondWithAPIKeyError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// apiKeyResponse 明确选择可公开字段，不包含密钥哈希或明文令牌。
// 时间统一转为 UTC RFC3339 文本，可选时间指针为空时省略相应字段。
// 调用方必须传入非空记录。
func apiKeyResponse(key *coreauth.APIKey) gin.H {
	resp := gin.H{
		"id":        key.ID,
		"name":      key.Name,
		"userId":    key.UserID,
		"prefix":    key.Prefix,
		"role":      key.Role,
		"createdBy": key.CreatedBy,
		"createdAt": key.CreatedAt.UTC().Format(time.RFC3339),
	}
	if key.ExpiresAt != nil {
		resp["expiresAt"] = key.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if key.LastUsedAt != nil {
		resp["lastUsedAt"] = key.LastUsedAt.UTC().Format(time.RFC3339)
	}
	if key.RevokedAt != nil {
		resp["revokedAt"] = key.RevokedAt.UTC().Format(time.RFC3339)
	}
	return resp
}
