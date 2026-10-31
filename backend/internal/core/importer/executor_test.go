package importer

import (
	"errors"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// fakeExecWiki 记录执行器对页面和附件接口的调用，允许回调控制页面返回结果。
// 它不写入真实存储，公开附件路径和页面标识使用测试值。
type fakeExecWiki struct {
	hash string

	ensureCalls int
	updateCalls int

	ensureFn func(userID, targetPath, title string, kind *tree.NodeKind) (*tree.Page, error)
	updateFn func(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error)

	lastUpdatedContent *string
	ensureTargets      []string
	ensureKinds        []tree.NodeKind
	updateTitles       []string
	uploadCalls        int
	uploadedAssets     []string
	lastUploadMaxBytes int64
}

// TreeHash 返回测试维护的摘要，普通成功写入会追加变化标记。
func (f *fakeExecWiki) TreeHash() string { return f.hash }

// LookupPagePath 不属于执行器的调用流程，调用即触发 panic 以暴露意外依赖。
func (f *fakeExecWiki) LookupPagePath(path string) (*tree.PathLookup, error) {
	panic("not used by Executor")
}

// EnsurePath 保存调用顺序和节点类型，再交给自定义回调或返回最小页面。
// 默认分支直接读取 kind，测试调用须提供非空类型指针。
func (f *fakeExecWiki) EnsurePath(userID string, targetPath string, title string, kind *tree.NodeKind) (*tree.Page, error) {
	f.ensureCalls++
	f.ensureTargets = append(f.ensureTargets, targetPath)
	if kind != nil {
		f.ensureKinds = append(f.ensureKinds, *kind)
	}
	if f.ensureFn != nil {
		return f.ensureFn(userID, targetPath, title, kind)
	}
	return &tree.Page{PageNode: &tree.PageNode{ID: "p1", Title: title, Slug: "slug", Kind: *kind}}, nil
}

// UpdatePage 记录最后正文和标题序列，回调负责自定义行为时不执行默认摘要变化。
func (f *fakeExecWiki) UpdatePage(userID string, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
	f.updateCalls++
	f.lastUpdatedContent = content
	f.updateTitles = append(f.updateTitles, title)
	if f.updateFn != nil {
		return f.updateFn(userID, id, title, slug, content, kind)
	}
	// 摘要变化用于判断执行器是否记录写入之后的页面树状态。
	f.hash += "-changed"
	return &tree.Page{PageNode: &tree.PageNode{ID: id, Title: title, Slug: slug, Kind: *kind}}, nil
}

// UploadAsset 记录上传次数、文件名和传入上限，不读取或验证文件内容。
func (f *fakeExecWiki) UploadAsset(userID, pageID string, file multipart.File, filename string, maxBytes int64) (string, error) {
	f.uploadCalls++
	f.uploadedAssets = append(f.uploadedAssets, filename)
	f.lastUploadMaxBytes = maxBytes
	return "/assets/" + pageID + "/" + filename, nil
}

// writeTmp 按斜杠形式相对路径建立测试文件，所需父目录随文件一并创建。
func writeTmp(t *testing.T, dir, rel, content string) {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestExecutor_StalePlan 验证计划摘要与当前树摘要不一致时立即拒绝执行。
// 过期计划返回空结果，不进入页面建立或正文处理。
func TestExecutor_StalePlan(t *testing.T) {
	w := &fakeExecWiki{hash: "new"}
	plan := &PlanResult{TreeHash: "old"}
	opts := &PlanOptions{SourceBasePath: t.TempDir()}
	ex := NewExecutor(plan, opts, 0, w, slog.Default())

	got, err := ex.Execute("user1")
	if err == nil {
		t.Fatalf("expected stale plan error")
	}
	if got != nil {
		t.Fatalf("expected nil result on stale plan, got %#v", got)
	}
}

// TestExecutor_Create_HappyPath_PreservesNonInternalFrontmatter 验证创建流程保留自定义元信息。
// 通过重新解析写入正文检查集合、普通 title 和正文内容，并确认源系统 ID 没有复制。
func TestExecutor_Create_HappyPath_PreservesNonInternalFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "a.md", "---\naliases:\n  - x\ncustom_key: keep-me\nleafdoc_id: source-id\nleafdoc_title: Source Title\ntitle: X\n---\n\n# Heading\nBody")

	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "a.md", TargetPath: "docs/a", Title: "A", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())

	res, err := ex.Execute("user1")
	if err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	if res.ImportedCount != 1 || res.SkippedCount != 0 {
		t.Fatalf("counts imported=%d skipped=%d", res.ImportedCount, res.SkippedCount)
	}
	if len(res.Items) != 1 || res.Items[0].Action != ExecutionActionCreated {
		t.Fatalf("item result: %#v", res.Items)
	}
	if w.ensureCalls != 1 || w.updateCalls != 1 {
		t.Fatalf("calls ensure=%d update=%d", w.ensureCalls, w.updateCalls)
	}

	if w.lastUpdatedContent == nil {
		t.Fatalf("expected content to be passed to UpdatePage")
	}
	// 对实际传给 UpdatePage 的文本执行真实解析，不只检查原始字符串片段。
	fm, body, has, err := markdown.ParseFrontmatter(*w.lastUpdatedContent)
	if err != nil {
		t.Fatalf("ParseFrontmatter err: %v", err)
	}
	if !has {
		t.Fatalf("expected preserved frontmatter, got: %q", *w.lastUpdatedContent)
	}
	if body != "\n# Heading\nBody" {
		t.Fatalf("unexpected body: %q", body)
	}
	if got := fm.ExtraFields["custom_key"]; got != "keep-me" {
		t.Fatalf("expected custom_key to be preserved, got %#v", got)
	}
	if got := fm.ExtraFields["title"]; got != "X" {
		t.Fatalf("expected title extra field to be preserved, got %#v", got)
	}
	aliases, ok := fm.ExtraFields["aliases"].([]interface{})
	if !ok || len(aliases) != 1 || aliases[0] != "x" {
		t.Fatalf("expected aliases to be preserved, got %#v", fm.ExtraFields["aliases"])
	}
	if strings.Contains(*w.lastUpdatedContent, "leafdoc_id: source-id") {
		t.Fatalf("expected source leafdoc_id to be dropped, got: %q", *w.lastUpdatedContent)
	}

	if res.TreeHashBefore != "h1" {
		t.Fatalf("TreeHashBefore = %q", res.TreeHashBefore)
	}
	if res.TreeHash == "h1" {
		t.Fatalf("expected TreeHash to change (fake changes it), got %q", res.TreeHash)
	}
}

