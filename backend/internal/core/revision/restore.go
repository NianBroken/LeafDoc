// 本文件把历史正文、额外前置字段和资源写回现有页面，并记录恢复操作。
// 数据库页面更新与资源目录替换分步执行，失败后通过显式补偿尝试恢复现场。
package revision

import (
	"errors"
	"fmt"

	"os"
	"path/filepath"

	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/markdown"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// RestoreRevision 要求现有页面和历史记录均存在，裁去两个 ID 边缘空白后执行。
// 恢复历史标题和内容，保留当前 slug，不使用历史父节点、路径或页面类型重建结构。
// 版本检查使用 VersionUnchecked，资源或恢复记录失败时尝试回写之前的页面和资源。
func (s *Service) RestoreRevision(pageID, revisionID, authorID string) error {
	pageID = strings.TrimSpace(pageID)
	revisionID = strings.TrimSpace(revisionID)
	if pageID == "" {
		return sharederrors.NewLocalizedError(
			"revision_restore_invalid_page_id",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			nil,
			pageID,
		)
	}
	if revisionID == "" {
		return sharederrors.NewLocalizedError(
			"revision_restore_invalid_revision",
			"Restore revision is invalid",
			"restore revision %s for page %s is invalid",
			nil,
			revisionID,
			pageID,
		)
	}

	if _, err := s.pages.GetPage(pageID); err != nil {
		if errors.Is(err, tree.ErrPageNotFound) {
			return sharederrors.NewLocalizedError(
				"revision_restore_page_not_found",
				"Page not found",
				"page %s not found",
				err,
				pageID,
			)
		}
		return sharederrors.NewLocalizedError(
			"revision_restore_failed",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			err,
			pageID,
		)
	}

	// 页面存在性先在锁外检查，后续历史读取、现场捕获与写回在同实例页面锁内进行。
	mu := s.pageWriteLock(pageID)
	mu.Lock()
	defer mu.Unlock()

	rev, err := s.store.GetRevision(pageID, revisionID)
	if err != nil {
		if os.IsNotExist(err) {
			return sharederrors.NewLocalizedError(
				"revision_restore_revision_not_found",
				"Restore revision not found",
				"restore revision %s for page %s not found",
				err,
				revisionID,
				pageID,
			)
		}
		return sharederrors.NewLocalizedError(
			"revision_restore_failed",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			err,
			pageID,
		)
	}

	content, err := s.store.ReadContentBlob(pageID, rev.ContentHash)
	if err != nil {
		return sharederrors.NewLocalizedError(
			"revision_restore_content_missing",
			"Restore content is unavailable",
			"restore content for page %s is unavailable",
			err,
			pageID,
		)
	}

	assets, err := s.store.LoadAssetManifest(rev.AssetManifestHash)
	if err != nil {
		return sharederrors.NewLocalizedError(
			"revision_restore_assets_missing",
			"Restore assets are unavailable",
			"restore assets for page %s are unavailable",
			err,
			pageID,
		)
	}

	// 捕获操作前页面状态供失败补偿使用，此步骤不主动保存一条操作前历史。
	beforeState, err := s.capturePageState(pageID, true)
	if err != nil {
		return sharederrors.NewLocalizedError(
			"revision_restore_failed",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			err,
			pageID,
		)
	}

	restoredContent, restorePreserveFrontmatter, err := buildRestoredRawContent(rev.ExtraFrontmatter, string(content))
	if err != nil {
		return sharederrors.NewLocalizedError(
			"revision_restore_failed",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			err,
			pageID,
		)
	}
	if err := s.pages.UpdateNode(tree.UpdateNodeInput{
		UserID:              authorID,
		ID:                  pageID,
		Title:               rev.Title,
		Slug:                beforeState.Slug,
		Content:             &restoredContent,
		ExpectedVersion:     tree.VersionUnchecked,
		PreserveFrontmatter: restorePreserveFrontmatter,
	}); err != nil {
		return sharederrors.NewLocalizedError(
			"revision_restore_failed",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			err,
			pageID,
		)
	}

	// 页面已经写入后才替换资源。资源失败时尝试分别还原正文和资源，
	// 补偿失败只记警告，最终仍返回最初资源恢复错误，不承诺补偿全部成功。
	if err := s.restoreAssets(pageID, assets); err != nil {
		restoreRollbackContent, rollbackPreserveFrontmatter, buildErr := buildRestoredRawContent(beforeState.ExtraFrontmatter, beforeState.Content)
		if buildErr != nil {
			s.log.Warn("failed to rebuild rollback content", "pageID", pageID, "error", buildErr)
			restoreRollbackContent = beforeState.Content
			rollbackPreserveFrontmatter = false
		}
		if rollbackErr := s.pages.UpdateNode(tree.UpdateNodeInput{
			UserID:              authorID,
			ID:                  pageID,
			Title:               beforeState.Title,
			Slug:                beforeState.Slug,
			Content:             &restoreRollbackContent,
			ExpectedVersion:     tree.VersionUnchecked,
			PreserveFrontmatter: rollbackPreserveFrontmatter,
		}); rollbackErr != nil {
			s.log.Warn("failed to rollback restored content", "pageID", pageID, "error", rollbackErr)
		}
		if rollbackErr := s.restoreAssets(pageID, beforeState.Assets); rollbackErr != nil {
			s.log.Warn("failed to rollback restored assets", "pageID", pageID, "error", rollbackErr)
		}
		return sharederrors.NewLocalizedError(
			"revision_restore_failed",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			err,
			pageID,
		)
	}

	// 历史保存失败同样触发现场补偿，已产生的正文块或资源块不在此分支统一删除。
	if err := s.recordRestoreRevision(pageID, authorID); err != nil {
		restoreRollbackContent, rollbackPreserveFrontmatter, buildErr := buildRestoredRawContent(beforeState.ExtraFrontmatter, beforeState.Content)
		if buildErr != nil {
			s.log.Warn("failed to rebuild rollback content", "pageID", pageID, "error", buildErr)
			restoreRollbackContent = beforeState.Content
			rollbackPreserveFrontmatter = false
		}
		if rollbackErr := s.pages.UpdateNode(tree.UpdateNodeInput{
			UserID:              authorID,
			ID:                  pageID,
			Title:               beforeState.Title,
			Slug:                beforeState.Slug,
			Content:             &restoreRollbackContent,
			ExpectedVersion:     tree.VersionUnchecked,
			PreserveFrontmatter: rollbackPreserveFrontmatter,
		}); rollbackErr != nil {
			s.log.Warn("failed to rollback restored content", "pageID", pageID, "error", rollbackErr)
		}
		if rollbackErr := s.restoreAssets(pageID, beforeState.Assets); rollbackErr != nil {
			s.log.Warn("failed to rollback restored assets", "pageID", pageID, "error", rollbackErr)
		}
		return sharederrors.NewLocalizedError(
			"revision_restore_failed",
			errFailedToRestorePage,
			errFailedToRestorePageFmt,
			err,
			pageID,
		)
	}

	return nil
}

// buildRestoredRawContent 有额外字段时重新构建 Markdown，并通知页面更新保留前置字段。
// 没有额外字段则直接返回正文及 false，不插入空前置字段区段。
func buildRestoredRawContent(extra map[string]interface{}, body string) (string, bool, error) {
	if len(extra) == 0 {
		return body, false, nil
	}

	raw, err := markdown.BuildMarkdownWithExtraFrontmatter(extra, body)
	if err != nil {
		return "", false, err
	}

	return raw, true, nil
}

// restoreAssets 先移除现有页面资源目录，再根据清单逐项恢复，空清单保持目录不存在。
// 名称逐项裁边缘空白并按当前系统路径规则校验，重复名称拒绝写入。
// 校验和复制发生在原目录删除之后，失败时目录中已写入项保留，外层负责补偿。
func (s *Service) restoreAssets(pageID string, refs []AssetRef) error {
	dir := s.liveAssetDir(pageID)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reset live asset dir: %w", err)
	}
	if len(refs) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure live asset dir: %w", err)
	}

	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		name := strings.TrimSpace(ref.Name)
		if name == "" || filepath.Base(name) != name || strings.Contains(name, string(os.PathSeparator)) {
			return fmt.Errorf("invalid asset name: %s", ref.Name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate asset name in manifest: %s", name)
		}
		seen[name] = struct{}{}

		if err := s.store.CopyAssetBlobToPath(ref.SHA256, ref.SizeBytes, filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("restore asset %s: %w", name, err)
		}
	}

	return nil
}

// recordRestoreRevision 捕获恢复后的完整现场，依次保存正文、资源块、清单和恢复描述。
// 本方法由已持页面锁的恢复入口调用，不再次取锁，也不执行保存后数量裁剪。
func (s *Service) recordRestoreRevision(pageID, authorID string) error {
	state, err := s.capturePageState(pageID, true)
	if err != nil {
		return err
	}

	contentHash, err := s.store.SaveContentBlob(pageID, []byte(state.Content))
	if err != nil {
		return err
	}
	if contentHash != state.ContentHash {
		return fmt.Errorf(errContentHashMismatch, state.ContentHash, contentHash)
	}

	if err := s.persistLiveAssets(pageID, state.Assets); err != nil {
		return err
	}

	savedManifestHash, err := s.store.SaveAssetManifest(state.Assets)
	if err != nil {
		return err
	}
	if savedManifestHash != state.AssetManifestHash {
		return fmt.Errorf(errAssetManifestHashMismatch, state.AssetManifestHash, savedManifestHash)
	}

	rev, err := s.newRevision(RevisionTypeRestore, state, authorID, "page restored", savedManifestHash)
	if err != nil {
		return err
	}
	return s.store.SaveRevision(rev)
}
