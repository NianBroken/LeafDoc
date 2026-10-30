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

// assetManifestEntry is the in-memory cache entry for a page's latest asset manifest hash.
type assetManifestEntry struct {
	hash string
}

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

// Assumption for V1:
// live assets are stored under <storageDir>/assets/<pageID>/...
// If your AssetService uses a different on-disk layout, only change this method.
func (s *Service) liveAssetDir(pageID string) string {
	return filepath.Join(s.storageDir, "assets", pageID)
}

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

func computeAssetManifestHash(items []AssetRef) (string, error) {
	canonical := cloneAndSortAssetRefs(items)

	raw, err := json.Marshal(assetManifest{Items: canonical})
	if err != nil {
		return "", fmt.Errorf("marshal asset manifest for hash: %w", err)
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
