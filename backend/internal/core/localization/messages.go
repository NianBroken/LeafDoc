// localization 包从嵌入词条表生成接口错误和邮件使用的提示文本。
// 本包按调用时的语言提供函数查询词条，不修改文档正文或账号标识。
package localization

import (
	"embed"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
)

// files 嵌入当前目录中的 JSON 词条，load 只加载明确列出的五种语言。
// 下方的 go:embed 保持工具要求的原文；嵌入目录的文件集合与 load 的明确语言列表分别维护。
//
//go:embed *.json
var files embed.FS

// 下方指令供 go generate 调用词条同步脚本，正常运行程序时不会执行。
// mu 保护语言提供函数的读写，不在调用该函数时持有锁。
//
//go:generate node ../../../../other/scripts/sync-error-catalogs.mjs
var mu sync.RWMutex

// language 初始返回中文，调用方可安装读取当前实例设置的函数。
var language = func() string { return "zh" }

// SetLanguageProvider 替换语言提供函数，写入期间使用互斥锁。
// 本函数不验证 provider 是否为 nil，也不缓存它返回的语言。
func SetLanguageProvider(provider func() string) { mu.Lock(); language = provider; mu.Unlock() }

// Language 在读锁内取得当前提供函数，释放锁后调用并返回其结果。
// 提供函数的内部并发安全和有效返回值由安装它的调用方负责。
func Language() string { mu.RLock(); fn := language; mu.RUnlock(); return fn() }

// 词条在包初始化期间加载，加载失败会触发 panic，不延迟到第一次请求。
var catalogues = load()

// numbers 匹配连续 ASCII 数字，供无显式参数时提取原提示中的数值。
var numbers = regexp.MustCompile(`[0-9]+`)

// load 将五种语言分别解析为模板到文本的映射，不对词条键和值作额外验证。
func load() map[string]map[string]string {
	result := map[string]map[string]string{}
	for _, lang := range []string{"zh", "en", "de", "es", "fr"} {
		data, err := files.ReadFile(lang + ".json")
		if err != nil {
			panic(err)
		}
		var v map[string]string
		if err = json.Unmarshal(data, &v); err != nil {
			panic(err)
		}
		result[lang] = v
	}
	return result
}

// Message 依次查找原模板、转成小写的模板和数字归一化后的模板。
// 数字归一化仅在没有显式参数时执行，将原文中的连续数字按顺序作为替换参数。
// 找到词条后，每个参数替换当前结果中的第一个 %s，多余占位符保持原样。
// 缺少词条或词条为空时返回原模板，不回退到其他语言，也不自行翻译参数。
// 每次查询均重新调用 Language，提供函数在一次调用内应保持返回值稳定。
func Message(template string, args ...string) string {
	value := catalogues[Language()][template]
	if value == "" {
		value = catalogues[Language()][strings.ToLower(template)]
	}
	if value == "" && len(args) == 0 {
		key := numbers.ReplaceAllString(strings.ToLower(template), "%s")
		value = catalogues[Language()][key]
		if value != "" {
			args = numbers.FindAllString(template, -1)
		}
	}
	if value == "" {
		return template
	}
	// 顺序替换作用于上一次结果，参数自身包含 %s 时也会参与后续参数的替换。
	for _, arg := range args {
		value = strings.Replace(value, "%s", arg, 1)
	}
	return value
}

// Fallback 将状态码映射到通用词条后交给 Message 生成提示。
// 小于 400 使用完成提示，已列出的客户端错误分别分类，五百及以上使用服务失败。
// 其余四百段状态保留通用请求失败提示，不检查响应正文。
func Fallback(status int) string {
	key := "request failed"
	switch {
	case status < 400:
		key = "operation completed"
	case status == 400 || status == 422:
		key = "invalid request input"
	case status == 401:
		key = "authentication required"
	case status == 403:
		key = "permission denied"
	case status == 404:
		key = "resource not found"
	case status == 409:
		key = "resource conflict"
	case status == 413:
		key = "uploaded file too large"
	case status == 429:
		key = "too many requests"
	case status >= 500:
		key = "service request failed"
	}
	return Message(key)
}
