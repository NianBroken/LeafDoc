package pagesave

import (
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// TagsSideEffect 从页面完整 Markdown 提取标签，或在删除事件中清理标签索引。
// 经 WithExecutor 绑定后复用页面工作单元的数据库执行器。
type TagsSideEffect struct {
	svc     *tags.TagsService
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

// NewTagsSideEffect 保存依赖，空日志器使用默认值，空服务允许事件处理直接跳过。
func NewTagsSideEffect(svc *tags.TagsService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *TagsSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &TagsSideEffect{svc: svc, log: log, metrics: metrics}
}

// Name 提供日志和指标中使用的固定标签索引名称。
func (e *TagsSideEffect) Name() string { return "tags" }

// WithExecutor 复制处理器并绑定标签服务，不改动原处理器或依赖树参数。
func (e *TagsSideEffect) WithExecutor(exec database.Executor, _ *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec)
	}
	return &copy
}

// Apply 将错误记录到日志和失败计数，没有向调用方返回错误的通道。
func (e *TagsSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("page index update failed", "index", e.Name(), "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional 对创建、更新和恢复事件索引 After.RawContent 中的标签。
// 删除事件批量清理 AffectedPages 的标识，移动事件不改变标签索引。
// 返回的错误由拥有事务的上层处理，此方法本身不提交或回滚。
func (e *TagsSideEffect) ApplyTransactional(event PageSaveEvent) error {
	if e.svc == nil {
		return nil
	}
	switch event.Operation {
	case PageOperationCreate, PageOperationUpdate, PageOperationRestore:
		if event.After != nil {
			return e.svc.IndexPageContent(event.After.ID, event.After.RawContent)
		}
	case PageOperationDelete:
		ids := []string{}
		for _, page := range event.AffectedPages {
			ids = append(ids, page.ID)
		}
		return e.svc.DeletePageIndexes(ids)
	}
	return nil
}
