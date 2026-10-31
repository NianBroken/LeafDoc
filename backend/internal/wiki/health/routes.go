package health

import (
	"net/http"

	"github.com/NianBroken/LeafDoc/internal/core/search"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/gin-gonic/gin"
)

// Routes 将健康用例暴露为基础路由下的 GET 接口，不另外挂载登录权限。
type Routes struct {
	health *HealthUseCase
}

// RoutesConfig 提供数据库连通性、搜索状态和存储目录检查所需的现有对象。
type RoutesConfig struct {
	Index      *search.MySQLIndex
	Status     *search.IndexingStatus
	StorageDir string
}

// NewRoutes 保存健康用例，构造时不执行检查，检查在每个请求中发生。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		health: NewHealthUseCase(cfg.Index, cfg.Status, cfg.StorageDir),
	}
}

// RegisterRoutes 将接口挂在共同基础路径下，沿用基础组已经配置的中间件。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	ctx.Base.GET("/api/health", r.handleHealth)
}

// handleHealth 将用例汇总结果映射为 200 或 503，并始终返回各项检查状态。
// 搜索仍在建立索引时不直接降为故障，是否健康由用例的明确条件计算。
func (r *Routes) handleHealth(c *gin.Context) {
	healthy, checks := r.health.Execute()

	status := "ok"
	code := http.StatusOK
	if !healthy {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"status": status,
		"checks": checks,
	})
}
