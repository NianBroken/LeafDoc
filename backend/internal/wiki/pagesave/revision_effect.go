package pagesave

import (
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// RevisionSideEffect 按页面事件选择正文历史或结构历史入口。
// 绑定执行器时历史元数据与页面使用同一工作单元，资源处理仍沿用历史服务规则。
type RevisionSideEffect struct {
	svc     *revision.Service
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

// NewRevisionSideEffect 保存历史服务，空日志器使用默认值，不在构造时捕获页面。
func NewRevisionSideEffect(svc *revision.Service, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *RevisionSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &RevisionSideEffect{svc: svc, log: log, metrics: metrics}
}

// Name 为历史处理提供固定指标名称。
func (e *RevisionSideEffect) Name() string { return "revision" }

// WithExecutor 复制处理器并让历史服务使用事务执行器与绑定树视图。
func (e *RevisionSideEffect) WithExecutor(exec database.Executor, boundTree *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec, boundTree)
	}
	return &copy
}

// Apply 对历史失败记录日志与指标，不向调用者返回错误。
func (e *RevisionSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("revision recording failed", "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional 在创建时记录正文，移动时逐页记录结构状态。
// 更新路径段时遍历受影响页面，正文也变化的目标页留给正文历史分支处理。
// 只有标题变化且正文未变时记录目标页结构，正文变化则使用事件 Summary。
// 恢复和删除事件不在此记录历史，首个历史服务错误直接返回。
func (e *RevisionSideEffect) ApplyTransactional(event PageSaveEvent) error {
	if e.svc == nil {
		return nil
	}
	structure := func(id string) error { _, _, err := e.svc.RecordStructureChange(id, event.UserID, ""); return err }
	content := func(id string) error {
		_, _, err := e.svc.RecordContentUpdate(id, event.UserID, event.Summary)
		return err
	}
	switch event.Operation {
	case PageOperationCreate:
		if event.After != nil {
			return content(event.After.ID)
		}
	case PageOperationUpdate:
		if event.SlugChanged {
			for _, page := range event.AffectedPages {
				if event.ContentChanged && event.After != nil && page.ID == event.After.ID {
					continue
				}
				if err := structure(page.ID); err != nil {
					return err
				}
			}
		} else if event.TitleChanged && !event.ContentChanged && event.After != nil {
			if err := structure(event.After.ID); err != nil {
				return err
			}
		}
		if event.ContentChanged && event.After != nil {
			return content(event.After.ID)
		}
	case PageOperationMove:
		for _, page := range event.AffectedPages {
			if err := structure(page.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