// TestExecutor_Create_HappyPath_PreservesDistinctExtraFieldValues
// 使用不同标量值和嵌套映射，验证导入重组不会把额外字段混淆或丢失。
func TestExecutor_Create_HappyPath_PreservesDistinctExtraFieldValues(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "a.md", "---\nalpha: first\nbeta: second\nnested:\n  key: value\n---\n\nBody")

	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "a.md", TargetPath: "docs/a", Title: "A", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	if w.lastUpdatedContent == nil {
		t.Fatalf("expected content to be passed to UpdatePage")
	}

	fm, _, has, err := markdown.ParseFrontmatter(*w.lastUpdatedContent)
	if err != nil {
		t.Fatalf("ParseFrontmatter err: %v", err)
	}
	if !has {
		t.Fatalf("expected frontmatter, got %q", *w.lastUpdatedContent)
	}
	if got := fm.ExtraFields["alpha"]; got != "first" {
		t.Fatalf("expected alpha=first, got %#v", got)
	}
	if got := fm.ExtraFields["beta"]; got != "second" {
		t.Fatalf("expected beta=second, got %#v", got)
	}
	nested, ok := fm.ExtraFields["nested"].(map[string]interface{})
	if !ok || nested["key"] != "value" {
		t.Fatalf("expected nested map to be preserved, got %#v", fm.ExtraFields["nested"])
	}
}

