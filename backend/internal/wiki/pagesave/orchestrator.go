package pagesave

import (
	"context"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/tree"

	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// PageSideEffect is implemented by any component that reacts to a page mutation.
// Apply is always called synchronously and errors are handled internally (best-effort).
type PageSideEffect interface {
	Apply(event PageSaveEvent)
}

type namedPageSideEffect interface {
	Name() string
}

// PageSaveOrchestrator fans out a PageSaveEvent to all registered side effects.
type PageSaveOrchestrator struct {
	metrics     *httpmetrics.HTTPMetrics
	sideEffects []PageSideEffect
}

// NewPageSaveOrchestrator creates an orchestrator with the given side effects.
func NewPageSaveOrchestrator(metrics *httpmetrics.HTTPMetrics, effects ...PageSideEffect) *PageSaveOrchestrator {
	return &PageSaveOrchestrator{metrics: metrics, sideEffects: effects}
}

// Run delivers the event to each side effect in registration order.
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

func sideEffectName(effect PageSideEffect) string {
	if named, ok := effect.(namedPageSideEffect); ok {
		return named.Name()
	}
	return "unknown"
}

// WithExecutor binds every database effect to one request transaction.
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

// Execute commits page data, history and indexes together before publishing the tree.
func (o *PageSaveOrchestrator) Execute(ctx context.Context, t *tree.TreeService, fn func(*tree.TreeService, *PageSaveOrchestrator, database.Executor) error) error {
	return t.WithUnitOfWork(ctx, func(bound *tree.TreeService, exec database.Executor) error {
		return fn(bound, o.WithExecutor(exec, bound), exec)
	})
}
