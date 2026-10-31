// 本文件把历史描述、正文与资源清单组合为预览，并计算两份快照的可见差异。
package revision

import (
	"fmt"

	"os"

	"sort"
	"strings"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// ListRevisions 返回指定页面全部历史，排序规则交给存储层。
func (s *Service) ListRevisions(pageID string) ([]*Revision, error) {
	return s.store.ListRevisions(pageID)
}

// ListRevisionsPage 按历史 ID 游标读取下一段结果，不在服务层额外过滤或重排。
func (s *Service) ListRevisionsPage(pageID, cursor string, limit int) ([]*Revision, string, error) {
	return s.store.ListRevisionsPage(pageID, cursor, limit)
}

// GetLatestRevision 获取排序最前的历史，没有记录时返回 nil 且无错误。
func (s *Service) GetLatestRevision(pageID string) (*Revision, error) {
	return s.store.GetLatestRevision(pageID)
}

// GetRevisionSnapshot 依次读取描述、正文和资源清单，任一步失败不返回部分快照。
// 正文或资源清单不可读时包装对应本地化错误，成功资源列表复制并按名称排序。
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

// CompareRevisionSnapshots 读取同页的两份历史，按正文字节内容和资源集合进行比较。
// ContentChanged 不包含标题、路径或额外前置字段变化，资源差异只比较名称、哈希和大小。
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

// GetRevisionAsset 从指定历史的清单中按名称查找资源并确认资源路径可被 Stat。
// 名称只去掉一个开头斜杠再裁边缘空白，不能用任意请求路径直接定位磁盘资源。
// 返回前不打开文件、不验证常规文件类型或重新计算摘要，实际读取由调用方负责。
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

// compareRevisionAssets 以名称建立两组索引，同名重复项由各自列表后项覆盖。
// 缺少目标项记为 removed，目标独有项记为 added，哈希或大小变化记为 modified。
// MIME 类型不参与比较，结果按资源名称升序，空差异返回空切片。
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
