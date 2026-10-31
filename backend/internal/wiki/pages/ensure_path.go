package pages

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// EnsurePathInput 指定目标路由和最终页面标题，缺失中间层级由树服务建立。
type EnsurePathInput struct {
	UserID      string
	TargetPath  string
	TargetTitle string
	Kind        *tree.NodeKind
}

// EnsurePathOutput 返回路径最终对应的页面，包括已存在路径的原页面。
type EnsurePathOutput struct {
	Page *tree.Page
}

// EnsurePathUseCase 验证路径并确保层级存在，为实际创建的节点发送保存事件。
type EnsurePathUseCase struct {
	tree         *tree.TreeService
	slug         *tree.SlugService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
}

// NewEnsurePathUseCase 保存路径验证、页面树与保存编排依赖，不执行结构变化。
func NewEnsurePathUseCase(
	t *tree.TreeService,
	s *tree.SlugService,
	o *pagesave.PageSaveOrchestrator,
	log *slog.Logger,
) *EnsurePathUseCase {
	return &EnsurePathUseCase{tree: t, slug: s, orchestrator: o, log: log}
}

// Execute 在编排器工作单元中处理路径建立，路径和标题去掉两端空白后不能为空。
// 完整路径已存在时返回原页面，不覆盖标题或类型，也不发送创建事件。
func (uc *EnsurePathUseCase) Execute(ctx context.Context, in EnsurePathInput) (*EnsurePathOutput, error) {
	if !uc.tree.InUnitOfWork() {
		var result *EnsurePathOutput
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

	cleanPath := strings.Trim(strings.TrimSpace(in.TargetPath), "/")
	if cleanPath == "" {
		ve.Add("path", "Path must not be empty")
	}

	cleanTitle := strings.TrimSpace(in.TargetTitle)
	if cleanTitle == "" {
		ve.Add("title", "Title must not be empty")
	}

	if ve.HasErrors() {
		return nil, ve
	}

	lookup, err := uc.tree.LookupPagePath(cleanPath)
	if err != nil {
		return nil, err
	}

	if lookup.Exists {
		page, err := uc.tree.GetPage(*lookup.Segments[len(lookup.Segments)-1].ID)
		if err != nil {
			return nil, err
		}
		return &EnsurePathOutput{Page: page}, nil
	}

	// 只验证缺失段的名称，已有路径段保持查询结果。
	for _, segment := range lookup.Segments {
		if !segment.Exists {
			if err := uc.slug.IsValidSlug(segment.Slug); err != nil {
				ve.Add("path", fmt.Sprintf("Invalid slug '%s': %s", segment.Slug, err.Error()))
			}
		}
	}
	if ve.HasErrors() {
		return nil, ve
	}

	result, err := uc.tree.EnsurePagePath(in.UserID, cleanPath, cleanTitle, in.Kind)
	if err != nil {
		return nil, err
	}

	// 最终页面也会出现在创建列表中，按标识去重后再批量读取。
	ids := make([]string, 0, len(result.Created)+1)
	seen := make(map[string]struct{}, len(result.Created)+1)
	appendUnique := func(id string) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	appendUnique(result.Page.ID)
	for _, n := range result.Created {
		appendUnique(n.ID)
	}

	// 最终页面读取失败中止处理，其他新建节点读取失败仅记录告警。
	pages, errs := uc.tree.GetPages(ids)
	pageByID := make(map[string]*tree.Page, len(ids))
	for i, p := range pages {
		if errs[i] != nil {
			if ids[i] == result.Page.ID {
				return nil, errs[i]
			}
			uc.log.Warn("failed to get page for post-create processing", "pageID", ids[i], "error", errs[i])
			continue
		}
		pageByID[ids[i]] = p
	}

	page := pageByID[result.Page.ID]
	if page == nil {
		return nil, tree.ErrPageNotFound
	}

	// 将各创建节点加入待读列表，最终页面标识已经登记，因此不会重复追加。
	// 按创建顺序发送事件，没有成功读取的节点不进入派生处理。
	for _, n := range result.Created {
		p := pageByID[n.ID]
		if p == nil {
			uc.log.Warn("failed to get page for post-create processing", "pageID", n.ID, "error", tree.ErrPageNotFound)
			continue
		}
		if err := uc.orchestrator.Run(pagesave.PageSaveEvent{
			Operation: pagesave.PageOperationCreate,
			UserID:    in.UserID,
			After:     p,
			Summary:   "page created via ensure path",
		}); err != nil {
			return nil, err
		}
	}

	return &EnsurePathOutput{Page: page}, nil
}
