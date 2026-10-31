// 本文件管理页面附件的磁盘文件。附件按页面 ID 存在 assets 子目录下，
// 公开地址同样使用页面 ID，页面标题和路径段变化不会改变这里的附件目录。
package assets

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"

	"github.com/NianBroken/LeafDoc/internal/core/ignore"
)

// 默认上传上限供调用方选用，保存入口实际执行传入的 maxBytes。
// 错误代码与展示文本由各操作封装为本地化错误，不在此修改界面翻译。
const (
	DefaultMaxUploadSizeBytes int64 = 50 * 1024 * 1024
	errInvalidAssetName             = "Invalid asset name"
	errInvalidAssetNameFmt          = "invalid asset name %s"
	errAssetNotFound                = "Asset not found"
	errAssetNotFoundFmt             = "asset %s not found"
)

// AssetService 维护附件目录、文件名生成器和页面忽略规则。
// 写操作持独占锁，列表持读锁；该锁只协调同一服务对象，不锁定外部文件写入。
type AssetService struct {
	assetsDir   string
	storageDir  string
	slugger     *tree.SlugService
	log         *slog.Logger
	ignoreCache *ignore.Cache

	mu sync.RWMutex
}

// SetIgnoreCache 设置按目录继承忽略规则的共享缓存，此赋值不使用附件读写锁。
func (s *AssetService) SetIgnoreCache(ignoreCache *ignore.Cache) {
	s.ignoreCache = ignoreCache
}

// getIgnoreForDir 从共享缓存取得目录对应的规则，未设置缓存时不应用忽略规则。
func (s *AssetService) getIgnoreForDir(dir string) *ignore.IgnoreFile {
	if s.ignoreCache == nil {
		return nil
	}
	return s.ignoreCache.Get(dir)
}

// isPageIgnored 生成页面路径，移除 root/ 开头并在父目录取得继承规则。
// 节点类型决定按目录还是文件匹配；空页面指针在此返回未忽略，不代表后续操作接受空指针。
func (s *AssetService) isPageIgnored(page *tree.PageNode) bool {
	if page == nil {
		return false
	}
	relPath := filepath.ToSlash(tree.GeneratePathFromPageNode(page))
	relPath = strings.TrimPrefix(relPath, "root/")
	dir := filepath.Join(s.storageDir, "root", filepath.Dir(relPath))
	ig := s.getIgnoreForDir(dir)
	if ig == nil {
		return false
	}
	isDir := page.Kind == tree.NodeKindSection
	return ig.Matches(relPath, isDir)
}

