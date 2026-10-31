package pages

import (
	"context"
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// ConvertPageInput 提供页面标识、预期版本、目标类型和操作用户。
type ConvertPageInput struct {
	UserID     string
	ID         string
	Version    string
	TargetKind tree.NodeKind
}

// ConvertPageUseCase 在页面与分区之间转换类型，并记录结构历史状态。
type ConvertPageUseCase struct {
	tree     *tree.TreeService
	revision *revision.Service
	log      *slog.Logger
}

// NewConvertPageUseCase 保存树、历史和日志依赖，不立即转换节点。
func NewConvertPageUseCase(t *tree.TreeService, r *revision.Service, log *slog.Logger) *ConvertPageUseCase {
	return &ConvertPageUseCase{tree: t, revision: r, log: log}
}

// Execute 拒绝根节点和空标识，清理外部版本标记后调用树服务转换。
// 外层工作单元绑定树与历史执行器，任一步错误向工作单元传播。
// 同类型请求在树服务中不产生类型写入，此处仍继续调用已配置的结构历史服务。
func (uc *ConvertPageUseCase) Execute(ctx context.Context, in ConvertPageInput) error {
	if !uc.tree.InUnitOfWork() {
		return uc.tree.WithUnitOfWork(ctx, func(bound *tree.TreeService, exec database.Executor) error {
			copy := *uc
			copy.tree = bound
			copy.revision = uc.revision.WithExecutor(exec, bound)
			return copy.Execute(ctx, in)
		})
	}

	if in.ID == "root" || in.ID == "" {
		return newPageRootOperationError("convert")
	}
	in.Version = sanitizeClientVersion(in.Version)
	if err := uc.tree.ConvertNode(in.UserID, in.ID, in.TargetKind, in.Version); err != nil {
		return err
	}
	if uc.revision != nil {
		if _, _, err := uc.revision.RecordStructureChange(in.ID, in.UserID, ""); err != nil {
			return err
		}
	}
	return nil
}
