// Package ignore 读取 .leafdocignore 并编译路径匹配规则，也提供祖先规则合并缓存。
// 实际匹配语义由 go-gitignore 提供，本包不自行实现模式语法。
package ignore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gitignore "github.com/sabhiram/go-gitignore"
)

// IgnoreFilename 为每层目录中的规则文件名，直接读取和缓存读取共用此名称。
const IgnoreFilename = ".leafdocignore"

// IgnoreFile 保存编译后的匹配器与非空非注释行数，计数不等于去重后的有效规则数。
type IgnoreFile struct {
	matcher      *gitignore.GitIgnore
	patternCount int
}

// LoadFromDir 只读取指定目录的一份规则，不合并祖先目录。
// 文件不存在时返回 nil 和 nil，目录占位、访问失败或读取失败返回带文件名的错误。
func LoadFromDir(dir string) (*IgnoreFile, error) {
	path := filepath.Join(dir, IgnoreFilename)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", IgnoreFilename, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, expected a file", IgnoreFilename)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", IgnoreFilename, err)
	}

	lines := strings.Split(string(raw), "\n")
	count := countPatterns(lines)
	matcher := gitignore.CompileIgnoreLines(lines...)

	return &IgnoreFile{matcher: matcher, patternCount: count}, nil
}

// countPatterns 去掉行尾回车与两端空白后统计规则行，空行和 # 开头的行不计数。
// 原始行仍交给第三方匹配器，不使用这里的规范化结果替换它们。
func countPatterns(lines []string) int {
	count := 0
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		count++
	}
	return count
}

// Cache 按请求目录缓存祖先规则的合并结果，nil 也作为已查询结果保存。
// 缓存没有自动失效或文件监听，规则文件改变后既有目录结果不会主动重新加载。
type Cache struct {
	root string
	mu   sync.RWMutex
	data map[string]*IgnoreFile
}

// NewCache 保存停止向上遍历的根目录文本，不转绝对路径，也不预读规则。
func NewCache(rootDir string) *Cache {
	return &Cache{
		root: rootDir,
		data: make(map[string]*IgnoreFile),
	}
}

// Get 从 dir 向上收集祖先，到等于配置根目录或到达文件系统根时停止。
// 规则按祖先到子目录的顺序合并，读取失败的文件均跳过，本方法不返回读取错误。
// 未检查 dir 是否属于 c.root，调用方需保证路径形式与所需遍历边界一致。
func (c *Cache) Get(dir string) *IgnoreFile {
	c.mu.RLock()
	if cached, ok := c.data[dir]; ok {
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

	// 从读锁切换到写锁后再次检查，避免等待期间其他调用已经完成同目录加载。
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.data[dir]; ok {
		return cached
	}

	var ancestors []string
	for d := dir; ; d = filepath.Dir(d) {
		ancestors = append(ancestors, d)
		if d == c.root {
			break
		}

		parent := filepath.Dir(d)
		if parent == d {
			break
		}
	}

	var allLines []string
	// 倒序遍历祖先列表，让更接近请求目录的规则排在合并文本后部。
	for i := len(ancestors) - 1; i >= 0; i-- {
		ignorePath := filepath.Join(ancestors[i], IgnoreFilename)
		if data, err := os.ReadFile(ignorePath); err == nil {
			allLines = append(allLines, strings.Split(string(data), "\n")...)
		}
	}

	if len(allLines) == 0 {
		c.data[dir] = nil
		return nil
	}

	compiled := CompileLines(allLines)
	c.data[dir] = compiled
	return compiled
}

// CompileLines 直接编译给定规则行并统计条数，不访问文件系统。
func CompileLines(lines []string) *IgnoreFile {
	return &IgnoreFile{
		matcher:      gitignore.CompileIgnoreLines(lines...),
		patternCount: countPatterns(lines),
	}
}

// PatternCount 对 nil 返回零，允许调用方统一处理不存在的规则文件。
func (f *IgnoreFile) PatternCount() int {
	if f == nil {
		return 0
	}
	return f.patternCount
}

// Matches 对 nil 或无匹配器返回 false，目录路径缺少末尾斜线时先追加斜线再匹配。
// 其余路径文本原样交给匹配器，本方法不根据 root 转换为相对路径。
func (f *IgnoreFile) Matches(path string, isDir bool) bool {
	if f == nil || f.matcher == nil {
		return false
	}
	if isDir && !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return f.matcher.MatchesPath(path)
}