// assetPageDiskPath 统一目录分隔符后追加页面 ID，不使用页面的 Slug。
func assetPageDiskPath(assetsDir string, pageID string) string {
	normalizedAssetsDir := filepath.FromSlash(strings.ReplaceAll(assetsDir, `\`, `/`))
	return filepath.Join(normalizedAssetsDir, pageID)
}

// assetFileDiskPath 统一附件目录的分隔符并拼接文件名，输入合法性由具体操作检查。
func assetFileDiskPath(assetPath string, filename string) string {
	normalizedAssetPath := filepath.FromSlash(strings.ReplaceAll(assetPath, `\`, `/`))
	return filepath.Join(normalizedAssetPath, filename)
}

// validateFilename 拒绝空文本、正反斜杠以及单独的点和双点路径段。
// 此处不验证扩展名、路径段格式或操作系统的其他文件名限制。
func validateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename must not be empty")
	}
	if strings.ContainsAny(filename, "/\\") {
		return fmt.Errorf("filename must not contain path separators")
	}
	if filename == "." || filename == ".." {
		return fmt.Errorf("filename must not be a dot component")
	}
	return nil
}

// NewAssetService 创建存储根目录及 assets 子目录，任一步失败均直接 panic。
// 构造时不启用忽略缓存，调用方通过 SetIgnoreCache 接入共享规则。
func NewAssetService(storageDir string, slugger *tree.SlugService) *AssetService {
	// 目录权限参数用于创建目录，已存在的目录不会因此重新设置权限。
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		panic(fmt.Sprintf("could not create storage directory: %v", err))
	}
	// 所有页面附件集中在此子目录内，再按页面 ID 划分。
	assetsDir := filepath.Join(storageDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		panic(fmt.Sprintf("could not create assets directory: %v", err))
	}

	return &AssetService{
		assetsDir:  assetsDir,
		storageDir: storageDir,
		slugger:    slugger,
		log:        slog.Default().With("component", "AssetService"),
	}
}

// GetAssetsDir 返回附件根目录的磁盘路径，不创建目录或生成公开地址。
func (s *AssetService) GetAssetsDir() string {
	return s.assetsDir
}

// ensureAssetPagePathExists 在页面附件目录不存在时创建它。
// Stat 的其他错误不在此返回，后续实际读写会再接触该路径。
func (s *AssetService) ensureAssetPagePathExists(page *tree.PageNode) (string, error) {
	pagePath := assetPageDiskPath(s.assetsDir, page.ID)
	// 只把不存在作为创建条件，不在这里确认已有路径是否为目录。
	if _, err := os.Stat(pagePath); os.IsNotExist(err) {
		// 同时创建缺失的上级目录，创建失败时附带原始错误。
		if err := os.MkdirAll(pagePath, 0755); err != nil {
			return "", fmt.Errorf("could not create page path: %w", err)
		}
	}

	return pagePath, nil
}

// getAssetPagePath 返回页面 ID 对应路径，只在 Stat 判定不存在时返回错误。
// 权限等其他 Stat 错误不在此分支中转成失败。
func (s *AssetService) getAssetPagePath(page *tree.PageNode) (string, error) {
	pagePath := assetPageDiskPath(s.assetsDir, page.ID)

	// 此入口不创建页面附件目录。
	if _, err := os.Stat(pagePath); os.IsNotExist(err) {
		return "", fmt.Errorf("page path does not exist: %w", err)
	}

	return pagePath, nil
}

// buildPublicPath 使用 URL 的正斜杠规则生成 /assets/页面ID/文件名 地址。
func (s *AssetService) buildPublicPath(page *tree.PageNode, filename string) string {
	return "/" + path.Join("assets", page.ID, filename)
}

// SaveAssetForPage 在页面 ID 目录内生成未占用文件名，以原子流写入保存附件，
// 成功后返回公开地址。文件句柄由调用方提供并关闭，此方法不关闭上传流。
func (s *AssetService) SaveAssetForPage(page *tree.PageNode, file multipart.File, originalFilename string, maxBytes int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 被忽略页面在创建附件目录和读取上传流之前直接拒绝。
	if s.isPageIgnored(page) {
		return "", sharederrors.NewLocalizedError("asset_upload_failed", "Failed to upload asset", "page is excluded by .leafdocignore", nil)
	}

	uploadPath, err := s.ensureAssetPagePathExists(page)
	if err != nil {
		return "", sharederrors.NewLocalizedError("asset_upload_failed", "Failed to upload asset", "failed to upload asset", err)
	}

	// 读取现有名称供重名处理使用。此处忽略目录读取错误，失败时按空列表处理。
	entries, _ := os.ReadDir(uploadPath)
	existing := make([]string, 0, len(entries))
	for _, e := range entries {
		existing = append(existing, e.Name())
	}

	// 文件名生成器处理名称和重名序号，磁盘位置仍由页面 ID 决定。
	finalFilename := s.slugger.GenerateUniqueFilename(existing, originalFilename)
	fullPath := assetFileDiskPath(uploadPath, finalFilename)

	// 写入工具执行字节上限和临时文件清理，超限错误使用单独的业务错误代码。
	if err := shared.WriteStreamAtomic(fullPath, file, maxBytes, 0o644); err != nil {
		if errors.Is(err, shared.ErrFileTooLarge) {
			return "", sharederrors.NewLocalizedError("asset_file_too_large", "File is too large", "file is too large", err)
		}
		return "", sharederrors.NewLocalizedError("asset_upload_failed", "Failed to upload asset", "failed to upload asset", err)
	}

	// 只在原子写入成功后返回使用最终文件名的公开地址。
	return s.buildPublicPath(page, finalFilename), nil
}

// ListAssetsForPage 返回页面附件的公开地址列表，不是磁盘绝对路径。
// 缺失目录及读取失败都归为空列表，不检查页面忽略规则；子目录不列入结果。
func (s *AssetService) ListAssetsForPage(page *tree.PageNode) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pagePath, err := s.getAssetPagePath(page)
	if err != nil {
		return []string{}, nil
	}

	files, err := os.ReadDir(pagePath)
	if err != nil && !os.IsNotExist(err) {
		return []string{}, nil
	}

	// ReadDir 返回的文件名顺序直接保留，结果以非空切片的空值开始。
	result := []string{}
	for _, f := range files {
		if !f.IsDir() {
			result = append(result, s.buildPublicPath(page, f.Name()))
		}
	}

	return result, nil
}

// DeleteAsset 校验单个文件名后删除附件，成功后尝试清理空的页面附件目录。
// 文件不存在使用专用错误，清理空目录失败不改变删除成功的返回值。
func (s *AssetService) DeleteAsset(page *tree.PageNode, filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 忽略规则优先于名称检查，被排除页面不允许单文件删除。
	if s.isPageIgnored(page) {
		return sharederrors.NewLocalizedError("asset_delete_failed", "Failed to delete asset", "page is excluded by .leafdocignore", nil)
	}

	if err := validateFilename(filename); err != nil {
		return sharederrors.NewLocalizedError("asset_invalid_name", errInvalidAssetName, errInvalidAssetNameFmt, nil, filename)
	}

	assetPath, err := s.getAssetPagePath(page)
	if err != nil {
		return sharederrors.NewLocalizedError("asset_not_found", errAssetNotFound, errAssetNotFoundFmt, nil, filename)
	}

	fullPath := assetFileDiskPath(assetPath, filename)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return sharederrors.NewLocalizedError("asset_not_found", errAssetNotFound, errAssetNotFoundFmt, nil, filename)
		}
		return sharederrors.NewLocalizedError("asset_delete_failed", "Failed to delete asset", "failed to delete asset %s", err, filename)
	}

	// 仅在读取目录成功且已空时尝试移除目录，不处理目录读取失败。
	files, err := os.ReadDir(assetPath)
	if err == nil && len(files) == 0 {
		_ = os.Remove(assetPath) // 目录清理失败不撤销已经完成的文件删除。
	}

	return nil
}

// DeleteAllAssetsForPage 递归删除页面附件目录；被忽略页面直接跳过。
// 路径获取失败或 Stat 失败均返回成功，只有执行 RemoveAll 的错误向上传递。
func (s *AssetService) DeleteAllAssetsForPage(page *tree.PageNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 此批量入口对忽略页面采用跳过策略，不与单文件删除使用同一种错误返回。
	if s.isPageIgnored(page) {
		return nil // 保留现有附件，不进入磁盘删除流程。
	}

	assetDir, err := s.getAssetPagePath(page)
	if err != nil {
		// 页面附件目录不存在时无需执行递归删除。
		return nil
	}
	if _, err := os.Stat(assetDir); err == nil {
		return os.RemoveAll(assetDir)
	}
	return nil
}

// RenameAsset 在同一页面目录中重命名附件，保持扩展名原值，
// 新名称的主体必须通过路径段校验，目标存在时返回冲突而不覆盖。
func (s *AssetService) RenameAsset(page *tree.PageNode, oldFilename, newFilename string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 被忽略页面不允许重命名，旧名称随后通过单文件名检查。
	if s.isPageIgnored(page) {
		return "", sharederrors.NewLocalizedError("asset_rename_failed", "Failed to rename asset", "page is excluded by .leafdocignore", nil)
	}

	if err := validateFilename(oldFilename); err != nil {
		return "", sharederrors.NewLocalizedError("asset_invalid_name", errInvalidAssetName, errInvalidAssetNameFmt, nil, oldFilename)
	}

	assetPath, err := s.getAssetPagePath(page)
	if err != nil {
		return "", sharederrors.NewLocalizedError("asset_not_found", errAssetNotFound, errAssetNotFoundFmt, nil, oldFilename)
	}

	oldFullPath := assetFileDiskPath(assetPath, oldFilename)
	newFullPath := assetFileDiskPath(assetPath, newFilename)

	// 扩展名直接比较文本，大小写不同也算发生变化。
	oldExt := path.Ext(oldFilename)
	newExt := path.Ext(newFilename)
	if oldExt != newExt {
		return "", sharederrors.NewLocalizedError("asset_invalid_extension", "Asset extension must not change", "asset extension must not change from %s", nil, oldExt)
	}

	// 去掉扩展名再校验主体，避免扩展名中的点参与路径段检查。
	newFilenameWithoutExt := newFilename[:len(newFilename)-len(newExt)]
	// 此入口要求主体本来就是合法路径段，不主动将用户输入转换为合法名称。
	if err := s.slugger.IsValidSlug(newFilenameWithoutExt); err != nil {
		return "", sharederrors.NewLocalizedError("asset_invalid_name", errInvalidAssetName, errInvalidAssetNameFmt, nil, newFilename)
	}

	// 先检查目标是否占用，其他 Stat 错误直接返回；同名重命名也会命中此冲突条件。
	if _, statErr := os.Stat(newFullPath); statErr == nil {
		return "", sharederrors.NewLocalizedError("asset_already_exists", "Asset already exists", "asset %s already exists", nil, newFilename)
	} else if !os.IsNotExist(statErr) {
		return "", sharederrors.NewLocalizedError("asset_rename_failed", "Failed to rename asset", "failed to rename asset %s", statErr, oldFilename)
	}

	if _, err := os.Stat(oldFullPath); os.IsNotExist(err) {
		return "", sharederrors.NewLocalizedError("asset_not_found", errAssetNotFound, errAssetNotFoundFmt, nil, oldFilename)
	}

	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		return "", sharederrors.NewLocalizedError("asset_rename_failed", "Failed to rename asset", "failed to rename asset %s", err, oldFilename)
	}

	return s.buildPublicPath(page, newFilename), nil
}

// CopyAllAssets 把源页面目录中的非目录项按原名复制到目标页面目录。
// 任一页面被忽略或源目录不存在时跳过，复制中途失败不回滚已经复制的文件。
func (s *AssetService) CopyAllAssets(sourcePage *tree.PageNode, targetPage *tree.PageNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 源和目标都应用忽略规则，跳过时不创建目标附件目录。
	if s.isPageIgnored(sourcePage) || s.isPageIgnored(targetPage) {
		return nil // 不修改任一页面的附件。
	}

	sourceAssetPath, err := s.getAssetPagePath(sourcePage)
	if err != nil {
		// 获取不到源附件目录时直接结束，不读取目标目录。
		return nil
	}

	targetAssetPath, err := s.ensureAssetPagePathExists(targetPage)
	if err != nil {
		return fmt.Errorf("could not create target asset path: %w", err)
	}

	entries, err := os.ReadDir(sourceAssetPath)
	if err != nil {
		return fmt.Errorf("could not read source asset directory: %w", err)
	}

	// 只复制当前目录一层，不递归复制子目录，也不生成避开重名的目标名称。
	for _, entry := range entries {
		if entry.IsDir() {
			continue // 子目录不作为单个附件复制。
		}
		if err := s.copySingleAsset(sourceAssetPath, targetAssetPath, entry); err != nil {
			return fmt.Errorf("could not copy asset %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// copySingleAsset 打开源文件并创建目标文件，再完整复制流。os.Create 会截断已有
// 同名目标；写入不是原子替换，复制失败后不在此删除不完整目标文件。
// 两个句柄均安排关闭，关闭失败只记录警告，不改变复制函数的返回结果。
func (s *AssetService) copySingleAsset(sourceAssetPath string, targetAssetPath string, entry os.DirEntry) error {
	sourceFilePath := assetFileDiskPath(sourceAssetPath, entry.Name())
	targetFilePath := assetFileDiskPath(targetAssetPath, entry.Name())

	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return fmt.Errorf("could not open source asset file: %w", err)
	}
	defer func() {
		err := sourceFile.Close()
		if err != nil {
			s.log.Warn("failed to close source file", "file", sourceFilePath, "error", err)
		}
	}()

	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		return fmt.Errorf("could not create target asset file: %w", err)
	}
	defer func() {
		err := targetFile.Close()
		if err != nil {
			s.log.Warn("failed to close target file", "file", targetFilePath, "error", err)
		}
	}()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("could not copy asset file: %w", err)
	}

	return nil
}
