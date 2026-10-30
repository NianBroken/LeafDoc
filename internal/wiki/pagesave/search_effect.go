package pagesave

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// SearchIndexSideEffect updates the search index after every page mutation.
type SearchIndexSideEffect struct {
	index   *search.MySQLIndex
	tree    *tree.TreeService // only used by IndexAllPages for the initial walk
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

func NewSearchIndexSideEffect(index *search.MySQLIndex, treeService *tree.TreeService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *SearchIndexSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &SearchIndexSideEffect{index: index, tree: treeService, log: log, metrics: metrics}
}

func (e *SearchIndexSideEffect) Name() string {
	return "search"
}

func (e *SearchIndexSideEffect) WithExecutor(exec database.Executor, boundTree *tree.TreeService) PageSideEffect {
	copy := *e
	copy.tree = boundTree
	if e.index != nil {
		copy.index = e.index.WithExecutor(exec)
	}
	return &copy
}
func (e *SearchIndexSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("search index update failed", "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional propagates indexing failures to the page transaction.
func (e *SearchIndexSideEffect) ApplyTransactional(event PageSaveEvent) error {
	if e.index == nil {
		return nil
	}
	switch event.Operation {
	case PageOperationCreate, PageOperationUpdate, PageOperationRestore:
		if event.After != nil {
			input := buildIndexInput(event.After)
			return e.index.IndexPage(input.Path, input.FilePath, input.PageID, input.Title, input.Kind, input.Raw)
		}
	case PageOperationMove:
		inputs := []search.IndexPageInput{}
		for _, page := range event.AffectedPages {
			inputs = append(inputs, buildIndexInput(page))
		}
		failures, err := e.index.IndexPages(inputs)
		if err != nil {
			return err
		}
		causes := []error{}
		for _, failure := range failures {
			causes = append(causes, failure.Err)
		}
		return errors.Join(causes...)
	case PageOperationDelete:
		ids := []string{}
		for _, page := range event.AffectedPages {
			ids = append(ids, page.ID)
		}
		return e.index.RemovePages(ids)
	}
	return nil
}

// buildIndexInput computes the (path, filePath) pair IndexPage/IndexPages
// need for page, mirroring writeToIndex's single-page logic.
func buildIndexInput(page *tree.Page) search.IndexPageInput {
	path := strings.TrimPrefix(page.CalculatePath(), "/")
	filePath := path
	if filePath != "" {
		filePath += ".md"
	}
	return search.IndexPageInput{
		Path: path, FilePath: filePath, PageID: page.ID, Title: page.Title, Kind: page.Kind, Raw: page.RawContent,
	}
}

// IndexAllPages clears the search index and rebuilds it from the current tree state.
// Call this once at startup; runtime updates are handled via Apply.
func (e *SearchIndexSideEffect) IndexAllPages() error {
	return e.IndexAllPagesContext(context.Background())
}

func (e *SearchIndexSideEffect) IndexAllPagesContext(ctx context.Context) error {
	if e.index == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := e.index.Clear(); err != nil {
		return err
	}

	var ids []string
	if err := e.tree.WalkNodes(func(id string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ids = append(ids, id)
		return nil
	}); err != nil {
		return err
	}

	pages, errs := e.tree.GetPages(ids)
	for i, page := range pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if errs[i] != nil {
			return errs[i]
		}
		if err := e.ApplyTransactional(PageSaveEvent{Operation: PageOperationCreate, After: page}); err != nil {
			return err
		}
	}
	return nil
}
