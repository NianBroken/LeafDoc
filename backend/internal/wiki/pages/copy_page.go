package pages

import (
	"context"
	"log/slog"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// CopyPageInput 指定来源页面及目标父标识、标题和路径段，不携带来源版本约束。
type CopyPageInput struct {
	UserID         string
	SourcePageID   string
	TargetParentID *string
	Title          string
	Slug           string
}

// CopyPageOutput 返回已保存复制正文并完成创建事件处理的目标页面。
type CopyPageOutput struct {
	Page *tree.Page
}

// CopyPageUseCase 将来源纯正文和附件复制到一个普通页面，不复制来源子树或节点类型。
// 来源自定义页首字段不随纯正文复制，目标使用其自身节点管理字段。
type CopyPageUseCase struct {
	tree         *tree.TreeService
	slug         *tree.SlugService
	assets       *assets.AssetService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
}

// NewCopyPageUseCase 保存页面、附件和保存编排依赖，不执行复制。
func NewCopyPageUseCase(
	t *tree.TreeService,
	s *tree.SlugService,
	o *pagesave.PageSaveOrchestrator,
	a *assets.AssetService,
	log *slog.Logger,
) *CopyPageUseCase {
	return &CopyPageUseCase{tree: t, slug: s, assets: a, orchestrator: o, log: log}
}

// Execute 在页面工作单元内验证目标名称，读取来源后建立普通页面并复制附件。
// 正文中的来源附件地址前缀通过字符串替换改为目标标识，不解析具体 Markdown 语法。
// 部分失败分支尝试删除目标节点，清理错误被忽略；附件磁盘操作不随数据库自动回滚。
func (uc *CopyPageUseCase) Execute(ctx context.Context, in CopyPageInput) (*CopyPageOutput, error) {
	if !uc.tree.InUnitOfWork() {
		var result *CopyPageOutput
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

	page, err := uc.tree.GetPage(in.SourcePageID)
	if err != nil {
		return nil, err
	}

	kind := tree.NodeKindPage
	copyID, err := uc.tree.CreateNode(in.UserID, in.TargetParentID, in.Title, in.Slug, &kind)
	if err != nil {
		return nil, err
	}
	// 清理使用内部跳过版本标记，不向外暴露；此回调只删除页面记录，不删除附件。
	cleanup := func() { _ = uc.tree.DeleteNode(in.UserID, *copyID, false, tree.VersionUnchecked) }

	copyPage, err := uc.tree.GetPage(*copyID)
	if err != nil {
		cleanup()
		return nil, err
	}

	if err := uc.assets.CopyAllAssets(page.PageNode, copyPage.PageNode); err != nil {
		cleanup()
		return nil, err
	}

	// 仅替换完整附件路径前缀，其他相对地址、页面链接和代码示例中的匹配文字沿用字符串规则。
	updatedContent := strings.ReplaceAll(page.Content, "/assets/"+page.ID+"/", "/assets/"+copyPage.ID+"/")
	if err := uc.tree.UpdateNode(tree.UpdateNodeInput{
		UserID:          in.UserID,
		ID:              copyPage.ID,
		Title:           copyPage.Title,
		Slug:            copyPage.Slug,
		Content:         &updatedContent,
		ExpectedVersion: tree.VersionUnchecked,
	}); err != nil {
		cleanup()
		_ = uc.assets.DeleteAllAssetsForPage(copyPage.PageNode)
		return nil, err
	}

	// 写入正文后重新读取，创建事件取得的是最终内容而不是初始标题正文。
	copyPage, err = uc.tree.GetPage(copyPage.ID)
	if err != nil {
		return nil, err
	}

	if err := uc.orchestrator.Run(pagesave.PageSaveEvent{
		Operation: pagesave.PageOperationCreate,
		UserID:    in.UserID,
		After:     copyPage,
		Summary:   "page copied",
	}); err != nil {
		return nil, err
	}

	return &CopyPageOutput{Page: copyPage}, nil
}
