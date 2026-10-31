package importer

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/testutil"

	"github.com/NianBroken/LeafDoc/internal/core/ignore"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// fakeWiki 为规划和执行测试提供可控制的路径查询、页面建立及写入结果。
// 调用计数与最后写入正文用于断言调用行为，不连接数据库或真实页面树。
type fakeWiki struct {
	treeHash string

	// 规划查询优先返回指定错误，其次按目标路径读取预设查询结果。
	lookups   map[string]*tree.PathLookup
	lookupErr error

	// 执行调用记录与故障开关，用于验证失败分支是否停止后续写入。
	ensureCalls        int
	updateCalls        int
	lastUpdatedContent *string

	ensureFn      func(userID, targetPath, title string, kind *tree.NodeKind) (*tree.Page, error)
	ensureErr     error
	ensureNilPage bool
	updateErr     error
}

// TreeHash 返回测试指定的树摘要，成功写入时由 UpdatePage 追加标记。
func (f *fakeWiki) TreeHash() string { return f.treeHash }

// LookupPagePath 对未预设的路径返回不存在结果，不自动推导父级节点。
func (f *fakeWiki) LookupPagePath(p string) (*tree.PathLookup, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if v, ok := f.lookups[p]; ok {
		return v, nil
	}
	return &tree.PathLookup{Path: p, Exists: false, Segments: []tree.PathSegment{}}, nil
}

// EnsurePath 先记录调用，再按回调、错误、空页面开关的顺序选择返回路径。
// 普通成功路径只建立最小页面对象，不实际维护目标路径对应的树结构。
func (f *fakeWiki) EnsurePath(userID string, targetPath string, title string, kind *tree.NodeKind) (*tree.Page, error) {
	f.ensureCalls++
	if f.ensureFn != nil {
		return f.ensureFn(userID, targetPath, title, kind)
	}
	if f.ensureErr != nil {
		return nil, f.ensureErr
	}
	if f.ensureNilPage {
		return nil, nil
	}
	k := tree.NodeKindPage
	if kind != nil {
		k = *kind
	}
	// 仅填充执行测试访问的字段，默认路径段固定为测试值。
	return &tree.Page{PageNode: &tree.PageNode{
		ID:    "p1",
		Title: title,
		Slug:  "slug",
		Kind:  k,
	}}, nil
}

// UpdatePage 记录正文指针与调用数，成功时更改树摘要并返回传入的页面元信息。
func (f *fakeWiki) UpdatePage(userID string, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
	f.updateCalls++
	f.lastUpdatedContent = content
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	// 用摘要后缀表示写入已发生，供执行结果的前后状态断言使用。
	f.treeHash += "-changed"
	k := tree.NodeKindPage
	if kind != nil {
		k = *kind
	}
	return &tree.Page{PageNode: &tree.PageNode{
		ID:    id,
		Title: title,
		Slug:  slug,
		Kind:  k,
	}}, nil
}

// UploadAsset 只构造公开路径，不读取文件流，也不实施大小限制。
func (f *fakeWiki) UploadAsset(userID, pageID string, file multipart.File, filename string, maxBytes int64) (string, error) {
	return "/assets/" + pageID + "/" + filename, nil
}

// newPlannerWithFake 让测试共享实际路径名称生成器，存储根目录留空。
func newPlannerWithFake(w *fakeWiki) *Planner {
	return NewPlanner(w, tree.NewSlugService(), "")
}

// TestPlanner_CreatePlan_CreateNewPage_NonIndex 验证普通文件映射为创建页面动作。
// 标题来自正文标题，目标基础路径与规范化文件名组合，树摘要来自 Wiki 替身。
func TestPlanner_CreatePlan_CreateNewPage_NonIndex(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "My Page.md", "# Hello\n\nbody")

	wiki := &fakeWiki{
		treeHash: "h1",
		lookups:  map[string]*tree.PathLookup{},
	}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "My Page.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "/docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if res.TreeHash != "h1" {
		t.Fatalf("TreeHash = %q", res.TreeHash)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}
	if len(res.Items) != 1 {
		t.Fatalf("Items len = %d", len(res.Items))
	}

	it := res.Items[0]
	if it.Action != PlanActionCreate {
		t.Fatalf("Action = %q", it.Action)
	}
	if it.Kind != tree.NodeKindPage {
		t.Fatalf("Kind = %v", it.Kind)
	}
	if it.Title != "Hello" {
		t.Fatalf("Title = %q", it.Title)
	}
	if it.TargetPath != "docs/my-page" {
		t.Fatalf("TargetPath = %q (want docs/my-page)", it.TargetPath)
	}
	if it.DesiredSlug != "my-page" {
		t.Fatalf("DesiredSlug = %q (want my-page)", it.DesiredSlug)
	}
}