// TestExecutor_Skip_DoesNotCallWiki 验证跳过动作不调用页面建立和内容写入接口。
// 摘要读取仍属于正常执行流程，因此这里的断言针对两种写操作。
func TestExecutor_Skip_DoesNotCallWiki(t *testing.T) {
	tmp := t.TempDir()
	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "a.md", TargetPath: "docs/a", Action: PlanActionSkip},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	res, err := ex.Execute("user1")
	if err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	if res.SkippedCount != 1 || res.ImportedCount != 0 {
		t.Fatalf("counts imported=%d skipped=%d", res.ImportedCount, res.SkippedCount)
	}
	if w.ensureCalls != 0 || w.updateCalls != 0 {
		t.Fatalf("expected no wiki calls, got ensure=%d update=%d", w.ensureCalls, w.updateCalls)
	}
}

// TestExecutor_Create_EnsurePathError_SkipsItem 注入页面建立错误，确认单项失败被累计。
// Execute 仍返回结果而非整体错误，正文写入次数必须为零。
func TestExecutor_Create_EnsurePathError_SkipsItem(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "a.md", "Body")

	w := &fakeExecWiki{
		hash: "h1",
		ensureFn: func(userID, targetPath, title string, kind *tree.NodeKind) (*tree.Page, error) {
			return nil, errors.New("boom")
		},
	}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "a.md", TargetPath: "docs/a", Title: "A", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	res, err := ex.Execute("user1")
	if err != nil {
		t.Fatalf("Execute err: %v", err)
	}
	if res.SkippedCount != 1 || res.ImportedCount != 0 {
		t.Fatalf("counts imported=%d skipped=%d", res.ImportedCount, res.SkippedCount)
	}
	if res.Items[0].Error == nil || *res.Items[0].Error == "" {
		t.Fatalf("expected error message")
	}
	if w.updateCalls != 0 {
		t.Fatalf("UpdatePage should not be called")
	}
}

// TestExecutor_UnknownAction_SkipsItem 使用已声明但未由执行分支处理的 update 动作。
// 该项按跳过累计，并保留 unknown action 错误文字。
func TestExecutor_UnknownAction_SkipsItem(t *testing.T) {
	tmp := t.TempDir()
	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "a.md", TargetPath: "docs/a", Action: PlanActionUpdate}, // 执行器没有对应的更新分支。
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	res, err := ex.Execute("user1")
	if err != nil {
		t.Fatalf("Execute err: %v", err)
	}
	if res.SkippedCount != 1 {
		t.Fatalf("SkippedCount=%d", res.SkippedCount)
	}
	if res.Items[0].Error == nil || *res.Items[0].Error != "unknown action" {
		t.Fatalf("Error=%#v", res.Items[0].Error)
	}
}

