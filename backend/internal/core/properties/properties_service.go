// 本文件连接正文解析与属性存储，并定义哪些前置字段能够成为检索属性。
package properties

import (
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/markdown"
)

// PropertiesService 通过存储接口读写派生属性，正文解析不改变原始页面内容。
type PropertiesService struct {
	store *PropertiesStore
}

// NewPropertiesService 绑定调用方提供的属性存储，不在构造时建立索引。
func NewPropertiesService(store *PropertiesStore) *PropertiesService {
	return &PropertiesService{store: store}
}

// Close 将连接释放交给存储对象，返回其关闭结果。
func (s *PropertiesService) Close() error {
	return s.store.Close()
}

// ClearIndex 清空属性索引，页面正文和其他种类的索引不受此调用影响。
func (s *PropertiesService) ClearIndex() error {
	return s.store.Clear()
}

// IndexPageContent 提取正文前置字段后整页替换属性。
// 无合法属性时传入 nil，存储层仍会删除该页已有属性。
func (s *PropertiesService) IndexPageContent(pageID, rawContent string) error {
	props := ExtractPropertiesFromContent(rawContent)
	return s.store.SetPropertiesForPage(pageID, props)
}

// SetPropertiesForPage 原样转交属性集合，不执行提取路径中的键过滤和值校验。
func (s *PropertiesService) SetPropertiesForPage(pageID string, props map[string]PropertyEntry) error {
	return s.store.SetPropertiesForPage(pageID, props)
}

// DeletePropertiesForPage 清理单页属性，不触碰页面本身。
func (s *PropertiesService) DeletePropertiesForPage(pageID string) error {
	return s.store.DeletePropertiesForPage(pageID)
}

// GetAllPropertyKeys 读取前缀匹配的属性键及页面数量，排序与限制交给存储层。
func (s *PropertiesService) GetAllPropertyKeys(filter string, limit int) ([]PropertyKeyCount, error) {
	return s.store.GetAllPropertyKeys(filter, limit)
}

// GetPageIDsByProperty 返回同时满足键和值等值条件的页面 ID。
func (s *PropertiesService) GetPageIDsByProperty(key, value string) ([]string, error) {
	return s.store.GetPageIDsByProperty(key, value)
}

// GetPropertiesForPages 批量读取属性映射，不为无记录的页面合成空记录。
func (s *PropertiesService) GetPropertiesForPages(pageIDs []string) (map[string]map[string]PropertyEntry, error) {
	return s.store.GetPropertiesForPages(pageIDs)
}

// ExtractPropertiesFromContent 解析前置字段，只收集非系统键下的非空单行字符串。
// 数字、布尔值、列表和空值均不转换为文本，嵌套字符串用点号连接父子键。
// 解析失败、没有前置字段或没有可收集的结果时返回 nil，不把解析错误向上传递。
func ExtractPropertiesFromContent(content string) map[string]PropertyEntry {
	fm, _, has, err := markdown.ParseFrontmatter(content)
	if err != nil || !has || len(fm.ExtraFields) == 0 {
		return nil
	}

	result := make(map[string]PropertyEntry)
	for rawKey, value := range fm.ExtraFields {
		key := strings.TrimSpace(rawKey)
		if markdown.IsSystemKey(key) {
			continue
		}
		extractFlatEntry(key, value, result)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// maxNestedPropertyDepth 限制递归访问深度，深度达到 20 的值直接跳过。
const maxNestedPropertyDepth = 20

// extractFlatEntry 从深度零开始展开字段，例如 meta 下的 author 写入 meta.author。
// 仅字符串和字符串键映射参与处理，其他动态类型直接跳过。
func extractFlatEntry(prefix string, value interface{}, result map[string]PropertyEntry) {
	extractFlatEntryDepth(prefix, value, result, 0)
}

// extractFlatEntryDepth 遍历子字段并保留每个完整键首次写入的有效值。
// 映射遍历不承诺顺序，带点号的原始键与嵌套键发生同名时不另设优先级。
func extractFlatEntryDepth(prefix string, value interface{}, result map[string]PropertyEntry, depth int) {
	if depth >= maxNestedPropertyDepth {
		return
	}
	switch v := value.(type) {
	case string:
		if _, exists := result[prefix]; !exists {
			entry, ok := toPropertyEntry(v)
			if ok {
				result[prefix] = entry
			}
		}
	case map[string]interface{}:
		for childKey, childValue := range v {
			childKey = strings.TrimSpace(childKey)
			if childKey == "" {
				continue
			}
			// 子键去除边缘空白后，空键以及忽略大小写的 leafdoc_ 前缀均不进入索引。
			// 子级在这里仅检查保留前缀，不套用顶层的全部系统键判断。
			if strings.HasPrefix(strings.ToLower(childKey), "leafdoc_") {
				continue
			}
			extractFlatEntryDepth(prefix+"."+childKey, childValue, result, depth+1)
		}
	}
}

// toPropertyEntry 接收字符串并裁去边缘空白，拒绝空串和含换行符的内容。
// 合格内容保留内部空白并标记为 text，不对数字或其他类型进行格式转换。
func toPropertyEntry(value interface{}) (PropertyEntry, bool) {
	s, ok := value.(string)
	if !ok {
		return PropertyEntry{}, false
	}
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsRune(s, '\n') {
		return PropertyEntry{}, false
	}
	return PropertyEntry{Value: s, Type: "text"}, true
}

// WithExecutor 让属性写入使用页面事务的执行器，提交由外层业务工作单元负责。
func (s *PropertiesService) WithExecutor(executor database.Executor) *PropertiesService {
	return &PropertiesService{store: s.store.WithExecutor(executor)}
}
