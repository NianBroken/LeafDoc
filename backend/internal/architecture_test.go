// 本文件从非测试 Go 文件的导入声明检查分层依赖，不执行领域业务。
// 扫描不按构建标签筛选，因此平台专用生产文件也参与边界检查。
package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/NianBroken/LeafDoc/"

// allowedSliceImports 精确列出可被其他 Wiki 功能模块导入的共享包路径。
// pagesave 提供共用页面写入流程，清单不自动允许其任意子包。
var allowedSliceImports = map[string]bool{
	"internal/wiki/pagesave": true,
}

// TestArchitecture_ImportBoundaries_FollowLayerRules 枚举生产导入并逐边检查，
// 将违规结果排序后报告，使输出顺序不受映射遍历顺序影响。
func TestArchitecture_ImportBoundaries_FollowLayerRules(t *testing.T) {
	imports := collectProductionImports(t, ".")
	if len(imports) == 0 {
		t.Fatal("no packages found; test must run from the internal directory")
	}

	var violations []string
	for pkg, deps := range imports {
		for _, dep := range deps {
			if reason := violation(pkg, dep); reason != "" {
				edge := pkg + " -> " + dep
				violations = append(violations, edge+": "+reason)
			}
		}
	}

	sort.Strings(violations)
	for _, v := range violations {
		t.Error(v)
	}
}

// violation 返回首个适用层级规则的违规原因，空值表示导入被接受。
// 本模块路径已去除模块前缀，第三方依赖保持完整路径。
func violation(pkg, dep string) string {
	switch {
	case isUnder(pkg, "internal/core"):
		if strings.HasPrefix(dep, "github.com/gin-gonic/") {
			return "internal/core must not depend on gin"
		}
		if strings.HasPrefix(dep, "internal/") && !isUnder(dep, "internal/core") {
			return "internal/core may only import other internal/core packages"
		}
	case isUnder(pkg, "internal/http"):
		if isUnder(dep, "internal/wiki") {
			return "internal/http must not import internal/wiki"
		}
	case isUnder(pkg, "internal/wiki"):
		if pkg == "internal/wiki" || !isUnder(dep, "internal/wiki") || dep == "internal/wiki" {
			return ""
		}
		if sliceOf(pkg) != sliceOf(dep) && !allowedSliceImports[dep] {
			return "slices must not import other slices (wire them in internal/wiki/wiki.go)"
		}
	case strings.HasPrefix(pkg, "internal/"):
		if isUnder(dep, "internal/wiki") || isUnder(dep, "internal/http") {
			return "packages below internal/wiki must not import internal/wiki or internal/http"
		}
	}
	return ""
}

// isUnder 只接受根包或带斜线的后代，避免同名前缀包被误归类。
func isUnder(pkg, root string) bool {
	return pkg == root || strings.HasPrefix(pkg, root+"/")
}

// sliceOf 取前三段作为 Wiki 功能标识，更深子包归入同一功能模块。
func sliceOf(pkg string) string {
	parts := strings.SplitN(pkg, "/", 4)
	if len(parts) < 3 {
		return pkg
	}
	return strings.Join(parts[:3], "/")
}

// collectProductionImports 只解析非测试 Go 文件的导入，跳过 testdata、dist 和隐藏子目录。
// 目录转换为 internal 下的包路径，当前模块导入去掉模块前缀，其他导入保持原文。
// 无导入包也登记空项，遍历与解析失败直接使测试失败。
func collectProductionImports(t *testing.T, root string) map[string][]string {
	t.Helper()
	result := map[string][]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != root && (name == "testdata" || name == "dist" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			return err
		}
		pkg := "internal"
		if dir := filepath.ToSlash(filepath.Dir(path)); dir != "." {
			pkg += "/" + dir
		}
		for _, spec := range file.Imports {
			dep, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			result[pkg] = append(result[pkg], strings.TrimPrefix(dep, modulePath))
		}
		if _, ok := result[pkg]; !ok {
			result[pkg] = nil
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return result
}

// TestArchitecture_Violation_DetectsForbiddenImports 用允许和拒绝的具体导入组合，
// 验证核心层、HTTP 层、组装层、功能子包和共享 pagesave 的边界。
func TestArchitecture_Violation_DetectsForbiddenImports(t *testing.T) {
	cases := []struct {
		pkg, dep      string
		wantViolation bool
	}{
		{"internal/core/tree", "internal/core/markdown", false},
		{"internal/core/tree", "github.com/gin-gonic/gin", true},
		{"internal/core/tree", "internal/core/links", false},
		{"internal/core/tree", "internal/wiki/pages", true},
		{"internal/core/links", "internal/core/tree", false},
		{"internal/core/links", "internal/http/dto", true},
		{"internal/core/links", "internal/wiki", true},
		{"internal/http/middleware/auth", "internal/core/auth", false},
		{"internal/http", "internal/wiki/pages", true},
		{"internal/wiki/pages", "internal/wiki/pagesave", false},
		{"internal/wiki/pages", "internal/wiki/assets", true},
		{"internal/wiki/pages/sub", "internal/wiki/pages", false},
		{"internal/wiki", "internal/wiki/pages", false},
		{"internal/wiki/pages", "internal/wiki", false},
		{"internal/wiki/pages", "internal/core/tree", false},
	}
	for _, tc := range cases {
		got := violation(tc.pkg, tc.dep) != ""
		if got != tc.wantViolation {
			t.Errorf("violation(%q, %q) = %v, want %v", tc.pkg, tc.dep, got, tc.wantViolation)
		}
	}
}
