package pagesave

import (
	"context"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/tree"

	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// PageSideEffect 定义页面事件处理入口，普通 Apply 同步执行且没有错误返回值。
// 实现同时提供 ApplyTransactional 时，编排器优先使用可返回错误的入口。
type PageSideEffect interface {
	Apply(event PageSaveEvent)
}

// namedPageSideEffect 为指标提供处理器名称，没有名称接口的处理器使用 unknown。
type namedPageSideEffect interface {
	Name() string
}

// PageSaveOrchestrator 按注册顺序执行页面事件处理器，串联历史和派生索引写入。
// 事务入口的错误会停止后续处理器，普通 Apply 的内部错误不通过此接口传播。
type PageSaveOrchestrator struct {
	metrics     *httpmetrics.HTTPMetrics
	sideEffects []PageSideEffect
}

// NewPageSaveOrchestrator 保存指标与处理器列表，不立即执行或复制每个处理器。
func NewPageSaveOrchestrator(metrics *httpmetrics.HTTPMetrics, effects ...PageSideEffect) *PageSaveOrchestrator {
	return &PageSaveOrchestrator{metrics: metrics, sideEffects: effects}
}

// Run 对空编排器直接成功，其余情况逐个同步处理事件。
// 可返回错误的处理器失败时立即返回，该次失败不会经过后面的耗时记录语句。
func (o *PageSaveOrchestrator) Run(event PageSaveEvent) error {
	if o == nil {
		return nil
	}
	for _, se := range o.sideEffects {
		started := time.Now()
		if effect, ok := se.(interface{ ApplyTransactional(PageSaveEvent) error }); ok {
			if err := effect.ApplyTransactional(event); err != nil {
				return err
			}
		} else {
			se.Apply(event)
		}
		o.metrics.ObservePageSaveSideEffect(string(event.Operation), sideEffectName(se), started)
	}
	return nil
}

// sideEffectName 获取可选的固定名称，不根据类型名称或日志内容推断。
func sideEffectName(effect PageSideEffect) string {
	if named, ok := effect.(namedPageSideEffect); ok {
		return named.Name()
	}
	return "unknown"
}

// WithExecutor 为支持绑定接口的处理器建立执行器视图，其余处理器复用原对象。
// 返回新的编排器列表，不改变原编排器；具体数据库绑定由各处理器负责。
func (o *PageSaveOrchestrator) WithExecutor(exec database.Executor, t *tree.TreeService) *PageSaveOrchestrator {
	if o == nil {
		return nil
	}
	effects := make([]PageSideEffect, 0, len(o.sideEffects))
	for _, effect := range o.sideEffects {
		if binder, ok := effect.(interface {
			WithExecutor(database.Executor, *tree.TreeService) PageSideEffect
		}); ok {
			effects = append(effects, binder.WithExecutor(exec, t))
		} else {
			effects = append(effects, effect)
		}
	}
	return NewPageSaveOrchestrator(o.metrics, effects...)
}

// Execute 通过树服务工作单元把绑定树、绑定编排器与执行器交给回调。
// 回调失败由工作单元回滚，成功提交后才向原树服务发布内存层级。
func (o *PageSaveOrchestrator) Execute(ctx context.Context, t *tree.TreeService, fn func(*tree.TreeService, *PageSaveOrchestrator, database.Executor) error) error {
	return t.WithUnitOfWork(ctx, func(bound *tree.TreeService, exec database.Executor) error {
		return fn(bound, o.WithExecutor(exec, bound), exec)
	})
}
