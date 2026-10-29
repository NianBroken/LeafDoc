package pagesave

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"log/slog"
)

// RevisionSideEffect records history in the same transaction as the page body.
type RevisionSideEffect struct {
	svc     *revision.Service
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

func NewRevisionSideEffect(svc *revision.Service, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *RevisionSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &RevisionSideEffect{svc: svc, log: log, metrics: metrics}
}
func (e *RevisionSideEffect) Name() string { return "revision" }
func (e *RevisionSideEffect) WithExecutor(exec database.Executor, boundTree *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec, boundTree)
	}
	return &copy
}
func (e *RevisionSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("revision recording failed", "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional leaves restoration history to RestoreRevision itself.
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
