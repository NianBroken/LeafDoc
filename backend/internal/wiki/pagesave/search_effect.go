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

// SearchIndexSideEffect 按事件写入或删除搜索索引，并提供从当前页面树重建的入口。
// 页面事件只使用事件携带的数据，全量重建才遍历树服务。
type SearchIndexSideEffect struct {
	index   *search.MySQLIndex
	tree    *tree.TreeService // 全量重建时收集页面标识和读取页面正文。
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

// NewSearchIndexSideEffect 保存索引、树和指标依赖，空日志器使用默认值。
func NewSearchIndexSideEffect(index *search.MySQLIndex, treeService *tree.TreeService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *SearchIndexSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &SearchIndexSideEffect{index: index, tree: treeService, log: log, metrics: metrics}
}

// Name 返回搜索处理器的固定指标名称。
func (e *SearchIndexSideEffect) Name() string {
	return "search"
}

// WithExecutor 在副本中绑定索引执行器和树视图，不改变原处理器。
func (e *SearchIndexSideEffect) WithExecutor(exec database.Executor, boundTree *tree.TreeService) PageSideEffect {
	copy := *e
	copy.tree = boundTree
	if e.index != nil {
		copy.index = e.index.WithExecutor(exec)
	}
	return &copy
}

// Apply 将索引错误记录到日志和失败计数，不向普通接口调用者返回错误。
func (e *SearchIndexSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("search index update failed", "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// ApplyTransactional 对创建、更新和恢复索引单页，对移动批量重写受影响页面。
// 批量接口的整体错误直接返回，单页失败列表合并后返回；删除按标识批量移除。
// 空索引表示未配置搜索，所有事件直接成功，不提交或回滚外部事务。
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

// buildIndexInput 将当前路由去掉一个前导斜杠作为路径，非空路径加 .md 形成索引文件名。
// 正文使用 RawContent，包含页首元信息；文件名是索引字段，不在此读取磁盘。
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

// IndexAllPages 使用后台上下文执行全量清空和重建，不在此限定调用次数。
func (e *SearchIndexSideEffect) IndexAllPages() error {
	return e.IndexAllPagesContext(context.Background())
}

// IndexAllPagesContext 先清空旧索引，再遍历树、批量读取页面并逐个重新索引。
// 取消、读取或索引失败立即返回，已清空及已经完成的部分不在此恢复。
// 上下文在多个阶段检查，不直接传给索引 Clear 或单页索引调用。
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
