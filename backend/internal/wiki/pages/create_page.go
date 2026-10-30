package pages

import (
	"context"
	"log/slog"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// CreatePageInput is the input for CreatePageUseCase.
type CreatePageInput struct {
	UserID   string
	ParentID *string
	Title    string
	Slug     string
	Kind     *tree.NodeKind
}

// CreatePageOutput is the output of CreatePageUseCase.
type CreatePageOutput struct {
	Page *tree.Page
}

// CreatePageUseCase creates a new page in the tree and fires post-save side effects.
type CreatePageUseCase struct {
	tree         *tree.TreeService
	slug         *tree.SlugService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
	metrics      *httpmetrics.HTTPMetrics
}

// NewCreatePageUseCase constructs a CreatePageUseCase.
func NewCreatePageUseCase(
	t *tree.TreeService,
	s *tree.SlugService,
	o *pagesave.PageSaveOrchestrator,
	log *slog.Logger,
	metrics *httpmetrics.HTTPMetrics,
) *CreatePageUseCase {
	return &CreatePageUseCase{tree: t, slug: s, orchestrator: o, log: log, metrics: metrics}
}

// Execute validates input, creates the page node, and fires post-save side effects.
func (uc *CreatePageUseCase) Execute(ctx context.Context, in CreatePageInput) (out *CreatePageOutput, err error) {
	if !uc.tree.InUnitOfWork() {
		var result *CreatePageOutput
		err := uc.orchestrator.Execute(ctx, uc.tree, func(bound *tree.TreeService, o *pagesave.PageSaveOrchestrator, exec database.Executor) error {
			copy := *uc
			copy.tree = bound
			copy.orchestrator = o
			var inner error
			result, inner = copy.Execute(ctx, in)
			return inner
		})
		return result, err
	}

	started := time.Now()
	defer func() {
		uc.metrics.ObservePageSaveWorkflow(string(pagesave.PageOperationCreate), err, started)
	}()

	ve := sharederrors.NewValidationErrors()

	if in.Title == "" {
		ve.Add("title", "Title must not be empty")
	}
	if in.Kind == nil {
		ve.Add("kind", "Kind must be specified")
	}
	if in.Kind != nil && *in.Kind != tree.NodeKindPage && *in.Kind != tree.NodeKindSection {
		ve.Add("kind", "Kind must be either 'page' or 'section'")
	}
	if err := uc.slug.IsValidSlug(in.Slug); err != nil {
		ve.Add("slug", err.Error())
	}
	if ve.HasErrors() {
		return nil, ve
	}

	if in.ParentID != nil && *in.ParentID != "" {
		if _, err := uc.tree.FindPageByID(*in.ParentID); err != nil {
			return nil, err
		}
	}

	id, err := uc.tree.CreateNode(in.UserID, in.ParentID, in.Title, in.Slug, in.Kind)
	if err != nil {
		return nil, err
	}

	page, err := uc.tree.GetPage(*id)
	if err != nil {
		return nil, err
	}

	if err := uc.orchestrator.Run(pagesave.PageSaveEvent{
		Operation: pagesave.PageOperationCreate,
		UserID:    in.UserID,
		After:     page,
		Summary:   "page created",
	}); err != nil {
		return nil, err
	}

	return &CreatePageOutput{Page: page}, nil
}
