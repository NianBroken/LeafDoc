package pages

import (
	"context"
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// ListFavoritesInput 指定需要查询个人收藏的用户标识。
type ListFavoritesInput struct {
	UserID string
}

// ListFavoritesOutput 提供能成功读取的收藏页面，没有结果时使用非空的空切片。
type ListFavoritesOutput struct {
	Pages []*tree.Page
}

// ListFavoritesUseCase 按用户收藏标识批量读取页面，失败页面记录告警后跳过。
// 此流程不删除失效收藏记录，也不将单页失败作为整个列表查询失败。
type ListFavoritesUseCase struct {
	treeService *tree.TreeService
	store       *favorites.FavoritesStore
	log         *slog.Logger
}

// NewListFavoritesUseCase 保存查询和日志依赖，调用方须提供可用日志器。
func NewListFavoritesUseCase(treeService *tree.TreeService, store *favorites.FavoritesStore, log *slog.Logger) *ListFavoritesUseCase {
	return &ListFavoritesUseCase{treeService: treeService, store: store, log: log}
}

// Execute 先取得收藏标识列表，存储错误立即返回。
// 页面读取结果按原标识顺序过滤后交付，不另行排序，也不使用上下文取消查询。
func (uc *ListFavoritesUseCase) Execute(_ context.Context, in ListFavoritesInput) (*ListFavoritesOutput, error) {
	ids, err := uc.store.ListPageIDsForUser(in.UserID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return &ListFavoritesOutput{Pages: []*tree.Page{}}, nil
	}

	pages, errs := uc.treeService.GetPages(ids)
	result := make([]*tree.Page, 0, len(pages))
	for i, p := range pages {
		if errs[i] != nil {
			uc.log.Warn("skipping stale favorite", "userID", in.UserID, "pageID", ids[i], "error", errs[i])
			continue
		}
		result = append(result, p)
	}
	return &ListFavoritesOutput{Pages: result}, nil
}