// TestExecutor_Create_FolderIndexAndSiblingPage_ImportsSectionThenNestedPage
// 按计划先执行目录首页再执行普通页面，检查路径、类型与写入标题的顺序。
func TestExecutor_Create_FolderIndexAndSiblingPage_ImportsSectionThenNestedPage(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Ordner/index.md", `---
title: Ordner
---

# Ordner`)
	writeTmp(t, tmp, "Ordner/Ordner.md", "# Unterseite")

	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Ordner/index.md", TargetPath: "ordner", Title: "Ordner", Kind: tree.NodeKindSection, Action: PlanActionCreate},
			{SourcePath: "Ordner/Ordner.md", TargetPath: "ordner/ordner", Title: "Unterseite", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	res, err := ex.Execute("user1")
	if err != nil {
		t.Fatalf("Execute err: %v", err)
	}
	if res.ImportedCount != 2 || res.SkippedCount != 0 {
		t.Fatalf("counts imported=%d skipped=%d", res.ImportedCount, res.SkippedCount)
	}
	if w.ensureCalls != 2 || w.updateCalls != 2 {
		t.Fatalf("calls ensure=%d update=%d", w.ensureCalls, w.updateCalls)
	}
	if len(w.ensureTargets) != 2 || w.ensureTargets[0] != "ordner" || w.ensureTargets[1] != "ordner/ordner" {
		t.Fatalf("unexpected ensure targets: %#v", w.ensureTargets)
	}
	if len(w.ensureKinds) != 2 || w.ensureKinds[0] != tree.NodeKindSection || w.ensureKinds[1] != tree.NodeKindPage {
		t.Fatalf("unexpected ensure kinds: %#v", w.ensureKinds)
	}
	if len(w.updateTitles) != 2 || w.updateTitles[0] != "Ordner" || w.updateTitles[1] != "Unterseite" {
		t.Fatalf("unexpected update titles: %#v", w.updateTitles)
	}
}

// TestExecutor_Create_RewritesMarkdownAndWikiLinksToImportedPages
// 验证相对文件路径、根路径、无扩展名路径和目录首页均映射到计划中的页面。
// 双中括号链接保留语法与别名，只把路径提示转换为目标页面路由。
func TestExecutor_Create_RewritesMarkdownAndWikiLinksToImportedPages(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Guides/index.md", "# Guides")
	writeTmp(t, tmp, "Guides/Setup.md", strings.Join([]string{
		"# Setup",
		"",
		"[Relative](../Reference/Endpoints.md)",
		"[Absolute](/Guides/index.md)",
		"[RouteStyle](/Reference/Endpoints)",
		"[Container](/Guides/)",
		"[[Reference/Endpoints|API Alias]]",
	}, "\n"))
	writeTmp(t, tmp, "Reference/Endpoints.md", "# Endpoints")

	w := &fakeExecWiki{hash: "h1"}
	updatedContentByTitle := map[string]string{}
	w.updateFn = func(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
		w.lastUpdatedContent = content
		w.updateTitles = append(w.updateTitles, title)
		if content != nil {
			updatedContentByTitle[title] = *content
		}
		w.hash += "-changed"
		return &tree.Page{PageNode: &tree.PageNode{ID: id, Title: title, Slug: slug, Kind: *kind}}, nil
	}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Guides/index.md", TargetPath: "guides", Title: "Guides", Kind: tree.NodeKindSection, Action: PlanActionCreate},
			{SourcePath: "Guides/Setup.md", TargetPath: "guides/setup", Title: "Setup", Kind: tree.NodeKindPage, Action: PlanActionCreate},
			{SourcePath: "Reference/Endpoints.md", TargetPath: "reference/endpoints", Title: "Endpoints", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	setupContent, ok := updatedContentByTitle["Setup"]
	if !ok {
		t.Fatalf("expected setup content to be updated")
	}

	for _, expected := range []string{
		"[Relative](/reference/endpoints)",
		"[Absolute](/guides)",
		"[RouteStyle](/reference/endpoints)",
		"[Container](/guides)",
		"[[reference/endpoints|API Alias]]",
	} {
		if !strings.Contains(setupContent, expected) {
			t.Fatalf("expected rewritten content to contain %q, got:\n%s", expected, setupContent)
		}
	}
}

// TestExecutor_Create_UploadsRelativeAndRootAssets 验证两种附件定位方式及同页缓存。
// 三处引用只有两个不同文件，应上传两次，并将配置的字节上限传给上传接口。
func TestExecutor_Create_UploadsRelativeAndRootAssets(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Guides/Setup.md", strings.Join([]string{
		"# Setup",
		"",
		"![Relative](./images/logo.png)",
		"[Asset](/shared/manual.pdf)",
		"![[./images/logo.png]]",
	}, "\n"))
	writeTmp(t, tmp, "Guides/images/logo.png", "png-bytes")
	writeTmp(t, tmp, "shared/manual.pdf", "pdf-bytes")

	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Guides/Setup.md", TargetPath: "guides/setup", Title: "Setup", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 1234, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	if w.uploadCalls != 2 {
		t.Fatalf("expected 2 asset uploads, got %d", w.uploadCalls)
	}
	if w.lastUpdatedContent == nil {
		t.Fatalf("expected content to be updated")
	}
	if !strings.Contains(*w.lastUpdatedContent, "![Relative](/assets/p1/logo.png)") {
		t.Fatalf("expected relative asset link rewrite, got:\n%s", *w.lastUpdatedContent)
	}
	if !strings.Contains(*w.lastUpdatedContent, "[Asset](/assets/p1/manual.pdf)") {
		t.Fatalf("expected root asset link rewrite, got:\n%s", *w.lastUpdatedContent)
	}
	if !strings.Contains(*w.lastUpdatedContent, "![logo.png](/assets/p1/logo.png)") {
		t.Fatalf("expected wiki asset link rewrite, got:\n%s", *w.lastUpdatedContent)
	}
	if w.lastUploadMaxBytes != 1234 {
		t.Fatalf("expected asset uploads to use configured max bytes, got %d", w.lastUploadMaxBytes)
	}
}

