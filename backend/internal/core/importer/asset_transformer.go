// 本文件把导入正文中的附件目标定位到解压工作区，并通过 Wiki 接口上传为页面附件。
// 文件路径边界检查和图像后缀分类各自独立，上传缓存按页面与绝对源路径区分。
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

// resolveAndUploadAsset 将能定位到的非 Markdown 文件上传为目标页面的附件。
// 找不到有效文件时返回空路径而不报错，由链接转换调用方保留或另行处理原目标。
// 打开或上传失败会携带本地文件路径返回错误，成功结果按页面和绝对文件路径缓存。
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

	// 同一页面重复引用同一文件复用上传路径，不同页面分别拥有各自的附件。
	cacheKey := page.ID + "::" + assetAbs
	if uploaded, ok := t.assetUploads[cacheKey]; ok {
		return uploaded, nil
	}

	file, err := os.Open(assetAbs)
	if err != nil {
		return "", fmt.Errorf("open asset %q: %w", assetAbs, err)
	}
	// 上传调用在文件关闭之前完成，关闭错误不覆盖上传结果。
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

// isNonMarkdownAssetTarget 解码并去除两端空白后检查扩展名。
// 只有存在扩展名且不是 .md 的目标被归为附件，这里不检查文件是否存在。
func isNonMarkdownAssetTarget(href string) bool {
	decoded := strings.TrimSpace(decodeImportTarget(href))
	ext := strings.ToLower(path.Ext(decoded))
	return ext != "" && ext != ".md"
}

// resolveAssetPath 按工作区根目录或当前源文件目录解释附件路径。
// 清理后的路径必须在工作区的路径层级内，并且能通过 Stat 找到非目录对象。
// 层级检查基于绝对路径和相对路径计算，不解析符号链接的实际指向。
func resolveAssetPath(sourceBasePath string, sourcePath string, href string) (string, bool) {
	raw := filepath.ToSlash(strings.TrimSpace(href))
	if raw == "" {
		return "", false
	}
	if strings.HasSuffix(strings.ToLower(raw), ".md") {
		return "", false
	}

	var rel string
	// 斜杠开头表示导入工作区根路径，其余目标相对当前 Markdown 文件定位。
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

	// 再以本机路径规则检查父目录跳转，覆盖清理后恰好等于上级目录的情况。
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

// isImageAssetTarget 去除查询串和片段后，按扩展名识别图片链接。
// 此处只做后缀分类，不读取文件头，也不验证图片内容。
func isImageAssetTarget(target string) bool {
	base, _ := splitURLSuffix(target)
	switch strings.ToLower(path.Ext(base)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".avif":
		return true
	default:
		return false
	}
}
