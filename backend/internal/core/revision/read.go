package revision

import (
	"fmt"

	"os"

	"sort"
	"strings"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

func (s *Service) ListRevisions(pageID string) ([]*Revision, error) {
	return s.store.ListRevisions(pageID)
}

func (s *Service) ListRevisionsPage(pageID, cursor string, limit int) ([]*Revision, string, error) {
	return s.store.ListRevisionsPage(pageID, cursor, limit)
}

func (s *Service) GetLatestRevision(pageID string) (*Revision, error) {
	return s.store.GetLatestRevision(pageID)
}

func (s *Service) GetRevisionSnapshot(pageID, revisionID string) (*RevisionSnapshot, error) {
	rev, err := s.store.GetRevision(pageID, revisionID)
	if err != nil {
		return nil, err
	}

	content, err := s.store.ReadContentBlob(rev.PageID, rev.ContentHash)
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			"revision_preview_content_unavailable",
			"Revision content is unavailable",
			"revision content for page %s revision %s is unavailable",
			err,
			pageID,
			revisionID,
		)
	}

	assets, err := s.store.LoadAssetManifest(rev.AssetManifestHash)
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			"revision_preview_assets_unavailable",
			"Revision assets are unavailable",
			"revision assets for page %s revision %s are unavailable",
			err,
			pageID,
			revisionID,
		)
	}

	return &RevisionSnapshot{
		Revision: rev,
		Content:  string(content),
		Assets:   cloneAndSortAssetRefs(assets),
	}, nil
}

func (s *Service) CompareRevisionSnapshots(pageID, baseRevisionID, targetRevisionID string) (*RevisionComparison, error) {
	base, err := s.GetRevisionSnapshot(pageID, baseRevisionID)
	if err != nil {
		return nil, err
	}
	target, err := s.GetRevisionSnapshot(pageID, targetRevisionID)
	if err != nil {
		return nil, err
	}
	return &RevisionComparison{
		Base:           base,
		Target:         target,
		ContentChanged: base.Content != target.Content,
		AssetChanges:   compareRevisionAssets(base.Assets, target.Assets),
	}, nil
}

func (s *Service) GetRevisionAsset(pageID, revisionID, assetName string) (*RevisionAssetContent, error) {
	assetName = strings.TrimSpace(strings.TrimPrefix(assetName, "/"))
	if assetName == "" {
		return nil, sharederrors.NewLocalizedError(
			"revision_preview_asset_invalid_name",
			"Revision asset name is invalid",
			"revision asset name for page %s revision %s is invalid",
			fmt.Errorf("asset name is required"),
			pageID,
			revisionID,
		)
	}

	rev, err := s.store.GetRevision(pageID, revisionID)
	if err != nil {
		return nil, err
	}

	assets, err := s.store.LoadAssetManifest(rev.AssetManifestHash)
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			"revision_preview_assets_unavailable",
			"Revision assets are unavailable",
			"revision assets for page %s revision %s are unavailable",
			err,
			pageID,
			revisionID,
		)
	}

	for _, asset := range assets {
		if asset.Name != assetName {
			continue
		}

		blobPath := s.store.AssetBlobPath(asset.SHA256)
		if _, err := os.Stat(blobPath); err != nil {
			return nil, sharederrors.NewLocalizedError(
				"revision_preview_asset_blob_unavailable",
				"Revision asset is unavailable",
				"revision asset %s for page %s revision %s is unavailable",
				err,
				assetName,
				pageID,
				revisionID,
			)
		}

		return &RevisionAssetContent{
			Asset: asset,
			Path:  blobPath,
		}, nil
	}

	return nil, sharederrors.NewLocalizedError(
		"revision_preview_asset_not_found",
		"Revision asset not found",
		"revision asset %s for page %s revision %s not found",
		fmt.Errorf("asset %q not found in revision manifest", assetName),
		assetName,
		pageID,
		revisionID,
	)
}

func compareRevisionAssets(baseAssets, targetAssets []AssetRef) []RevisionAssetDelta {
	baseByName := make(map[string]AssetRef, len(baseAssets))
	for _, asset := range baseAssets {
		baseByName[asset.Name] = asset
	}
	targetByName := make(map[string]AssetRef, len(targetAssets))
	for _, asset := range targetAssets {
		targetByName[asset.Name] = asset
	}
	changes := make([]RevisionAssetDelta, 0)
	for name, baseAsset := range baseByName {
		targetAsset, ok := targetByName[name]
		if !ok {
			changes = append(changes, RevisionAssetDelta{Name: name, Status: "removed"})
			continue
		}
		if baseAsset.SHA256 != targetAsset.SHA256 || baseAsset.SizeBytes != targetAsset.SizeBytes {
			changes = append(changes, RevisionAssetDelta{Name: name, Status: "modified"})
		}
	}
	for name := range targetByName {
		if _, ok := baseByName[name]; !ok {
			changes = append(changes, RevisionAssetDelta{Name: name, Status: "added"})
		}
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].Name < changes[j].Name })
	return changes
}
