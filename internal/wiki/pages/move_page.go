package pages

import (
	"context"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"log/slog"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// MovePageInput is the input for MovePageUseCase.
type MovePageInput struct {
	UserID   string
	ID       string
	Version  string
	ParentID string
	// Position is the target index among the new parent's children; nil appends at the end.
	Position *int
}

// MovePageUseCase moves a page to a new parent, updating links and recording revisions.
type MovePageUseCase struct {
	tree         *tree.TreeService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
	metrics      *httpmetrics.HTTPMetrics
}

// NewMovePageUseCase constructs a MovePageUseCase.
func NewMovePageUseCase(
	t *tree.TreeService,
	o *pagesave.PageSaveOrchestrator,
	log *slog.Logger,
	metrics *httpmetrics.HTTPMetrics,
) *MovePageUseCase {
	return &MovePageUseCase{tree: t, orchestrator: o, log: log, metrics: metrics}
}

// Execute moves the page and fires post-save side effects for the whole subtree.
func (uc *MovePageUseCase) Execute(ctx context.Context, in MovePageInput) (err error) {
	if !uc.tree.InUnitOfWork() {
		err := uc.orchestrator.Execute(ctx, uc.tree, func(bound *tree.TreeService, o *pagesave.PageSaveOrchestrator, exec database.Executor) error {
			copy := *uc
			copy.tree = bound
			copy.orchestrator = o
			return copy.Execute(ctx, in)
		})
		return err
	}

	started := time.Now()
	defer func() {
		uc.metrics.ObservePageSaveWorkflow(string(pagesave.PageOperationMove), err, started)
	}()

	if in.ID == "root" || in.ID == "" {
		return newPageRootOperationError("move")
	}

	in.Version = sanitizeClientVersion(in.Version)

	var subtreeIDs []string
	var beforePage *tree.Page

	if uc.tree.IsLoaded() {
		if node, err := uc.tree.FindPageByID(in.ID); err == nil && node != nil {
			subtreeIDs = collectSubtreeIDs(node)
			if p, err := uc.tree.GetPage(in.ID); err == nil {
				beforePage = p
			}
		}
	}
	if len(subtreeIDs) == 0 {
		subtreeIDs = []string{in.ID}
	}
	if beforePage == nil {
		p, err := uc.tree.GetPage(in.ID)
		if err != nil {
			return err
		}
		beforePage = p
	}

	var oldPath string
	if beforePage != nil {
		oldPath = beforePage.CalculatePath()
	}

	position := -1
	if in.Position != nil {
		position = *in.Position
	}
	if err = uc.tree.MoveNodeToPosition(in.UserID, in.ID, in.ParentID, in.Version, position); err != nil {
		return err
	}

	event := pagesave.PageSaveEvent{
		Operation: pagesave.PageOperationMove,
		UserID:    in.UserID,
		OldPath:   oldPath,
	}

	pages, errs := uc.tree.GetPages(subtreeIDs)
	for i, p := range pages {
		if errs[i] != nil {
			uc.log.Warn("failed to get page after move", "pageID", subtreeIDs[i], "error", errs[i])
			continue
		}
		event.AffectedPages = append(event.AffectedPages, p)
	}

	if err := uc.orchestrator.Run(event); err != nil {
		return err
	}

	return nil
}
