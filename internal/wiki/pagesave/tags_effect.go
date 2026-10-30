package pagesave

import (
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// TagsSideEffect keeps the derived tags index in the page mutation transaction.
type TagsSideEffect struct {
	svc     *tags.TagsService
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

func NewTagsSideEffect(svc *tags.TagsService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *TagsSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &TagsSideEffect{svc: svc, log: log, metrics: metrics}
}
func (e *TagsSideEffect) Name() string { return "tags" }
func (e *TagsSideEffect) WithExecutor(exec database.Executor, _ *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec)
	}
	return &copy
}
func (e *TagsSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("page index update failed", "index", e.Name(), "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional propagates failure to the owner of the page transaction.
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
