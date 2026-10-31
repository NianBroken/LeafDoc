// 本文件注册管理员索引重建入口与任务状态轮询入口。
package wikiresync

import (
	"net/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes 将数据库页面树及相关索引的重建用例接入管理员接口。
type Routes struct {
	triggerUC   *TriggerResyncUseCase
	statusUC    *GetResyncStatusUseCase
	authService *coreauth.AuthService
}

// NewRoutes 保存触发、查询和身份认证依赖，路由在注册阶段绑定。
func NewRoutes(triggerUC *TriggerResyncUseCase, statusUC *GetResyncStatusUseCase, authService *coreauth.AuthService) *Routes {
	return &Routes{
		triggerUC:   triggerUC,
		statusUC:    statusUC,
		authService: authService,
	}
}

// RegisterRoutes 在已认证 API 分组下设置管理员限制，普通用户不能触发或查询。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {

	authGroup := ctx.APIAuthGroup(r.authService)

	adminGroup := authGroup.Group("/admin")
	adminGroup.Use(authmw.RequireAdmin())

	adminGroup.POST("/resync", r.handleTriggerResync)
	adminGroup.GET("/resync/status", r.handleResyncStatus)
}

// handleTriggerResync 在启动回调返回后响应 202 和 ok 标记，不等待后台重建结束。
// 已有任务运行时响应 409，客户端继续通过状态接口查看任务结果。
func (r *Routes) handleTriggerResync(c *gin.Context) {
	if err := r.triggerUC.Execute(c.Request.Context()); err != nil {
		respondWithResyncError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// handleResyncStatus 返回当前任务快照，成功、失败和执行中均通过字段表示并响应 200。
func (r *Routes) handleResyncStatus(c *gin.Context) {
	out := r.statusUC.Execute(c.Request.Context())
	c.JSON(http.StatusOK, out.Status)
}
