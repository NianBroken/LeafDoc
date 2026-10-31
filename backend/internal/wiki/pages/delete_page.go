package pages

import (
	"context"
	"log/slog"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// DeletePageInput 指定操作用户、页面标识、预期版本和是否允许删除子树。
type DeletePageInput struct {
	UserID    string
	ID        string
	Version   string
	Recursive bool
}

// DeletePageUseCase 删除页面记录并组织链接、附件、收藏和历史清理。
// 附件与收藏清理错误只记录告警，用例成功不表示每项资源都已删除。
type DeletePageUseCase struct {
	tree         *tree.TreeService
	revision     *revision.Service
	assets       *assets.AssetService
	favorites    *favorites.FavoritesStore
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
	metrics      *httpmetrics.HTTPMetrics
}

// NewDeletePageUseCase 保存删除流程依赖，不预先检查页面或清理资源。
func NewDeletePageUseCase(
	t *tree.TreeService,
	r *revision.Service,
	a *assets.AssetService,
	f *favorites.FavoritesStore,
	o *pagesave.PageSaveOrchestrator,
	log *slog.Logger,
	metrics *httpmetrics.HTTPMetrics,
) *DeletePageUseCase {
	return &DeletePageUseCase{tree: t, revision: r, assets: a, favorites: f, orchestrator: o, log: log, metrics: metrics}
}

// Execute 通过编排器工作单元绑定页面、历史和收藏的数据库操作。
// 根标识和空标识被拒绝，外部跳过版本标记转为空后交给树服务校验。
// 页面及事件失败立即返回，附件和收藏失败继续执行，历史删除失败向外传播。
func (uc *DeletePageUseCase) Execute(ctx context.Context, in DeletePageInput) (err error) {
	if !uc.tree.InUnitOfWork() {
		err := uc.orchestrator.Execute(ctx, uc.tree, func(bound *tree.TreeService, o *pagesave.PageSaveOrchestrator, exec database.Executor) error {
			copy := *uc
			copy.tree = bound
			copy.orchestrator = o
			copy.revision = uc.revision.WithExecutor(exec, bound)
			if uc.favorites != nil {
				copy.favorites = uc.favorites.WithExecutor(exec)
			}
			return copy.Execute(ctx, in)
		})
		return err
	}

	started := time.Now()
	defer func() {
		uc.metrics.ObservePageSaveWorkflow(string(pagesave.PageOperationDelete), err, started)
	}()

	if in.ID == "root" || in.ID == "" {
		return newPageRootOperationError("delete")
	}

	in.Version = sanitizeClientVersion(in.Version)

	page, err := uc.tree.GetPage(in.ID)
	if err != nil {
		return err
	}

	if in.Recursive {
		var subtreeIDs []string

		if uc.tree.IsLoaded() {
			node, err := uc.tree.FindPageByID(in.ID)
			if err == nil && node != nil {
				subtreeIDs = collectSubtreeIDs(node)
			}
		}
		if len(subtreeIDs) == 0 {
			subtreeIDs = []string{in.ID}
		}

		// 删除前读取受影响页面，失败项不进入事件及附件收藏清理列表。
		// 历史删除仍使用完整子树标识，与此处成功读取的页面列表分开。
		affectedPages := make([]*tree.Page, 0, len(subtreeIDs))
		pages, errs := uc.tree.GetPages(subtreeIDs)
		for i, p := range pages {
			if errs[i] != nil {
				uc.log.Warn("failed to get page before recursive delete", "pageID", subtreeIDs[i], "error", errs[i])
				continue
			}
			affectedPages = append(affectedPages, p)
		}

		oldPath := page.CalculatePath()

		if err := uc.tree.DeleteNode(in.UserID, in.ID, true, in.Version); err != nil {
			return err
		}

		if err := uc.orchestrator.Run(pagesave.PageSaveEvent{
			Operation:     pagesave.PageOperationDelete,
			UserID:        in.UserID,
			Before:        page,
			OldPath:       oldPath,
			AffectedPages: affectedPages,
		}); err != nil {
			return err
		}

		// 单页资源清理失败只记录告警，不阻止后续页面及历史清理。
		for _, p := range affectedPages {
			if err := uc.assets.DeleteAllAssetsForPage(p.PageNode); err != nil {
				uc.log.Warn("failed to delete assets for page", "pageID", p.ID, "error", err)
			}
			if err := uc.favorites.DeleteAllForPage(p.ID); err != nil {
				uc.log.Warn("failed to delete favorites for page", "pageID", p.ID, "error", err)
			}
		}

		return deleteRevisionData(uc.revision, subtreeIDs)
	}

	// 非递归路径只处理目标页，有子节点时由树服务在实际删除前拒绝。
	oldPath := page.CalculatePath()

	if err := uc.tree.DeleteNode(in.UserID, in.ID, false, in.Version); err != nil {
		return err
	}

	if err := uc.orchestrator.Run(pagesave.PageSaveEvent{
		Operation:     pagesave.PageOperationDelete,
		UserID:        in.UserID,
		Before:        page,
		OldPath:       oldPath,
		AffectedPages: []*tree.Page{page},
	}); err != nil {
		return err
	}

	// 文件系统附件操作沿用附件服务实现，不由数据库回滚自动恢复。
	if err := uc.assets.DeleteAllAssetsForPage(page.PageNode); err != nil {
		uc.log.Warn("failed to delete assets for page", "pageID", page.ID, "error", err)
	}
	if err := uc.favorites.DeleteAllForPage(page.ID); err != nil {
		uc.log.Warn("failed to delete favorites for page", "pageID", page.ID, "error", err)
	}

	return deleteRevisionData(uc.revision, []string{in.ID})
}
