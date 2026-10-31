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

// CreatePageInput 提供创建用户、父标识、标题、路径段和明确节点类型。
// 标题只检查空字符串，路径段交给统一名称验证器检查。
type CreatePageInput struct {
	UserID   string
	ParentID *string
	Title    string
	Slug     string
	Kind     *tree.NodeKind
}

// CreatePageOutput 返回完成页面保存流程的页面对象及初始正文。
type CreatePageOutput struct {
	Page *tree.Page
}

// CreatePageUseCase 验证输入、建立树节点并向页面保存编排器发送创建事件。
// 支持绑定执行器的派生处理与页面保存共用外层工作单元。
type CreatePageUseCase struct {
	tree         *tree.TreeService
	slug         *tree.SlugService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
	metrics      *httpmetrics.HTTPMetrics
}

// NewCreatePageUseCase 保存树、名称规则、编排器、日志和指标依赖，不执行页面操作。
func NewCreatePageUseCase(
	t *tree.TreeService,
	s *tree.SlugService,
	o *pagesave.PageSaveOrchestrator,
	log *slog.Logger,
	metrics *httpmetrics.HTTPMetrics,
) *CreatePageUseCase {
	return &CreatePageUseCase{tree: t, slug: s, orchestrator: o, log: log, metrics: metrics}
}

// Execute 未绑定工作单元时先复制用例并绑定事务树，再执行验证与保存。
// 字段校验错误汇总返回，读取或事件处理错误使外层工作单元失败。
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

	// 在实际工作单元内统计一次保存耗时，命名返回值提供最终错误状态。
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

	// 显式非空父标识必须可查询，创建入口随后检查父类型与同级冲突。
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

	// 事件使用刚读取的页面作为 After，派生处理按编排器注册顺序执行。
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