// TestPlanner_CreatePlan_CreateNewSection_IndexMd 验证目录首页映射为分区而非子页面。
// YAML 标题优先于正文标题，目标路径不保留 index 文件名。
func TestPlanner_CreatePlan_CreateNewSection_IndexMd(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "Guides/index.md", "---\ntitle: Guides\n---\n\n# Ignored")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "Guides/index.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	it := res.Items[0]

	if it.Kind != tree.NodeKindSection {
		t.Fatalf("Kind = %v", it.Kind)
	}
	if it.Action != PlanActionCreate {
		t.Fatalf("Action = %q", it.Action)
	}
	if it.TargetPath != "docs/guides" {
		t.Fatalf("TargetPath = %q (want docs/guides)", it.TargetPath)
	}
	if it.DesiredSlug != "guides" {
		t.Fatalf("DesiredSlug = %q (want guides)", it.DesiredSlug)
	}
	if it.Title != "Guides" {
		t.Fatalf("Title = %q", it.Title)
	}
}

// TestPlanner_CreatePlan_PrefersLeafDocTitleOverTitle 同时提供两个标题字段与正文标题。
// 计划项必须采用 leafdoc_title，确认导入标题沿用实际 Markdown 标题提取优先级。
func TestPlanner_CreatePlan_PrefersLeafDocTitleOverTitle(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "Guide.md", "---\nleafdoc_title: Preferred Title\ntitle: Fallback Title\n---\n\n# Heading")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "Guide.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("Items len = %d", len(res.Items))
	}

	if got := res.Items[0].Title; got != "Preferred Title" {
		t.Fatalf("Title = %q (want Preferred Title)", got)
	}
}

// TestPlanner_CreatePlan_TitleFallbackPriority_WindowsPathFilenameFallback
// 直接测试 Markdown 对象的标题提取，覆盖元信息、首个标题与 Windows 文件名回退。
// 此测试没有调用规划器，也不要求给出的 Windows 示例路径真实存在。
func TestPlanner_CreatePlan_TitleFallbackPriority_WindowsPathFilenameFallback(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "frontmatter wins",
			content: "---\nleafdoc_title: Frontmatter Title\n---\n\n# Heading Title",
			want:    "Frontmatter Title",
		},
		{
			name:    "first heading wins when frontmatter missing",
			content: "Intro text\n\n# Heading Title\nBody",
			want:    "Heading Title",
		},
		{
			name:    "filename fallback strips windows path",
			content: "Body without title markers",
			want:    "1999-07-23 - Memo to Staff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdFile, err := markdown.NewMarkdownFileFromRaw(`C:\Users\johnjkr\AppData\Local\Temp\import-1280817455\1999-07-23 - Memo to Staff.md`, tt.content)
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			title, err := mdFile.GetTitle()
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if title != tt.want {
				t.Fatalf("title = %q (want %q)", title, tt.want)
			}
		})
	}
}

// TestPlanner_CreatePlan_SkipExisting_UsesLookupLastSegment 验证已存在目标默认跳过。
// 页面标识和期望路径段取查询末段，不使用源文件推导值覆盖现有节点身份。
func TestPlanner_CreatePlan_SkipExisting_UsesLookupLastSegment(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "a.md", "# A")

	existingID := "id123"
	existingKind := tree.NodeKindPage
	existingTitle := "Existing A"

	wiki := &fakeWiki{
		treeHash: "h",
		lookups: map[string]*tree.PathLookup{
			"docs/a": {
				Path:   "docs/a",
				Exists: true,
				Segments: []tree.PathSegment{
					{Slug: "docs", Exists: true},
					{Slug: "a", Exists: true, ID: &existingID, Kind: &existingKind, Title: &existingTitle},
				},
			},
		},
	}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "a.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}

	it := res.Items[0]
	if it.Action != PlanActionSkip {
		t.Fatalf("Action = %q", it.Action)
	}
	if !it.Exists {
		t.Fatalf("Exists = false")
	}
	if it.ExistingID == nil || *it.ExistingID != existingID {
		t.Fatalf("ExistingID = %#v (want %q)", it.ExistingID, existingID)
	}
	if it.DesiredSlug != "a" {
		t.Fatalf("DesiredSlug = %q (want a)", it.DesiredSlug)
	}
}

