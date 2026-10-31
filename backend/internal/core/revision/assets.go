// 本文件扫描页面资源、计算内容引用并保存资源块，为历史描述提供可重用清单。
package revision

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"fmt"
	"io"

	"mime"
	"os"
	"path/filepath"

	"sort"
	"strings"
)

// assetManifestEntry 仅缓存页面当前使用的清单哈希，不缓存清单内容或文件状态。
type assetManifestEntry struct {
	hash string
}

// persistLiveAssets 按引用列表逐个复制现场资源，并核对复制所得摘要与大小。
// 扫描后文件发生变化时返回不一致错误，先前已保存的资源块不会回滚。
func (s *Service) persistLiveAssets(pageID string, refs []AssetRef) error {
	if len(refs) == 0 {
		return nil
	}

	for _, ref := range refs {
		srcPath := filepath.Join(s.liveAssetDir(pageID), ref.Name)
		hash, size, err := s.store.SaveAssetBlobFromPath(srcPath)
		if err != nil {
			return err
		}
		if hash != ref.SHA256 {
			return fmt.Errorf("asset hash mismatch for %s: computed=%s saved=%s", ref.Name, ref.SHA256, hash)
		}
		if size != ref.SizeBytes {
			return fmt.Errorf("asset size mismatch for %s: computed=%d saved=%d", ref.Name, ref.SizeBytes, size)
		}
	}
	return nil
}

// scanLiveAssets 读取页面资源目录的直接子项，跳过子目录，不递归扫描。
// 目录不存在返回空列表，其他读取错误向上传递；每个非目录项须能打开并完整读取。
// 返回引用按名称、同名时按哈希排序，扩展名用于推断 MIME 类型。
func (s *Service) scanLiveAssets(pageID string) ([]AssetRef, error) {
	dir := s.liveAssetDir(pageID)
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("asset path is not a directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []AssetRef{}, nil
		}
		return nil, fmt.Errorf("read live asset dir %s: %w", dir, err)
	}

	refs := make([]AssetRef, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		absPath := filepath.Join(dir, name)

		ref, err := buildAssetRef(absPath, name)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}

	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Name == refs[j].Name {
			return refs[i].SHA256 < refs[j].SHA256
		}
		return refs[i].Name < refs[j].Name
	})

	return refs, nil
}

// liveAssetDir 将页面 ID 拼接到数据目录的 assets 子目录，不创建或校验目录。
func (s *Service) liveAssetDir(pageID string) string {
	return filepath.Join(s.storageDir, "assets", pageID)
}

// buildAssetRef 流式读取资源计算 SHA-256 和字节数，名称由调用方提供。
// MIME 类型只按小写扩展名判断，未命中时使用二进制通用类型，不探测文件内容。
// 文件延迟关闭，关闭结果不改变引用构造结果。
func buildAssetRef(absPath, name string) (AssetRef, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return AssetRef{}, fmt.Errorf("open asset %s: %w", absPath, err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return AssetRef{}, fmt.Errorf("hash asset %s: %w", absPath, err)
	}

	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return AssetRef{
		Name:      name,
		SHA256:    hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: size,
		MIMEType:  mimeType,
	}, nil
}

// computeAssetManifestHash 复制排序列表并编码成 assetManifest，与存储清单使用相同结构。
// 摘要覆盖完整 JSON 字节，不修改调用方切片。
func computeAssetManifestHash(items []AssetRef) (string, error) {
	canonical := cloneAndSortAssetRefs(items)

	raw, err := json.Marshal(assetManifest{Items: canonical})
	if err != nil {
		return "", fmt.Errorf("marshal asset manifest for hash: %w", err)
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
