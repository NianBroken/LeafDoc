package importer

import (
	"fmt"
	"mime/multipart"

	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// resolveAndUploadAsset imports local non-Markdown files into the target page's asset folder
// and caches the public path so repeated references on the same page reuse the upload result.
func (t *contentTransformer) resolveAndUploadAsset(
	userID string,
	sourcePath string,
	page *tree.Page,
	href string,
	wiki ImporterWiki,
) (string, error) {
	assetAbs, ok := resolveAssetPath(t.sourceBasePath, sourcePath, href)
	if !ok {
		return "", nil
	}

	cacheKey := page.ID + "::" + assetAbs
	if uploaded, ok := t.assetUploads[cacheKey]; ok {
		return uploaded, nil
	}

	file, err := os.Open(assetAbs)
	if err != nil {
		return "", fmt.Errorf("open asset %q: %w", assetAbs, err)
	}
	defer func() {
		_ = file.Close()
	}()

	publicPath, err := wiki.UploadAsset(userID, page.ID, multipart.File(file), filepath.Base(assetAbs), t.assetMaxBytes)
	if err != nil {
		return "", fmt.Errorf("upload asset %q: %w", assetAbs, err)
	}

	t.assetUploads[cacheKey] = publicPath
	return publicPath, nil
}

func isNonMarkdownAssetTarget(href string) bool {
	decoded := strings.TrimSpace(decodeImportTarget(href))
	ext := strings.ToLower(path.Ext(decoded))
	return ext != "" && ext != ".md"
}

// resolveAssetPath keeps asset resolution inside the extracted import workspace.
// This prevents uploaded archives from referencing files outside the package on disk.
func resolveAssetPath(sourceBasePath string, sourcePath string, href string) (string, bool) {
	raw := filepath.ToSlash(strings.TrimSpace(href))
	if raw == "" {
		return "", false
	}
	if strings.HasSuffix(strings.ToLower(raw), ".md") {
		return "", false
	}

	var rel string
	if strings.HasPrefix(raw, "/") {
		rel = path.Clean(strings.TrimPrefix(raw, "/"))
	} else {
		currentDir := path.Dir(filepath.ToSlash(sourcePath))
		if currentDir == "." {
			currentDir = ""
		}
		rel = path.Clean(path.Join(currentDir, raw))
	}

	if rel == "." || strings.HasPrefix(rel, "../") {
		return "", false
	}

	abs := filepath.Join(sourceBasePath, filepath.FromSlash(rel))
	baseAbs, err := filepath.Abs(sourceBasePath)
	if err != nil {
		return "", false
	}
	absResolved, err := filepath.Abs(abs)
	if err != nil {
		return "", false
	}

	relCheck, err := filepath.Rel(baseAbs, absResolved)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
		return "", false
	}

	info, err := os.Stat(absResolved)
	if err != nil || info.IsDir() {
		return "", false
	}

	return absResolved, true
}

func isImageAssetTarget(target string) bool {
	base, _ := splitURLSuffix(target)
	switch strings.ToLower(path.Ext(base)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".avif":
		return true
	default:
		return false
	}
}
