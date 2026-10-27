package revision

import (
	"crypto/sha256"
	"encoding/hex"

	"fmt"
	"io"

	"strings"
)

// pruneAfterSave removes the oldest revisions beyond the configured limit.
// Errors are non-fatal and only logged — a prune failure must not fail the save.
func (s *Service) pruneAfterSave(pageID string) {
	if s.maxRevisions <= 0 {
		return
	}
	if err := s.store.PruneRevisions(pageID, s.maxRevisions); err != nil {
		s.log.Warn("failed to prune old revisions", "pageID", pageID, "maxRevisions", s.maxRevisions, "error", err)
	}
}

func (s *Service) DeletePageData(pageID string) error {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return nil
	}

	if err := s.store.DeletePageRevisions(pageID); err != nil {
		return err
	}
	s.assetManifestCache.Delete(pageID)

	return nil
}

func (s *Service) CheckRevisionIntegrity(pageID string) ([]RevisionIntegrityIssue, error) {
	revisions, err := s.store.ListRevisions(pageID)
	if err != nil {
		return nil, err
	}

	issues := make([]RevisionIntegrityIssue, 0)
	for _, rev := range revisions {
		if rev == nil {
			continue
		}
		if strings.TrimSpace(rev.ContentHash) != "" {
			rc, err := s.store.OpenContentBlob(rev.PageID, rev.ContentHash)
			if err != nil {
				issues = append(issues, RevisionIntegrityIssue{PageID: rev.PageID, RevisionID: rev.ID, Code: "missing_content_blob", Message: "Revision content blob is missing or unreadable", Path: s.store.contentBlobPath(rev.PageID, rev.ContentHash)})
			} else {
				_ = rc.Close()
			}
		}
		refs, err := s.store.LoadAssetManifest(rev.AssetManifestHash)
		if err != nil {
			issues = append(issues, RevisionIntegrityIssue{PageID: rev.PageID, RevisionID: rev.ID, Code: "missing_asset_manifest", Message: "Revision asset manifest is missing or unreadable", Path: s.store.assetManifestPath(rev.AssetManifestHash)})
			continue
		}
		for _, ref := range refs {
			blobPath := s.store.AssetBlobPath(ref.SHA256)
			f, err := s.store.OpenAssetBlob(ref.SHA256)
			if err != nil {
				issues = append(issues, RevisionIntegrityIssue{PageID: rev.PageID, RevisionID: rev.ID, Code: "missing_asset_blob", Message: fmt.Sprintf("Revision asset blob for %s is missing or unreadable", ref.Name), Path: blobPath})
				continue
			}
			hasher := sha256.New()
			size, copyErr := io.Copy(hasher, f)
			_ = f.Close()
			if copyErr != nil {
				issues = append(issues, RevisionIntegrityIssue{PageID: rev.PageID, RevisionID: rev.ID, Code: "missing_asset_blob", Message: fmt.Sprintf("Revision asset blob for %s is missing or unreadable", ref.Name), Path: blobPath})
				continue
			}
			if hex.EncodeToString(hasher.Sum(nil)) != ref.SHA256 {
				issues = append(issues, RevisionIntegrityIssue{PageID: rev.PageID, RevisionID: rev.ID, Code: "asset_blob_hash_mismatch", Message: fmt.Sprintf("Revision asset blob for %s failed hash verification", ref.Name), Path: blobPath})
				continue
			}
			if size != ref.SizeBytes {
				issues = append(issues, RevisionIntegrityIssue{PageID: rev.PageID, RevisionID: rev.ID, Code: "asset_blob_size_mismatch", Message: fmt.Sprintf("Revision asset blob for %s failed size verification", ref.Name), Path: blobPath})
			}
		}
	}
	return issues, nil
}