// TestPlanner_CreatePlan_Error_SourceMissing_IsCollected 确认源文件缺失成为单项错误。
// 整份计划仍成功返回，失败来源不进入可执行 Items。
func TestPlanner_CreatePlan_Error_SourceMissing_IsCollected(t *testing.T) {
	tmp := t.TempDir()
	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "missing.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("Items len = %d (want 0)", len(res.Items))
	}
	if len(res.Errors) != 1 {
		t.Fatalf("Errors len = %d (want 1)", len(res.Errors))
	}
}

// TestPlanner_CreatePlan_Error_SourceIsDirectory_IsCollected 验证目录不能作为文件条目导入。
// 该失败记录到 Errors，不产生计划项，也不使 CreatePlan 返回整体错误。
func TestPlanner_CreatePlan_Error_SourceIsDirectory_IsCollected(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "dir"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("Items len = %d (want 0)", len(res.Items))
	}
	if len(res.Errors) != 1 {
		t.Fatalf("Errors len = %d (want 1)", len(res.Errors))
	}
}

// TestPlanner_CreatePlan_Error_ExistingZeroSegments_IsCollected
// 构造存在目标却没有路径段的查询结果，验证规划器拒绝无法读取末段的状态。
func TestPlanner_CreatePlan_Error_ExistingZeroSegments_IsCollected(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "x.md", "# X")

	wiki := &fakeWiki{
		treeHash: "h",
		lookups: map[string]*tree.PathLookup{
			"docs/x": {Path: "docs/x", Exists: true, Segments: []tree.PathSegment{}},
		},
	}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "x.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("Items len = %d (want 0)", len(res.Items))
	}
	if len(res.Errors) != 1 {
		t.Fatalf("Errors len = %d (want 1)", len(res.Errors))
	}
}

// TestPlanner_CreatePlan_TitleExtractionError_AddsNote 用加载替身注入标题读取错误。
// 文件存在性仍通过真实磁盘检查，错误只进入备注，标题回退到目标路径末段。
func TestPlanner_CreatePlan_TitleExtractionError_AddsNote(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "unreadable.md", "# Title")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)
	p.loadMarkdown = func(string) (*markdown.MarkdownFile, error) { return nil, fmt.Errorf("injected read failure") }

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "unreadable.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}
	if len(res.Items) != 1 {
		t.Fatalf("Items len = %d (want 1)", len(res.Items))
	}

	it := res.Items[0]
	if len(it.Notes) != 1 {
		t.Fatalf("Notes len = %d (want 1)", len(it.Notes))
	}
	if !strings.Contains(it.Notes[0], "Failed to load markdown file for title extraction") {
		t.Fatalf("Note = %q (should contain 'Failed to load markdown file for title extraction')", it.Notes[0])
	}
	// 加载对象为空时采用已推导目标路径的末段，不读取正文标题。
	if it.Title != "unreadable" {
		t.Fatalf("Title = %q (want unreadable)", it.Title)
	}
}

// TestPlanner_analyzeEntry_NormalizesSourceDirSegments 验证目录名称也经过统一路径段转换。
func TestPlanner_analyzeEntry_NormalizesSourceDirSegments(t *testing.T) {
	// 含空格的源目录变为 my-guides，再与目标基础路径和文件名路径段组合。
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "My Guides/Intro.md", "# Intro")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "My Guides/Intro.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}
	if res.Items[0].TargetPath != "docs/my-guides/intro" {
		t.Fatalf("TargetPath = %q (want docs/my-guides/intro)", res.Items[0].TargetPath)
	}
}

// TestPlanner_analyzeEntry_ReservedSlugSegmentsUseCentralizedSafeNormalization
// 以保留名称 API 验证路径生成器的避让结果同时写入目标路径与期望路径段。
func TestPlanner_analyzeEntry_ReservedSlugSegmentsUseCentralizedSafeNormalization(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "Reference/API.md", "# API")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "Reference/API.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}
	if got := res.Items[0].TargetPath; got != "docs/reference/api-1" {
		t.Fatalf("TargetPath = %q (want docs/reference/api-1)", got)
	}
	if got := res.Items[0].DesiredSlug; got != "api-1" {
		t.Fatalf("DesiredSlug = %q (want api-1)", got)
	}
}

