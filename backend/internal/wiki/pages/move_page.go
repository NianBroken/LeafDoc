package pages

import (
	"context"
	"log/slog"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// MovePageInput 指定移动节点、目标父节点、操作用户和预期版本。
type MovePageInput struct {
	UserID   string
	ID       string
	Version  string
	ParentID string
	// Position 传递给树服务，空值使用 -1，最终顺序沿用树服务的重排规则。
	Position *int
}

// MovePageUseCase 移动页面后为整个子树发送保存事件，派生处理由编排器执行。
type MovePageUseCase struct {
	tree         *tree.TreeService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
	metrics      *httpmetrics.HTTPMetrics
}

// NewMovePageUseCase 保存移动所需依赖，不预读页面或修改层级。
func NewMovePageUseCase(
	t *tree.TreeService,
	o *pagesave.PageSaveOrchestrator,
	log *slog.Logger,
	metrics *httpmetrics.HTTPMetrics,
) *MovePageUseCase {
	return &MovePageUseCase{tree: t, orchestrator: o, log: log, metrics: metrics}
}

// Execute 在编排器工作单元内先取得旧路径和子树标识，再移动并读取受影响页面。
// 移动后单页读取失败仅记录告警，该页不进入事件；事件处理错误向外传播。
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

	// 子树读取失败时退回目标标识，但目标正文必须能在移动前读取成功。
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

	// 路径提前保存为字符串，避免后续节点父指针变化影响旧地址。
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

	// 移动后的页面逐个进入事件，旧地址在独立 OldPath 字段中提供。
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
