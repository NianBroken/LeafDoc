// 本文件协调站点名称、品牌资源文件和数据库配置，服务锁串行保护更新操作。
package branding

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// 删除错误提示由标志和站点图标的对应失败分支共用，不作为错误分类码。
const (
	errFailedToDeleteLogo    = "Failed to delete logo"
	errFailedToDeleteFavicon = "Failed to delete favicon"
)

// BrandingService 持有配置存储和读写锁，文件操作与数据库保存按方法内顺序完成。
// 两种存储不组成同一事务，后续步骤失败不会自动回滚已经完成的文件变更。
type BrandingService struct {
	store *BrandingStore
	mu    sync.RWMutex
}

// NewBrandingService 先加载数据库配置，再创建磁盘资源目录，任一步失败均返回错误。
func NewBrandingService(storageDir string) (*BrandingService, error) {
	store, err := NewBrandingStore(storageDir)
	if err != nil {
		return nil, err
	}

	// 配置记录与资源文件分别存储，这里仅保证 branding 资源目录存在。
	assetsDir := store.brandingAssetsDir()
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create branding assets directory: %w", err)
	}

	return &BrandingService{
		store: store,
	}, nil
}

// Reload 持写锁从 MySQL 重新加载配置，供恢复流程刷新进程内缓存。
// 此方法不从磁盘 JSON 文件读取，也不扫描或重建资源目录。
func (s *BrandingService) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.store.Reload()
}

// GetBranding 持读锁获取配置副本，为两类资源选择当前可用文件后生成响应。
// 资源发现只改变本次副本，不自动把发现的文件名写回数据库。
func (s *BrandingService) GetBranding() (*BrandingConfigResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := s.store.Load()
	cfg.LogoFile = s.availableAsset(cfg.LogoFile, "logo", cfg.BrandingConstraints.LogoExts)
	cfg.FaviconFile = s.availableAsset(cfg.FaviconFile, "favicon", cfg.BrandingConstraints.FaviconExts)
	return cfg.ToResponse(), nil
}

// availableAsset 优先使用安全、扩展名允许、非空的普通配置文件。
// 不满足条件时按字典序扫描 prefix.* 并取首个允许的非空普通文件，失败或无匹配返回空串。
// 查询只核验路径、类型和大小，不解码图像内容。
func (s *BrandingService) availableAsset(configured, prefix string, allowed map[string]bool) string {
	assetsDir := s.store.brandingAssetsDir()
	if configured != "" && !containsPathTraversal(configured) && allowed[strings.ToLower(filepath.Ext(configured))] {
		if info, err := os.Stat(filepath.Join(assetsDir, configured)); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
			return configured
		}
	}
	matches, err := filepath.Glob(filepath.Join(assetsDir, prefix+".*"))
	if err != nil {
		return ""
	}
	sort.Strings(matches)
	for _, path := range matches {
		ext := strings.ToLower(filepath.Ext(path))
		info, statErr := os.Stat(path)
		if statErr == nil && info.Mode().IsRegular() && info.Size() > 0 && allowed[ext] {
			return filepath.Base(path)
		}
	}
	return ""
}

// UpdateBranding 去掉站点名称两端空白，校验后只更新名称，保留资源文件配置。
// 校验按空值、字节长度、控制字符顺序匹配，最多添加一条字段错误。
func (s *BrandingService) UpdateBranding(siteName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.store.Load()

	// len 使用 UTF-8 字节数，长度提示中的字符数不代表按 Unicode 字符计数。
	ve := sharederrors.NewValidationErrors()
	trimmedSiteName := strings.TrimSpace(siteName)

	switch {
	case trimmedSiteName == "":
		ve.Add("siteName", "Site name must not be empty")
	case len(trimmedSiteName) > cfg.BrandingConstraints.MaxSiteNameLength:
		ve.Add("siteName", fmt.Sprintf("Site name must not exceed %d characters", cfg.BrandingConstraints.MaxSiteNameLength))
	case containsControlCharacters(trimmedSiteName):
		ve.Add("siteName", "Site name contains invalid control characters")
	}

	if ve.HasErrors() {
		return ve
	}

	cfg.SiteName = trimmedSiteName

	if err := s.store.Save(cfg); err != nil {
		return sharederrors.NewLocalizedError(
			"branding_update_failed",
			"Failed to update branding",
			"failed to update branding",
			err,
		)
	}

	return nil
}