// TestPlanner_analyzeEntry_InvalidSourceDirSegment_ReturnsError
// 验证纯标点目录无法形成有效路径段时，该文件进入错误列表。
func TestPlanner_analyzeEntry_InvalidSourceDirSegment_ReturnsError(t *testing.T) {
	// 目录 !!! 去除不支持的字符后为空，不能形成可执行页面路径。
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "!!!/a.md", "# A")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "!!!/a.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("Items len = %d (want 0)", len(res.Items))
	}
	if len(res.Errors) != 1 {
		t.Fatalf("Errors len = %d (want 1)", len(res.Errors))
	}
	// 错误文字必须非空，此处不约束具体错误类型或完整消息。
	if res.Errors[0] == "" {
		t.Fatalf("unexpected error: %v", res.Errors[0])
	}
}

// TestPlanner_CreatePlan_RootIndexMd_EmptyWikiPath_UsesFallbackTitle
// 验证根目录首页在标题加载失败时使用 index 作为回退标题，并保留失败备注。
func TestPlanner_CreatePlan_RootIndexMd_EmptyWikiPath_UsesFallbackTitle(t *testing.T) {
	// 目标基础路径为空，因此根目录 index.md 对应空路由和分区节点。
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "index.md", "# Title")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)
	p.loadMarkdown = func(string) (*markdown.MarkdownFile, error) { return nil, fmt.Errorf("injected read failure") }

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "index.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "", // 空基础路径使该首页直接映射到根位置。
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}
	if len(res.Items) != 1 {
		t.Fatalf("Items len = %d (want 1)", len(res.Items))
	}

	it := res.Items[0]
	if it.TargetPath != "" {
		t.Fatalf("TargetPath = %q (want empty)", it.TargetPath)
	}
	if it.Kind != tree.NodeKindSection {
		t.Fatalf("Kind = %v (want Section)", it.Kind)
	}
	// 空路由的标题回退使用源文件名，不使用 path.Base 返回的点号。
	if it.Title != "index" {
		t.Fatalf("Title = %q (want index as fallback when wikiPath is empty and markdown fails)", it.Title)
	}
	// 标题加载失败属于可保留的备注，计划仍包含这个分区。
	if len(it.Notes) == 0 {
		t.Fatalf("Expected notes about failed markdown loading")
	}
	if !strings.Contains(it.Notes[0], "Failed to load markdown file for title extraction") {
		t.Fatalf("Note = %q (should contain 'Failed to load markdown file for title extraction')", it.Notes[0])
	}
}

// TestPlanner_CreatePlan_FolderIndexAndSiblingPage_MapToSectionAndNestedPage
// 验证同目录首页与普通文件分别映射为分区及分区内页面，名称相同也不合并目标。
func TestPlanner_CreatePlan_FolderIndexAndSiblingPage_MapToSectionAndNestedPage(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "Ordner/index.md", `---
title: Ordner
---

# Ordner`)
	testutil.WriteFile(t, tmp, "Ordner/Ordner.md", "# Unterseite")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{
		{SourcePath: "Ordner/index.md"},
		{SourcePath: "Ordner/Ordner.md"},
	}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}
	if len(res.Items) != 2 {
		t.Fatalf("Items len = %d (want 2)", len(res.Items))
	}

	section := res.Items[0]
	page := res.Items[1]

	if section.SourcePath != "Ordner/index.md" || section.Kind != tree.NodeKindSection || section.TargetPath != "ordner" {
		t.Fatalf("unexpected section item: %#v", section)
	}
	if page.SourcePath != "Ordner/Ordner.md" || page.Kind != tree.NodeKindPage || page.TargetPath != "ordner/ordner" {
		t.Fatalf("unexpected nested page item: %#v", page)
	}
}

// TestPlanner_CreatePlan_FolderMarkdownWithoutIndex_RemainsNestedPage
// 验证缺少目录首页时，普通文件仍保留父目录层级，不自动提升为分区首页。
func TestPlanner_CreatePlan_FolderMarkdownWithoutIndex_RemainsNestedPage(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "Ordner/Ordner.md", "# Unterseite")

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "Ordner/Ordner.md"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "wiki",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %#v", res.Errors)
	}
	if len(res.Items) != 1 {
		t.Fatalf("Items len = %d (want 1)", len(res.Items))
	}

	it := res.Items[0]
	if it.Kind != tree.NodeKindPage {
		t.Fatalf("Kind = %v (want Page)", it.Kind)
	}
	if it.TargetPath != "wiki/ordner/ordner" {
		t.Fatalf("TargetPath = %q (want wiki/ordner/ordner)", it.TargetPath)
	}
}

