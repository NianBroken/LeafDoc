package wiki

import (
	"context"
	"log/slog"
	"mime/multipart"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	"github.com/NianBroken/LeafDoc/internal/core/links"
	"github.com/NianBroken/LeafDoc/internal/core/properties"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	wikiassets "github.com/NianBroken/LeafDoc/internal/wiki/assets"
	wikipages "github.com/NianBroken/LeafDoc/internal/wiki/pages"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// WikiImportAdapter 将导入模块接口接入 Wiki 的现有用例及核心服务。
// 查询复用树与资源服务，写操作通过页面或资源用例触发索引及历史等附带处理。
// 字段保存构造时的服务指针，不在每次调用中从 Wiki 重新取值。
type WikiImportAdapter struct {
	tree        *tree.TreeService
	slug        *tree.SlugService
	revision    *revision.Service
	links       *links.LinkService
	asset       *assets.AssetService
	tags        *tags.TagsService
	props       *properties.PropertiesService
	searchIndex *search.MySQLIndex
	log         *slog.Logger
}

// NewWikiImportAdapter 保存当前 Wiki 的服务与日志，不打开额外存储，也不复制服务状态。
func NewWikiImportAdapter(w *Wiki) *WikiImportAdapter {
	return &WikiImportAdapter{
		tree:        w.tree,
		slug:        w.slug,
		revision:    w.revision,
		links:       w.links,
		asset:       w.asset,
		tags:        w.tags,
		props:       w.props,
		searchIndex: w.searchIndex,
		log:         w.log,
	}
}

// TreeHash 转交树哈希，用于导入计划检查规划依据是否仍与当前树一致。
func (a *WikiImportAdapter) TreeHash() string {
	return a.tree.TreeHash()
}

// LookupPagePath 使用树服务的路径分析结果，保留其节点及查找错误语义。
func (a *WikiImportAdapter) LookupPagePath(path string) (*tree.PathLookup, error) {
	return a.tree.LookupPagePath(path)
}

// FindByPath 按页面路由路径查找，不将 route 当作磁盘文件位置读取。
func (a *WikiImportAdapter) FindByPath(route string) (*tree.Page, error) {
	return a.tree.FindPageByRoutePath(route)
}

// ListAssets 先验证页面存在，再返回资源服务提供的该页资源地址列表。
func (a *WikiImportAdapter) ListAssets(pageID string) ([]string, error) {
	page, err := a.tree.FindPageByID(pageID)
	if err != nil {
		return nil, err
	}
	return a.asset.ListAssetsForPage(page)
}

// orchestrator 为导入写操作建立保存编排器，按搜索、链接、标签、属性、历史的顺序配置处理器。
// 所有指标入口传入空值，导入路径不通过此编排器累计 HTTPMetrics 保存指标。
func (a *WikiImportAdapter) orchestrator() *pagesave.PageSaveOrchestrator {
	return pagesave.NewPageSaveOrchestrator(nil,
		pagesave.NewSearchIndexSideEffect(a.searchIndex, a.tree, a.log, nil),
		pagesave.NewLinkIndexSideEffect(a.links, a.log, nil),
		pagesave.NewTagsSideEffect(a.tags, a.log, nil),
		pagesave.NewPropertiesSideEffect(a.props, a.log, nil),
		pagesave.NewRevisionSideEffect(a.revision, a.log, nil),
	)
}

// EnsurePath 通过已有路径用例创建所需层级，并将用例输出解包为页面。
// 接口未接收请求上下文，这里使用后台上下文，不跟随 HTTP 请求取消。
func (a *WikiImportAdapter) EnsurePath(userID, targetPath, title string, kind *tree.NodeKind) (*tree.Page, error) {
	out, err := wikipages.NewEnsurePathUseCase(a.tree, a.slug, a.orchestrator(), a.log).Execute(
		context.Background(),
		wikipages.EnsurePathInput{UserID: userID, TargetPath: targetPath, TargetTitle: title, Kind: kind},
	)
	if err != nil {
		return nil, err
	}
	return out.Page, nil
}

// UpdatePage 先读取当前版本，再通过更新用例提交导入内容并要求保留前置元数据。
// 读取与更新之间存在其他写入时，用例继续按版本规则判断冲突，不绕过并发保护。
func (a *WikiImportAdapter) UpdatePage(userID, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error) {
	current, err := a.tree.GetPage(id)
	if err != nil {
		return nil, err
	}
	out, err := wikipages.NewUpdatePageUseCase(a.tree, a.slug, a.orchestrator(), a.log, nil).Execute(
		context.Background(),
		wikipages.UpdatePageInput{UserID: userID, ID: id, Version: current.Version(), Title: title, Slug: slug, Content: content, Kind: kind, PreserveFrontmatter: true},
	)
	if err != nil {
		return nil, err
	}
	return out.Page, nil
}

// UploadAsset 将文件、名称与字节限制交给资源上传用例，成功时只返回公开资源地址。
// 错误原样交回导入执行器，文件句柄的所有权仍遵循上传用例与调用方约定。
func (a *WikiImportAdapter) UploadAsset(userID, pageID string, file multipart.File, filename string, maxBytes int64) (string, error) {
	out, err := wikiassets.NewUploadAssetUseCase(a.tree, a.asset, a.revision, a.log).Execute(
		context.Background(),
		wikiassets.UploadAssetInput{UserID: userID, PageID: pageID, File: file, Filename: filename, MaxBytes: maxBytes},
	)
	if err != nil {
		return "", err
	}
	return out.URL, nil
}
