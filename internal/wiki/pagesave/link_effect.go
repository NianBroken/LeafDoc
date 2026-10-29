package pagesave

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/links"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"log/slog"
	"strings"
)

// LinkIndexSideEffect keeps outgoing links and incoming link resolution consistent.
type LinkIndexSideEffect struct {
	svc     *links.LinkService
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

func NewLinkIndexSideEffect(svc *links.LinkService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *LinkIndexSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &LinkIndexSideEffect{svc: svc, log: log, metrics: metrics}
}
func (e *LinkIndexSideEffect) Name() string { return "links" }
func (e *LinkIndexSideEffect) WithExecutor(exec database.Executor, boundTree *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec, boundTree)
	}
	return &copy
}
func (e *LinkIndexSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("link index update failed", "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}
func (e *LinkIndexSideEffect) updateAndHeal(page *tree.Page) error {
	if page == nil {
		return nil
	}
	if err := e.svc.UpdateLinksForPage(page, page.Content); err != nil {
		return err
	}
	return e.svc.HealLinksForExactPath(page)
}
func (e *LinkIndexSideEffect) reconcileTitles(titles ...string) error {
	seen := map[string]bool{}
	for _, title := range titles {
		key := strings.ToLower(strings.TrimSpace(title))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if err := e.svc.ReconcileTitle(title); err != nil {
			return err
		}
	}
	return nil
}
func (e *LinkIndexSideEffect) markBrokenForOldPath(path string) error {
	if path == "" {
		return nil
	}
	return e.svc.MarkLinksBrokenForPrefix(path)
}

// ApplyTransactional returns the first failed link change so all page writes roll back.
func (e *LinkIndexSideEffect) ApplyTransactional(event PageSaveEvent) error {
	if e.svc == nil {
		return nil
	}
	switch event.Operation {
	case PageOperationCreate, PageOperationRestore:
		if err := e.updateAndHeal(event.After); err != nil {
			return err
		}
		if event.After != nil {
			return e.reconcileTitles(event.After.Title)
		}
	case PageOperationUpdate:
		if event.SlugChanged {
			if err := e.markBrokenForOldPath(event.OldPath); err != nil {
				return err
			}
			for _, page := range event.AffectedPages {
				if err := e.updateAndHeal(page); err != nil {
					return err
				}
			}
		} else if err := e.updateAndHeal(event.After); err != nil {
			return err
		}
		if event.TitleChanged && event.After != nil {
			return e.reconcileTitles(event.OldTitle, event.After.Title)
		}
	case PageOperationMove:
		if err := e.markBrokenForOldPath(event.OldPath); err != nil {
			return err
		}
		for _, page := range event.AffectedPages {
			if err := e.updateAndHeal(page); err != nil {
				return err
			}
		}
	case PageOperationDelete:
		for _, page := range event.AffectedPages {
			if err := e.svc.DeleteOutgoingLinksForPage(page.ID); err != nil {
				return err
			}
		}
		if event.Before != nil {
			if len(event.AffectedPages) > 1 {
				if err := e.markBrokenForOldPath(event.OldPath); err != nil {
					return err
				}
				for _, page := range event.AffectedPages {
					if err := e.svc.MarkIncomingLinksBrokenForPage(page.ID); err != nil {
						return err
					}
				}
			} else {
				if err := e.svc.MarkIncomingLinksBrokenForPage(event.Before.ID); err != nil {
					return err
				}
				if event.OldPath != "" {
					if err := e.svc.MarkLinksBrokenForPath(event.OldPath); err != nil {
						return err
					}
				}
			}
		}
		titles := []string{}
		for _, page := range event.AffectedPages {
			titles = append(titles, page.Title)
		}
		return e.reconcileTitles(titles...)
	}
	return nil
}
