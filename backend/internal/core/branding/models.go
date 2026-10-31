// 本文件定义站点外观配置、上传限制和对客户端输出的结构。
package branding

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/brandassets"
)

// BrandingConfig 保存站点名称和自定义资源文件名，限制字段只在内存中使用。
// JSON 编码排除 BrandingConstraints，加载后由存储层注入默认限制。
type BrandingConfig struct {
	SiteName            string              `json:"siteName"`
	LogoFile            string              `json:"logoFile"`
	FaviconFile         string              `json:"faviconFile"`
	BrandingConstraints BrandingConstraints `json:"-"`
}

// ToResponse 复制名称与文件名，并把限制映射转为排序后的切片，避免输出顺序随 map 改变。
func (bc *BrandingConfig) ToResponse() *BrandingConfigResponse {
	return &BrandingConfigResponse{
		SiteName:            bc.SiteName,
		LogoFile:            bc.LogoFile,
		FaviconFile:         bc.FaviconFile,
		BrandingConstraints: bc.BrandingConstraints.ToResponse(),
	}
}

// AllowedLogoExts 收集标志扩展名映射的全部键并排序，不检查对应布尔值是否为 true。
func (bc *BrandingConfig) AllowedLogoExts() []string {
	extensions := []string{}
	for ext := range bc.BrandingConstraints.LogoExts {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

// AllowedLogoExtsAsString 将排序结果用逗号连接，不在分隔符后加入空格。
func (bc *BrandingConfig) AllowedLogoExtsAsString() string {
	return strings.Join(bc.AllowedLogoExts(), ",")
}

// AllowedFaviconExts 收集图标扩展名映射的全部键并排序，返回独立切片。
func (bc *BrandingConfig) AllowedFaviconExts() []string {
	extensions := []string{}
	for ext := range bc.BrandingConstraints.FaviconExts {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

// BrandingConstraints 保存上传扩展名、字节数上限和站点名称长度上限。
// 扩展名校验读取映射中的布尔值，站点名称的实际长度校验使用字符串字节数。
type BrandingConstraints struct {
	LogoExts          map[string]bool `json:"logoExts"`
	FaviconExts       map[string]bool `json:"faviconExts"`
	MaxLogoSize       int64           `json:"maxLogoSize"`
	MaxFaviconSize    int64           `json:"maxFaviconSize"`
	MaxSiteNameLength int             `json:"maxSiteNameLength"`
}

// ToResponse 将两个映射转换为排序列表，其余限制按值复制。
func (bc BrandingConstraints) ToResponse() BrandingConstraintsResponse {
	return BrandingConstraintsResponse{
		LogoExts:          bc.getSortedLogoExts(),
		FaviconExts:       bc.getSortedFaviconExts(),
		MaxLogoSize:       bc.MaxLogoSize,
		MaxFaviconSize:    bc.MaxFaviconSize,
		MaxSiteNameLength: bc.MaxSiteNameLength,
	}
}

// getSortedLogoExts 只遍历映射键，不筛选布尔值，空映射返回非 nil 空切片。
func (bc BrandingConstraints) getSortedLogoExts() []string {
	extensions := []string{}
	for ext := range bc.LogoExts {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

// getSortedFaviconExts 用字典序组织图标扩展名，使响应顺序稳定。
func (bc BrandingConstraints) getSortedFaviconExts() []string {
	extensions := []string{}
	for ext := range bc.FaviconExts {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

// AllowedFaviconExtsAsString 用逗号连接排序后的图标扩展名，供错误提示复用。
func (bc *BrandingConfig) AllowedFaviconExtsAsString() string {
	return strings.Join(bc.AllowedFaviconExts(), ",")
}

// BrandingConfigResponse 同时输出当前外观值和客户端可据以提示的上传限制。
type BrandingConfigResponse struct {
	SiteName            string                      `json:"siteName"`
	LogoFile            string                      `json:"logoFile"`
	FaviconFile         string                      `json:"faviconFile"`
	BrandingConstraints BrandingConstraintsResponse `json:"brandingConstraints"` // 供客户端展示和预先校验，服务端仍独立检查。
}

// BrandingConstraintsResponse 使用切片而非映射表示扩展名，便于客户端直接构造选择列表。
type BrandingConstraintsResponse struct {
	LogoExts          []string `json:"logoExts"`
	FaviconExts       []string `json:"faviconExts"`
	MaxLogoSize       int64    `json:"maxLogoSize"`
	MaxFaviconSize    int64    `json:"maxFaviconSize"`
	MaxSiteNameLength int      `json:"maxSiteNameLength"`
}

// DefaultBrandingConfig 每次构造独立配置和扩展名映射，默认没有自定义标志与图标文件。
// 两种上传上限均为一 MiB，站点名称限制为一百字节。
func DefaultBrandingConfig() *BrandingConfig {
	return &BrandingConfig{
		SiteName:    brandassets.SiteName,
		LogoFile:    "",
		FaviconFile: "",
		BrandingConstraints: BrandingConstraints{
			LogoExts:          map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true},
			FaviconExts:       map[string]bool{".png": true, ".gif": true, ".ico": true, ".webp": true},
			MaxLogoSize:       1 * 1024 * 1024, // 一 MiB，按二进制字节单位计算。
			MaxFaviconSize:    1 * 1024 * 1024, // 一 MiB，按二进制字节单位计算。
			MaxSiteNameLength: 100,
		},
	}
}

// IsAllowedLogoExt 按文件名取最后一个扩展名并转小写，读取允许映射，不检查文件内容。
func (bc *BrandingConfig) IsAllowedLogoExt(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return bc.BrandingConstraints.LogoExts[ext]
}

// IsAllowedFaviconExt 只校验小写扩展名，不解码图像，也不验证 MIME 类型。
func (bc *BrandingConfig) IsAllowedFaviconExt(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return bc.BrandingConstraints.FaviconExts[ext]
}
