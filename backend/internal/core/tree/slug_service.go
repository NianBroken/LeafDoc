package tree

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gosimple/slug"
)

// reservedSlugs 列出不能直接作为页面路径段的保留名称，验证时不区分大小写。
var reservedSlugs = map[string]bool{
	"e":        true,
	"edit":     true,
	"api":      true,
	"assets":   true,
	"branding": true,
	"index":    true,
	"users":    true,
	"user":     true,
	"login":    true,
	"history":  true,
	"settings": true,
}

// slugPattern 允许英文字母与数字组成的非空段，段与段之间只能有一个连字符。
// 因此首尾连字符、连续连字符及其他字符均不能通过匹配。
var slugPattern = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

// SlugService 集中提供路径段规范处理、保留名称检查和同级冲突避让。
// 实例没有可变字段，同级唯一性来自每次调用传入的父节点。
type SlugService struct {
}

// NewSlugService 返回无状态的路径名称服务，不读取页面树或磁盘。
func NewSlugService() *SlugService {
	return &SlugService{}
}

// GenerateUniqueChildSlug 规范处理期望名称，再按递增数字后缀避开无效名称和同级冲突。
// currentID 对应的子节点不构成冲突，供同一页面保留自身路径段。
// 规范处理后为空时返回空串，与 GenerateValidSlug 的无有效名称结果保持一致。
// 非空名称的冲突检查要求调用方提供非空父节点。
func (s *SlugService) GenerateUniqueChildSlug(parent *PageNode, currentID, desired string) string {
	slug := normalizeSlug(desired)
	if slug == "" {
		return ""
	}
	original := slug
	i := 1

	for hasSlugConflict(parent, currentID, slug) || s.IsValidSlug(slug) != nil {
		slug = fmt.Sprintf("%s-%d", original, i)
		i++
	}

	return slug
}

// IsValidSlug 验证非空、非保留名称以及字母数字和单连字符连接规则。
// 验证允许英文字母大写，不会规范处理或重写输入；首次失败直接返回对应错误。
func (s *SlugService) IsValidSlug(slug string) error {
	if slug == "" {
		return errors.New("slug must not be empty")
	}

	// 保留名称按小写键比较，不因输入采用大写形式而绕过。
	lowerSlug := strings.ToLower(slug)
	if reservedSlugs[lowerSlug] {
		return fmt.Errorf("slug '%s' is reserved", slug)
	}

	if !slugPattern.MatchString(slug) {
		return errors.New("slug must contain only letters, numbers and hyphens")
	}

	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return errors.New("slug must not start or end with a hyphen")
	}

	return nil
}

// GenerateValidSlug 将名称规范化，再通过递增数字后缀得到通过验证的路径段。
// 规范处理后为空直接返回空串，不访问页面树，因此不保证同级唯一。
func (s *SlugService) GenerateValidSlug(desired string) string {
	slug := normalizeSlug(desired)
	if slug == "" {
		return ""
	}

	original := slug
	i := 1
	for s.IsValidSlug(slug) != nil {
		slug = fmt.Sprintf("%s-%d", original, i)
		i++
	}

	return slug
}

// normalizeSlug 先将下划线换成连字符，再交给 slug.Make 生成适合地址使用的文字。
// 字符转写、大小写和其他标点的处理沿用该库实现，不在此重复定义。
func normalizeSlug(title string) string {
	return slug.Make(strings.ReplaceAll(title, "_", "-"))
}

// hasSlugConflict 逐个检查直接子节点，名称比较不区分大小写，并排除当前页面标识。
func hasSlugConflict(parent *PageNode, currentID string, slug string) bool {
	for _, child := range parent.Children {
		if strings.EqualFold(child.Slug, slug) && child.ID != currentID {
			return true
		}
	}
	return false
}

// NormalizePath 按正斜杠分段并跳过原始空段，逐段规范处理后重新连接。
// validate 为 true 时还验证每段，保留名称会报错而不自动添加后缀。
// validate 为 false 时不检查规范处理后为空的段，也不保证最终路由通过业务验证。
func (s *SlugService) NormalizePath(path string, validate bool) (string, error) {
	segments := make([]string, 0)

	for _, segment := range strings.Split(path, string("/")) {

		if segment == "" {
			continue
		}

		if validate {
			// 验证针对规范处理后的结果，错误文字仍保留原路径段以定位输入。
			seg := normalizeSlug(segment)
			if err := s.IsValidSlug(seg); err != nil {
				return "", fmt.Errorf("segment '%s' is not a valid slug: %v", segment, err)
			}
			segment = seg
		} else {
			segment = normalizeSlug(segment)
		}
		segments = append(segments, segment)
	}
	return strings.Join(segments, string("/")), nil
}

// NormalizePathToValidSlugs 将每个非空路径段转换为有效名称，保留名称采用数字后缀。
// 任意段转换为空使整个处理失败，所有原始段都为空时返回空路径且不报错。
// 此方法用于没有父节点上下文的路径映射，不检测页面树中的名称冲突。
func (s *SlugService) NormalizePathToValidSlugs(value string) (string, error) {
	segments := make([]string, 0)

	for _, segment := range strings.Split(value, "/") {
		if segment == "" {
			continue
		}

		safe := s.GenerateValidSlug(segment)
		if safe == "" {
			return "", fmt.Errorf("segment '%s' is not a valid slug: slug must not be empty", segment)
		}
		segments = append(segments, safe)
	}

	return strings.Join(segments, "/"), nil
}

// NormalizeFilenameToValidSlug 将最后一个扩展名之前的部分转换为有效路径名称。
// 扩展名按原文字接回，基础名称为空时报错，不检查文件是否存在。
func (s *SlugService) NormalizeFilenameToValidSlug(filename string) (string, error) {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]
	safe := s.GenerateValidSlug(base)
	if safe == "" {
		return "", fmt.Errorf("filename '%s' is not a valid slug: slug must not be empty", filename)
	}
	return safe + ext, nil
}

// GenerateUniqueFilename 保留扩展名，对基础名称规范处理后避开列表中的同名文件。
// 冲突比较区分大小写，数字后缀插在扩展名前，不调用页面保留名称验证。
func (s *SlugService) GenerateUniqueFilename(existing []string, desired string) string {
	ext := filepath.Ext(desired)
	base := desired[:len(desired)-len(ext)]
	slugged := normalizeSlug(base)
	name := slugged + ext
	i := 1

	// 先建立精确文件名集合，每次从原始基础名称尝试下一个递增后缀。
	conflicts := make(map[string]bool)
	for _, f := range existing {
		conflicts[f] = true
	}
	for conflicts[name] {
		name = fmt.Sprintf("%s-%d%s", slugged, i, ext)
		i++
	}

	return name
}
