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

// UpdatePageInput 提供页面身份、预期版本、目标标题路径段和可选正文元信息。
// Kind 保留在输入结构中，当前 Execute 不将其传入类型转换流程。
type UpdatePageInput struct {
	UserID              string
	ID                  string
	Version             string
	Title               string
	Slug                string
	Content             *string
	Kind                *tree.NodeKind
	Tags                []string
	Properties          map[string]string
	PreserveFrontmatter bool
}

// UpdatePageOutput 返回更新并完成保存事件处理后的页面正文与元信息。
type UpdatePageOutput struct {
	Page *tree.Page
}

// UpdatePageUseCase 在工作单元内保存页面，再按实际变化构造派生处理事件。
type UpdatePageUseCase struct {
	tree         *tree.TreeService
	slug         *tree.SlugService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
	metrics      *httpmetrics.HTTPMetrics
}

// NewUpdatePageUseCase 保存页面保存依赖，构造时不读取旧页面或校验版本。
func NewUpdatePageUseCase(
	t *tree.TreeService,
	s *tree.SlugService,
	o *pagesave.PageSaveOrchestrator,
	log *slog.Logger,
	metrics *httpmetrics.HTTPMetrics,
) *UpdatePageUseCase {
	return &UpdatePageUseCase{tree: t, slug: s, orchestrator: o, log: log, metrics: metrics}
}

// Execute 先验证非空标题和有效路径段，再读取旧页面并保存输入字段。
// 外层通过编排器绑定工作单元，页面保存或事件处理失败向工作单元传播。
// 正文变化以纯正文比较，只有附加元信息变化不会令 ContentChanged 为 true。
func (uc *UpdatePageUseCase) Execute(ctx context.Context, in UpdatePageInput) (out *UpdatePageOutput, err error) {
	if !uc.tree.InUnitOfWork() {
		var result *UpdatePageOutput
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
		uc.metrics.ObservePageSaveWorkflow(string(pagesave.PageOperationUpdate), err, started)
	}()

	ve := sharederrors.NewValidationErrors()
	if in.Title == "" {
		ve.Add("title", "Title must not be empty")
	}
	if err := uc.slug.IsValidSlug(in.Slug); err != nil {
		ve.Add("slug", err.Error())
	}
	if ve.HasErrors() {
		return nil, ve
	}

	in.Version = sanitizeClientVersion(in.Version)

	before, err := uc.tree.GetPage(in.ID)
	if err != nil {
		return nil, err
	}

	slugChanged := in.Slug != before.Slug
	oldPath := before.CalculatePath()
	// 旧页面持有树节点指针，提前保存标题与正文值，避免更新后失去比较依据。
	oldTitle := before.Title
	oldContent := before.Content

	// 路径段变化影响后代路由，先收集子树标识供保存后构造受影响列表。
	var subtreeIDs []string
	if slugChanged {
		subtreeIDs = collectSubtreeIDs(before.PageNode)
		if len(subtreeIDs) == 0 {
			subtreeIDs = []string{in.ID}
		}
	}

	if err = uc.tree.UpdateNode(tree.UpdateNodeInput{
		UserID:              in.UserID,
		ID:                  in.ID,
		Title:               in.Title,
		Slug:                in.Slug,
		Content:             in.Content,
		ExpectedVersion:     in.Version,
		Tags:                in.Tags,
		Properties:          in.Properties,
		PreserveFrontmatter: in.PreserveFrontmatter,
	}); err != nil {
		return nil, err
	}

	after, err := uc.tree.GetPage(in.ID)
	if err != nil {
		return nil, err
	}

	// 使用重新读取的结果判断正文和标题，路径变化已在写入前按输入比较。
	contentChanged := oldContent != after.Content
	titleChanged := oldTitle != after.Title

	event := pagesave.PageSaveEvent{
		Operation:      pagesave.PageOperationUpdate,
		UserID:         in.UserID,
		After:          after,
		OldPath:        oldPath,
		OldTitle:       oldTitle,
		ContentChanged: contentChanged,
		SlugChanged:    slugChanged,
		TitleChanged:   titleChanged,
	}

	// 子树读取失败只记录告警，其余成功页面继续进入事件处理。
	if slugChanged {
		pages, errs := uc.tree.GetPages(subtreeIDs)
		for i, p := range pages {
			if errs[i] != nil {
				uc.log.Warn("failed to get page for affected list", "pageID", subtreeIDs[i], "error", errs[i])
				continue
			}
			event.AffectedPages = append(event.AffectedPages, p)
		}
	}

	if err := uc.orchestrator.Run(event); err != nil {
		return nil, err
	}

	return &UpdatePageOutput{Page: after}, nil
}
