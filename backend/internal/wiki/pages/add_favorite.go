package pages

import (
	"context"

	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// AddFavoriteInput 指定收藏所属用户和页面标识，用户身份由上层认证流程提供。
type AddFavoriteInput struct {
	UserID string
	PageID string
}

// AddFavoriteUseCase 先读取目标页面，再为指定用户保存个人收藏。
// 此对象不自行认证用户或检查编辑角色，路由层负责访问限制。
type AddFavoriteUseCase struct {
	treeService *tree.TreeService
	store       *favorites.FavoritesStore
}

// NewAddFavoriteUseCase 保存页面查询与收藏存储依赖，不立即写入收藏。
func NewAddFavoriteUseCase(treeService *tree.TreeService, store *favorites.FavoritesStore) *AddFavoriteUseCase {
	return &AddFavoriteUseCase{treeService: treeService, store: store}
}

// Execute 只有页面读取成功才调用 Add，错误原样返回，上下文不进入底层同步接口。
func (uc *AddFavoriteUseCase) Execute(_ context.Context, in AddFavoriteInput) error {
	if _, err := uc.treeService.GetPage(in.PageID); err != nil {
		return err
	}
	return uc.store.Add(in.UserID, in.PageID)
}
