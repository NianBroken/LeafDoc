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
