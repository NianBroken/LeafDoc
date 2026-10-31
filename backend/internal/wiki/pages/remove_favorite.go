// 本文件提供取消收藏用例，只删除用户与页面的关联，不要求目标页面仍然存在。
package pages

import (
	"context"

	"github.com/NianBroken/LeafDoc/internal/core/favorites"
)

// RemoveFavoriteInput 指定用户与页面的收藏关联，不要求页面当前仍能读取。
type RemoveFavoriteInput struct {
	UserID string
	PageID string
}

// RemoveFavoriteUseCase 直接删除指定收藏关联，不预读页面。
// 关联不存在不会形成业务错误，实际存储故障仍向上传递。
type RemoveFavoriteUseCase struct {
	store *favorites.FavoritesStore
}

// NewRemoveFavoriteUseCase 保存收藏存储引用，不执行清理。
func NewRemoveFavoriteUseCase(store *favorites.FavoritesStore) *RemoveFavoriteUseCase {
	return &RemoveFavoriteUseCase{store: store}
}

// Execute 原样传递用户与页面标识，返回底层删除结果，不在此检查上下文取消。
func (uc *RemoveFavoriteUseCase) Execute(_ context.Context, in RemoveFavoriteInput) error {
	return uc.store.Remove(in.UserID, in.PageID)
}
