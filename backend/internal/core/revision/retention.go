// 本文件执行历史数量裁剪、页面历史清理，以及资源可读性和内容摘要检查。
package revision

import (
	"crypto/sha256"
	"encoding/hex"

	"fmt"
	"io"

	"strings"
)

// pruneAfterSave 在正数保留上限下删除超量的旧历史描述。
// 清理失败只记录警告，不向保存调用方返回错误，也不撤销已经完成的保存。
func (s *Service) pruneAfterSave(pageID string) {
	if s.maxRevisions <= 0 {
		return
	}
	if err := s.store.PruneRevisions(pageID, s.maxRevisions); err != nil {
		s.log.Warn("failed to prune old revisions", "pageID", pageID, "maxRevisions", s.maxRevisions, "error", err)
	}
}

// DeletePageData 裁去页面 ID 边缘空白，非空时清理历史描述及正文记录。
// 存储清理成功后删除本实例资源缓存，资源块和共享资源清单不在此处删除。
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

// CheckRevisionIntegrity 遍历页面历史，收集正文不可读、清单缺失和资源块异常。
// 历史列表读取失败直接返回错误；逐项异常写入 issues 后继续检查，不中断全部扫描。
// 正文仅检查可读取，资源块才重新计算 SHA-256 与大小，清单本身不重新验算哈希。
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
		// 无法加载清单时无法确定资源引用，登记问题后继续下一条历史。
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
			// 流式读完资源计算摘要和实际字节数，文件关闭结果不单独报告。
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
			// 哈希不匹配时上方已继续下一资源，只有哈希一致时才单独报告大小问题。
			if size != ref.SizeBytes {
				issues = append(issues, RevisionIntegrityIssue{PageID: rev.PageID, RevisionID: rev.ID, Code: "asset_blob_size_mismatch", Message: fmt.Sprintf("Revision asset blob for %s failed size verification", ref.Name), Path: blobPath})
			}
		}
	}
	return issues, nil
}