// TestExecutor_Create_WikiLinkToNonImageAssetStaysNormalLink
// 对同一个 PDF，普通双中括号引用输出链接，显式嵌入标记仍输出带感叹号的语法。
func TestExecutor_Create_WikiLinkToNonImageAssetStaysNormalLink(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Guides/Setup.md", strings.Join([]string{
		"# Setup",
		"",
		"[[../shared/manual.pdf]]",
		"![[../shared/manual.pdf]]",
	}, "\n"))
	writeTmp(t, tmp, "shared/manual.pdf", "pdf-bytes")

	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Guides/Setup.md", TargetPath: "guides/setup", Title: "Setup", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	if w.lastUpdatedContent == nil {
		t.Fatalf("expected content to be updated")
	}
	if !strings.Contains(*w.lastUpdatedContent, "[manual.pdf](/assets/p1/manual.pdf)") {
		t.Fatalf("expected non-embed wiki asset to stay a normal link, got:\n%s", *w.lastUpdatedContent)
	}
	if !strings.Contains(*w.lastUpdatedContent, "![manual.pdf](/assets/p1/manual.pdf)") {
		t.Fatalf("expected embed wiki asset to use embed syntax, got:\n%s", *w.lastUpdatedContent)
	}
}

// TestExecutor_Create_WikiLinkFallsBackToUniqueNestedBasenameOnly
// 同时放入唯一名称与重复名称候选，确认不带路径的双中括号文字保留原格式。
// 本断言检查最终正文，不单独断言内部名称查找是否命中。
func TestExecutor_Create_WikiLinkFallsBackToUniqueNestedBasenameOnly(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Home.md", strings.Join([]string{
		"# Home",
		"",
		"[[Brainstorm]]",
		"[[Meeting Notes]]",
	}, "\n"))
	writeTmp(t, tmp, "Daily/Brainstorm.md", "# Brainstorm")
	writeTmp(t, tmp, "Daily/Meeting Notes.md", "# Daily Meeting Notes")
	writeTmp(t, tmp, "Archive/Meeting Notes.md", "# Archived Meeting Notes")

	w := &fakeExecWiki{hash: "h1"}
	updatedContentByTitle := map[string]string{}
	w.updateFn = func(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
		w.lastUpdatedContent = content
		if content != nil {
			updatedContentByTitle[title] = *content
		}
		w.hash += "-changed"
		return &tree.Page{PageNode: &tree.PageNode{ID: id, Title: title, Slug: slug, Kind: *kind}}, nil
	}

	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Home.md", TargetPath: "home", Title: "Home", Kind: tree.NodeKindPage, Action: PlanActionCreate},
			{SourcePath: "Daily/Brainstorm.md", TargetPath: "daily/brainstorm", Title: "Brainstorm", Kind: tree.NodeKindPage, Action: PlanActionCreate},
			{SourcePath: "Daily/Meeting Notes.md", TargetPath: "daily/meeting-notes", Title: "Meeting Notes", Kind: tree.NodeKindPage, Action: PlanActionCreate},
			{SourcePath: "Archive/Meeting Notes.md", TargetPath: "archive/meeting-notes", Title: "Meeting Notes", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	homeContent := updatedContentByTitle["Home"]
	if !strings.Contains(homeContent, "[[Brainstorm]]") {
		t.Fatalf("expected unique basename wiki link to stay as wikilink, got:\n%s", homeContent)
	}
	if !strings.Contains(homeContent, "[[Meeting Notes]]") {
		t.Fatalf("expected ambiguous basename wiki link to stay unchanged, got:\n%s", homeContent)
	}
}

