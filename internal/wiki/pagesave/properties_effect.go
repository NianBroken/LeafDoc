package pagesave

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/properties"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"log/slog"
)

// PropertiesSideEffect keeps the derived properties index in the page mutation transaction.
type PropertiesSideEffect struct {
	svc     *properties.PropertiesService
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

func NewPropertiesSideEffect(svc *properties.PropertiesService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *PropertiesSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &PropertiesSideEffect{svc: svc, log: log, metrics: metrics}
}
func (e *PropertiesSideEffect) Name() string { return "properties" }
func (e *PropertiesSideEffect) WithExecutor(exec database.Executor, _ *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec)
	}
	return &copy
}
func (e *PropertiesSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("page index update failed", "index", e.Name(), "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional propagates failure to the owner of the page transaction.
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