// TestPlanner_CreatePlan_CreateNewSection_IndexUppercaseMD
// 验证首页识别不区分扩展名大小写，index.MD 与小写形式具有相同分区映射。
func TestPlanner_CreatePlan_CreateNewSection_IndexUppercaseMD(t *testing.T) {
	tmp := t.TempDir()
	testutil.WriteFile(t, tmp, "Guides/index.MD", `---
title: Guides
---

# Ignored`)

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := newPlannerWithFake(wiki)

	res, err := p.CreatePlan([]ImportMDFile{{SourcePath: "Guides/index.MD"}}, PlanOptions{
		SourceBasePath: tmp,
		TargetBasePath: "docs",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}
	it := res.Items[0]
	if it.Kind != tree.NodeKindSection {
		t.Fatalf("Kind = %v", it.Kind)
	}
	if it.TargetPath != "docs/guides" {
		t.Fatalf("TargetPath = %q (want docs/guides)", it.TargetPath)
	}
}

// TestPlanner_CreatePlan_SkipsIgnoredPaths 使用真实忽略缓存过滤 drafts 目录中的目标。
// 断言只保留未命中的 readme 页面，不把被忽略文件当作计划错误。
func TestPlanner_CreatePlan_SkipsIgnoredPaths(t *testing.T) {
	tmp := t.TempDir()
	// 测试存储根目录包含 root 子目录，与规划器查询忽略缓存的目录结构一致。
	if err := os.MkdirAll(filepath.Join(tmp, "root", "drafts"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	testutil.WriteFile(t, tmp, "root/readme.md", "# Readme")
	testutil.WriteFile(t, tmp, "root/drafts/note.md", "# Draft Note")

	// 根目录规则过滤 drafts 子目录，缓存以同一个 root 目录为边界。
	mustWriteLeafdocIgnore(t, filepath.Join(tmp, "root"), "drafts/")
	cache := ignore.NewCache(filepath.Join(tmp, "root"))

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := NewPlanner(wiki, tree.NewSlugService(), tmp)
	p.SetIgnoreCache(cache)

	res, err := p.CreatePlan([]ImportMDFile{
		{SourcePath: "readme.md"},
		{SourcePath: "drafts/note.md"},
	}, PlanOptions{
		SourceBasePath: filepath.Join(tmp, "root"),
		TargetBasePath: "",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}

	if len(res.Items) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(res.Items))
	}
	if res.Items[0].TargetPath != "readme" {
		t.Fatalf("expected only readme, got %q", res.Items[0].TargetPath)
	}
}

// TestPlanner_CreatePlan_IncludesNonIgnoredPaths 验证存在忽略文件并不代表过滤所有输入。
// 规则只匹配日志文件，两个 Markdown 源文件都必须进入计划。
func TestPlanner_CreatePlan_IncludesNonIgnoredPaths(t *testing.T) {
	tmp := t.TempDir()
	// 在存储 root 层级建立源文件，让规则查询使用实际目录结构。
	if err := os.MkdirAll(filepath.Join(tmp, "root", "notes"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	testutil.WriteFile(t, tmp, "root/readme.md", "# Readme")
	testutil.WriteFile(t, tmp, "root/notes/note.md", "# Note")

	// 日志后缀规则与两个源文件均不匹配，用作不会过滤页面的对照。
	mustWriteLeafdocIgnore(t, filepath.Join(tmp, "root"), "*.log")
	cache := ignore.NewCache(filepath.Join(tmp, "root"))

	wiki := &fakeWiki{treeHash: "h", lookups: map[string]*tree.PathLookup{}}
	p := NewPlanner(wiki, tree.NewSlugService(), tmp)
	p.SetIgnoreCache(cache)

	res, err := p.CreatePlan([]ImportMDFile{
		{SourcePath: "readme.md"},
		{SourcePath: "notes/note.md"},
	}, PlanOptions{
		SourceBasePath: filepath.Join(tmp, "root"),
		TargetBasePath: "",
	})
	if err != nil {
		t.Fatalf("CreatePlan err: %v", err)
	}

	if len(res.Items) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(res.Items))
	}
}

// mustWriteLeafdocIgnore 将指定规则写入目标目录的忽略文件，父目录由调用方建立。
func mustWriteLeafdocIgnore(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".leafdocignore"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .leafdocignore: %v", err)
	}
}