// TestExecutor_Create_WikiLinkResolvesUniqueNestedPathSuffix
// 链接省略导入包顶级目录时，唯一后缀匹配必须还原完整目标路径并保留别名。
func TestExecutor_Create_WikiLinkResolvesUniqueNestedPathSuffix(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "knowledge-main/tools/kubernetes/resources/StatefulSet.md", strings.Join([]string{
		"# StatefulSet",
		"",
		"[[Tools/Kubernetes/Resources/Deployment|Deployment]]",
	}, "\n"))
	writeTmp(t, tmp, "knowledge-main/tools/kubernetes/resources/Deployment.md", "# Deployment")

	w := &fakeExecWiki{hash: "h1"}
	updatedContentByTitle := map[string]string{}
	w.updateFn = func(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
		w.lastUpdatedContent = content
		if content != nil {
			updatedContentByTitle[title] = *content
		}
		w.hash += "-changed"
		return &tree.Page{PageNode: &tree.PageNode{ID: id, Title: title, Slug: slug, Kind: *kind}}, nil
	}

	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "knowledge-main/tools/kubernetes/resources/StatefulSet.md", TargetPath: "knowledge-main/tools/kubernetes/resources/statefulset", Title: "StatefulSet", Kind: tree.NodeKindPage, Action: PlanActionCreate},
			{SourcePath: "knowledge-main/tools/kubernetes/resources/Deployment.md", TargetPath: "knowledge-main/tools/kubernetes/resources/deployment", Title: "Deployment", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	statefulSetContent := updatedContentByTitle["StatefulSet"]
	if !strings.Contains(statefulSetContent, "[[knowledge-main/tools/kubernetes/resources/deployment|Deployment]]") {
		t.Fatalf("expected unique nested path suffix wiki link to stay as wikilink, got:\n%s", statefulSetContent)
	}
}

// TestExecutor_Create_UnresolvedWikiLinkStaysAsWikiLink
// 对计划中不存在且没有路径提示的页面名称，导入后仍保留双中括号引用。
func TestExecutor_Create_UnresolvedWikiLinkStaysAsWikiLink(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Home.md", strings.Join([]string{
		"# Home",
		"",
		"[[Missing Note]]",
	}, "\n"))

	w := &fakeExecWiki{hash: "h1"}
	updatedContentByTitle := map[string]string{}
	w.updateFn = func(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
		w.lastUpdatedContent = content
		if content != nil {
			updatedContentByTitle[title] = *content
		}
		w.hash += "-changed"
		return &tree.Page{PageNode: &tree.PageNode{ID: id, Title: title, Slug: slug, Kind: *kind}}, nil
	}

	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Home.md", TargetPath: "home", Title: "Home", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	homeContent := updatedContentByTitle["Home"]
	if !strings.Contains(homeContent, "[[Missing Note]]") {
		t.Fatalf("expected unresolved wiki link to stay as wikilink, got:\n%s", homeContent)
	}
}

// TestExecutor_Create_DoesNotRewriteLinksInsideCode 对相同目标同时设置代码内与代码外引用。
// 代码内示例保持源路径，普通正文链接必须转换，用以确认分段处理确实生效。
func TestExecutor_Create_DoesNotRewriteLinksInsideCode(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Guides/Setup.md", strings.Join([]string{
		"# Setup",
		"",
		"`[Inline](../Reference/Endpoints.md)`",
		"",
		"```md",
		"[Fence](../Reference/Endpoints.md)",
		"[[Reference/Endpoints|Fence Alias]]",
		"```",
		"",
		"[Real](../Reference/Endpoints.md)",
		"[[Reference/Endpoints|Real Alias]]",
	}, "\n"))
	writeTmp(t, tmp, "Reference/Endpoints.md", "# Endpoints")

	w := &fakeExecWiki{hash: "h1"}
	updatedContentByTitle := map[string]string{}
	w.updateFn = func(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
		w.lastUpdatedContent = content
		if content != nil {
			updatedContentByTitle[title] = *content
		}
		w.hash += "-changed"
		return &tree.Page{PageNode: &tree.PageNode{ID: id, Title: title, Slug: slug, Kind: *kind}}, nil
	}

	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Guides/Setup.md", TargetPath: "guides/setup", Title: "Setup", Kind: tree.NodeKindPage, Action: PlanActionCreate},
			{SourcePath: "Reference/Endpoints.md", TargetPath: "reference/endpoints", Title: "Endpoints", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	setupContent := updatedContentByTitle["Setup"]
	for _, expected := range []string{
		"`[Inline](../Reference/Endpoints.md)`",
		"[Fence](../Reference/Endpoints.md)",
		"[[Reference/Endpoints|Fence Alias]]",
		"[Real](/reference/endpoints)",
		"[[reference/endpoints|Real Alias]]",
	} {
		if !strings.Contains(setupContent, expected) {
			t.Fatalf("expected content to contain %q, got:\n%s", expected, setupContent)
		}
	}
}

