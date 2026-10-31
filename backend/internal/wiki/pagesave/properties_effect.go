package pagesave

import (
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/properties"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// PropertiesSideEffect 将完整 Markdown 提取出的属性写入派生索引，并处理页面删除。
// 经 WithExecutor 绑定后复用页面工作单元的数据库执行器。
type PropertiesSideEffect struct {
	svc     *properties.PropertiesService
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

// NewPropertiesSideEffect 保存属性服务和指标，空日志器使用默认值。
func NewPropertiesSideEffect(svc *properties.PropertiesService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *PropertiesSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &PropertiesSideEffect{svc: svc, log: log, metrics: metrics}
}

// Name 提供属性处理器的固定指标名称。
func (e *PropertiesSideEffect) Name() string { return "properties" }

// WithExecutor 复制处理器并绑定属性服务，原实例继续使用原执行器。
func (e *PropertiesSideEffect) WithExecutor(exec database.Executor, _ *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec)
	}
	return &copy
}

// Apply 记录属性处理错误并递增失败计数，不将错误返回到调用方。
func (e *PropertiesSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("page index update failed", "index", e.Name(), "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional 在创建、更新和恢复时提取 After.RawContent 的属性并整体保存。
// 删除时按列表逐页清理，首个错误立即返回；移动和空服务不执行属性写入。
func (e *PropertiesSideEffect) ApplyTransactional(event PageSaveEvent) error {
	if e.svc == nil {
		return nil
	}
	switch event.Operation {
	case PageOperationCreate, PageOperationUpdate, PageOperationRestore:
		if event.After != nil {
			return e.svc.SetPropertiesForPage(event.After.ID, properties.ExtractPropertiesFromContent(event.After.RawContent))
		}
	case PageOperationDelete:
		for _, page := range event.AffectedPages {
			if err := e.svc.DeletePropertiesForPage(page.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