// ContainsUnsafePath 拒绝包含两个连续点、任一斜线、空字节或平台绝对路径和卷名的文本。
// 两个连续点即使只是文件名的一部分也会被拒绝，空串本身不会命中这些条件。
// filepath 的绝对路径和卷名判断遵循程序运行平台。
func ContainsUnsafePath(s string) bool {
	return strings.Contains(s, "..") ||
		strings.Contains(s, "/") ||
		strings.Contains(s, "\\") ||
		filepath.IsAbs(s) ||
		filepath.VolumeName(s) != "" ||
		strings.Contains(s, "\x00")
}

// containsPathTraversal 直接复用导出的路径片段校验，保持包内外使用相同规则。
func containsPathTraversal(s string) bool { return ContainsUnsafePath(s) }

// containsControlCharacters 按 Unicode 字符扫描控制字符，制表、换行和回车作为例外允许。
// 其他控制字符遇到一个就返回 true，不对普通空格或可见字符作额外限制。
func containsControlCharacters(s string) bool {
	for _, r := range s {
		// unicode.IsControl 的范围由标准库定义，三个常用空白字符单独排除。
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

// writeAsset 创建资源目录后按字节上限写入临时文件并替换目标，不关闭传入的上传文件。
func (s *BrandingService) writeAsset(targetPath string, file multipart.File, maxBytes int64) error {
	if err := os.MkdirAll(s.store.brandingAssetsDir(), 0o755); err != nil {
		return err
	}
	return shared.WriteStreamAtomic(targetPath, file, maxBytes, 0o644)
}

// UploadLogo 持写锁校验扩展名，以 logo 加小写扩展名保存原始上传字节。
// 写入成功后清理其他 logo.*，再保存数据库文件名，不解码或重编码图片。
// 清理错误只记录日志，配置保存失败也不会撤销已经写入的资源。
func (s *BrandingService) UploadLogo(file multipart.File, filename string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	assetsDir := s.store.brandingAssetsDir()
	ext := strings.ToLower(filepath.Ext(filename))

	cfg := s.store.Load()

	if !cfg.IsAllowedLogoExt(filename) {
		allowedExts := cfg.AllowedLogoExtsAsString()
		return "", sharederrors.NewLocalizedError(
			"branding_logo_invalid_type",
			"Invalid logo file type",
			"invalid logo file type %s (allowed: %s)",
			nil,
			ext,
			allowedExts,
		)
	}

	targetPath := filepath.Join(assetsDir, "logo"+ext)

	// 文件写入失败立即结束，不清理其他资源，也不修改配置记录。
	if err := s.writeAsset(targetPath, file, cfg.BrandingConstraints.MaxLogoSize); err != nil {
		return "", sharederrors.NewLocalizedError(
			"branding_logo_upload_failed",
			"Failed to save logo file",
			"failed to save logo file",
			err,
		)
	}

	// 目标提交成功后清理同类文件，保留当前目标路径。
	removeOtherMatches(filepath.Join(assetsDir, "logo.*"), targetPath)

	// 更新配置副本，再由 Store 在数据库保存成功后更新缓存。
	cfg.LogoFile = "logo" + ext
	if err := s.store.Save(cfg); err != nil {
		return "", sharederrors.NewLocalizedError(
			"branding_logo_upload_failed",
			"Failed to save logo file",
			"failed to save logo file",
			err,
		)
	}

	return cfg.LogoFile, nil
}

// DeleteLogo 优先使用配置文件名，配置为空时按可用资源规则发现一项文件。
// 删除目标后清空配置，目标不存在也允许继续；配置保存失败不恢复已删除文件。
func (s *BrandingService) DeleteLogo() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.store.Load()
	// 只有空配置才扫描目录，非空但失效的配置仍按其指定路径处理。
	if cfg.LogoFile == "" {
		cfg.LogoFile = s.availableAsset("", "logo", cfg.BrandingConstraints.LogoExts)
	}

	if cfg.LogoFile == "" {
		return nil // 没有配置名也没有发现可用标志时直接成功，不写数据库。
	}

	if containsPathTraversal(cfg.LogoFile) {
		return sharederrors.NewLocalizedError(
			"branding_logo_delete_failed",
			errFailedToDeleteLogo,
			"invalid logo file path",
			nil,
		)
	}

	logoPath := filepath.Join(s.store.brandingAssetsDir(), cfg.LogoFile)
	if err := os.Remove(logoPath); err != nil && !os.IsNotExist(err) {
		return sharederrors.NewLocalizedError(
			"branding_logo_delete_failed",
			errFailedToDeleteLogo,
			"failed to delete logo",
			err,
		)
	}

	cfg.LogoFile = ""
	if err := s.store.Save(cfg); err != nil {
		return sharederrors.NewLocalizedError(
			"branding_logo_delete_failed",
			errFailedToDeleteLogo,
			"failed to delete logo",
			err,
		)
	}

	return nil
}

// UploadFavicon 按图标允许扩展名和字节上限保存原始上传内容。
// 顺序为写文件、清理其他 favicon.*、保存配置，后续失败不回滚前面的文件操作。
func (s *BrandingService) UploadFavicon(file multipart.File, filename string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	assetsDir := s.store.brandingAssetsDir()
	ext := strings.ToLower(filepath.Ext(filename))

	cfg := s.store.Load()

	if !cfg.IsAllowedFaviconExt(filename) {
		allowedExts := cfg.AllowedFaviconExtsAsString()
		return "", sharederrors.NewLocalizedError(
			"branding_favicon_invalid_type",
			"Invalid favicon file type",
			"invalid favicon file type %s (allowed: %s)",
			nil,
			ext,
			allowedExts,
		)
	}

	targetPath := filepath.Join(assetsDir, "favicon"+ext)

	// 固定前缀加小写扩展名决定目标，原上传文件名不会作为路径使用。
	if err := s.writeAsset(targetPath, file, cfg.BrandingConstraints.MaxFaviconSize); err != nil {
		return "", sharederrors.NewLocalizedError(
			"branding_favicon_upload_failed",
			"Failed to save favicon file",
			"failed to save favicon file",
			err,
		)
	}

	// 写入成功后删除其他扩展名的同类资源，删除失败仅记录日志。
	removeOtherMatches(filepath.Join(assetsDir, "favicon.*"), targetPath)

	// 将目标文件名写入配置，数据库保存失败会返回错误但保留已提交的文件。
	cfg.FaviconFile = "favicon" + ext
	if err := s.store.Save(cfg); err != nil {
		return "", sharederrors.NewLocalizedError(
			"branding_favicon_upload_failed",
			"Failed to save favicon file",
			"failed to save favicon file",
			err,
		)
	}

	return cfg.FaviconFile, nil
}

// DeleteFavicon 删除配置或发现的一个图标并清空配置，不一次删除全部匹配文件。
// 路径不安全时在删除前返回错误，不存在的目标视为已删除。
func (s *BrandingService) DeleteFavicon() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.store.Load()
	// 空配置时复用资源发现顺序，非空配置不在此步骤验证文件实际存在。
	if cfg.FaviconFile == "" {
		cfg.FaviconFile = s.availableAsset("", "favicon", cfg.BrandingConstraints.FaviconExts)
	}

	if cfg.FaviconFile == "" {
		return nil // 没有可删除的图标时直接返回，不改变配置缓存。
	}

	if containsPathTraversal(cfg.FaviconFile) {
		return sharederrors.NewLocalizedError(
			"branding_favicon_delete_failed",
			errFailedToDeleteFavicon,
			"invalid favicon file path",
			nil,
		)
	}

	faviconPath := filepath.Join(s.store.brandingAssetsDir(), cfg.FaviconFile)
	if err := os.Remove(faviconPath); err != nil && !os.IsNotExist(err) {
		return sharederrors.NewLocalizedError(
			"branding_favicon_delete_failed",
			errFailedToDeleteFavicon,
			"failed to delete favicon",
			err,
		)
	}

	cfg.FaviconFile = ""
	if err := s.store.Save(cfg); err != nil {
		return sharederrors.NewLocalizedError(
			"branding_favicon_delete_failed",
			errFailedToDeleteFavicon,
			"failed to delete favicon",
			err,
		)
	}

	return nil
}

// GetBrandingAssetsDir 返回资源目录路径，不进行存在性检查或创建操作。
func (s *BrandingService) GetBrandingAssetsDir() string {
	return s.store.brandingAssetsDir()
}