// TestExecutor_Create_RewritesWindowsStyleMarkdownAndAssetPaths
// 使用反斜杠相对地址验证页面和附件都能经过地址规范处理，不要求 Windows 磁盘布局。
func TestExecutor_Create_RewritesWindowsStyleMarkdownAndAssetPaths(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Guides/Setup.md", strings.Join([]string{
		"# Setup",
		"",
		"[Doc](..\\Reference\\Endpoints.md)",
		"![Diagram](images\\diagram.png)",
	}, "\n"))
	writeTmp(t, tmp, "Reference/Endpoints.md", "# Endpoints")
	writeTmp(t, tmp, "Guides/images/diagram.png", "png-bytes")

	w := &fakeExecWiki{hash: "h1"}
	updatedContentByTitle := map[string]string{}
	w.updateFn = func(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
		w.lastUpdatedContent = content
		if content != nil {
			updatedContentByTitle[title] = *content
		}
		w.hash += "-changed"
		return &tree.Page{PageNode: &tree.PageNode{ID: id, Title: title, Slug: slug, Kind: *kind}}, nil
	}

	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Guides/Setup.md", TargetPath: "guides/setup", Title: "Setup", Kind: tree.NodeKindPage, Action: PlanActionCreate},
			{SourcePath: "Reference/Endpoints.md", TargetPath: "reference/endpoints", Title: "Endpoints", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	setupContent := updatedContentByTitle["Setup"]
	for _, expected := range []string{
		"[Doc](/reference/endpoints)",
		"![Diagram](/assets/p1/diagram.png)",
	} {
		if !strings.Contains(setupContent, expected) {
			t.Fatalf("expected content to contain %q, got:\n%s", expected, setupContent)
		}
	}
}

// TestExecutor_Create_LeavesWindowsDriveLetterPathsUntouched
// 验证带盘符的示例绝对路径不被当作普通导入相对地址改写，正文保留原文字。
func TestExecutor_Create_LeavesWindowsDriveLetterPathsUntouched(t *testing.T) {
	tmp := t.TempDir()
	writeTmp(t, tmp, "Guides/Setup.md", strings.Join([]string{
		"# Setup",
		"",
		"[Windows File](C:\\Users\\John\\Notes\\Endpoints.md)",
		"![Windows Image](C:\\Users\\John\\Images\\diagram.png)",
	}, "\n"))

	w := &fakeExecWiki{hash: "h1"}
	plan := &PlanResult{
		TreeHash: "h1",
		Items: []PlanItem{
			{SourcePath: "Guides/Setup.md", TargetPath: "guides/setup", Title: "Setup", Kind: tree.NodeKindPage, Action: PlanActionCreate},
		},
	}
	opts := &PlanOptions{SourceBasePath: tmp}

	ex := NewExecutor(plan, opts, 0, w, slog.Default())
	if _, err := ex.Execute("user1"); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	if w.lastUpdatedContent == nil {
		t.Fatalf("expected updated content")
	}
	for _, expected := range []string{
		"[Windows File](C:\\Users\\John\\Notes\\Endpoints.md)",
		"![Windows Image](C:\\Users\\John\\Images\\diagram.png)",
	} {
		if !strings.Contains(*w.lastUpdatedContent, expected) {
			t.Fatalf("expected content to keep %q untouched, got:\n%s", expected, *w.lastUpdatedContent)
		}
	}
}
