// 本文件把页面版本和置顶标志交给树服务，返回保存后的页面，不编排其他页面副作用。
package pages

import (
	"context"
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// PinPageInput 指定页面标识、预期版本和目标置顶状态。
type PinPageInput struct {
	ID      string
	Version string
	Pinned  bool
}

// PinPageOutput 返回置顶保存后的页面与正文。
type PinPageOutput struct {
	Page *tree.Page
}

// PinPageUseCase 直接调用树服务保存置顶状态，不使用通用页面保存编排器。
type PinPageUseCase struct {
	treeService *tree.TreeService
	log         *slog.Logger
}

// NewPinPageUseCase 保存树服务与日志依赖，构造时不读取页面。
func NewPinPageUseCase(treeService *tree.TreeService, log *slog.Logger) *PinPageUseCase {
	return &PinPageUseCase{treeService: treeService, log: log}
}

// Execute 原样传递版本和置顶值，存储错误直接返回。
// 此入口不清理内部版本标记，也不使用上下文中断同步调用。
func (uc *PinPageUseCase) Execute(_ context.Context, in PinPageInput) (*PinPageOutput, error) {
	page, err := uc.treeService.SetPinned(in.ID, in.Version, in.Pinned)
	if err != nil {
		return nil, err
	}
	return &PinPageOutput{Page: page}, nil
}
