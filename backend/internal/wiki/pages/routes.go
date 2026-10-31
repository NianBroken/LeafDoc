package pages

import (
	"net/http"
	"strconv"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/NianBroken/LeafDoc/internal/http/dto"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// 页面标识路由与请求错误消息由本模块各处理器共用，响应文字属于接口内容。
const (
	pagesIdRoutePath         = "/pages/:id"
	errInvalidRequestUserMsg = "Invalid request"
	errInvalidRequestLogMsg  = "invalid request"
)

// Routes 保存页面接口用例、树查询及身份解析依赖，负责请求转换和响应组装。
// 业务写入由各用例执行，路由注册处统一设置认证与编辑角色限制。
type Routes struct {
	treeService     *tree.TreeService
	createPage      *CreatePageUseCase
	updatePage      *UpdatePageUseCase
	deletePage      *DeletePageUseCase
	movePage        *MovePageUseCase
	convertPage     *ConvertPageUseCase
	copyPage        *CopyPageUseCase
	queries         *PageQueries
	sortPages       *SortPagesUseCase
	ensurePath      *EnsurePathUseCase
	suggestSlug     *SuggestSlugUseCase
	previewRefactor *PreviewPageRefactorUseCase
	applyRefactor   *ApplyPageRefactorUseCase
	pinPage         *PinPageUseCase
	addFavorite     *AddFavoriteUseCase
	removeFavorite  *RemoveFavoriteUseCase
	listFavorites   *ListFavoritesUseCase
	userResolver    *coreauth.UserResolver
	authService     *coreauth.AuthService
}

// RoutesConfig 提供路由装配所需全部依赖，构造器逐项保存，不自动创建缺失服务。
type RoutesConfig struct {
	TreeService     *tree.TreeService
	CreatePage      *CreatePageUseCase
	UpdatePage      *UpdatePageUseCase
	DeletePage      *DeletePageUseCase
	MovePage        *MovePageUseCase
	ConvertPage     *ConvertPageUseCase
	CopyPage        *CopyPageUseCase
	Queries         *PageQueries
	SortPages       *SortPagesUseCase
	EnsurePath      *EnsurePathUseCase
	SuggestSlug     *SuggestSlugUseCase
	PreviewRefactor *PreviewPageRefactorUseCase
	ApplyRefactor   *ApplyPageRefactorUseCase
	PinPage         *PinPageUseCase
	AddFavorite     *AddFavoriteUseCase
	RemoveFavorite  *RemoveFavoriteUseCase
	ListFavorites   *ListFavoritesUseCase
	UserResolver    *coreauth.UserResolver
	AuthService     *coreauth.AuthService
}

// NewRoutes 将配置依赖绑定为路由对象，实际注册由 RegisterRoutes 调用完成。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		treeService:     cfg.TreeService,
		createPage:      cfg.CreatePage,
		updatePage:      cfg.UpdatePage,
		deletePage:      cfg.DeletePage,
		movePage:        cfg.MovePage,
		convertPage:     cfg.ConvertPage,
		copyPage:        cfg.CopyPage,
		queries:         cfg.Queries,
		sortPages:       cfg.SortPages,
		ensurePath:      cfg.EnsurePath,
		suggestSlug:     cfg.SuggestSlug,
		previewRefactor: cfg.PreviewRefactor,
		applyRefactor:   cfg.ApplyRefactor,
		pinPage:         cfg.PinPage,
		addFavorite:     cfg.AddFavorite,
		removeFavorite:  cfg.RemoveFavorite,
		listFavorites:   cfg.ListFavorites,
		userResolver:    cfg.UserResolver,
		authService:     cfg.AuthService,
	}
}

// RegisterRoutes 将读接口、认证接口和编辑接口注册到对应路由组。
// 链接重构接口是否存在由注册时的 EnableLinkRefactor 配置决定。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	opts := ctx.Opts

	// 读取组由 APIReadGroup 在请求时判断公开访问状态，不在这里重复注册路由。
	readGroup := ctx.APIReadGroup(r.authService)
	readGroup.GET("/tree", r.handleGetTree)
	readGroup.GET(pagesIdRoutePath, r.handleGetPage)
	readGroup.GET("/pages/lookup", r.handleLookupPath)
	readGroup.GET("/pages/by-path", r.handleGetByPath)
	readGroup.GET("/pages/by-title", r.handleFindByTitle)
	readGroup.GET("/pages/permalink/:id", r.handleResolvePermalink)

	// 下列接口都要求真实认证用户，编辑写入及名称建议另外要求编辑者或管理员角色。
	authGroup := ctx.APIAuthGroup(r.authService)

	authGroup.GET("/pages/slug-suggestion", authmw.RequireEditorOrAdmin(), r.handleSuggestSlug)
	authGroup.POST("/pages", authmw.RequireEditorOrAdmin(), r.handleCreate)
	authGroup.PUT(pagesIdRoutePath, authmw.RequireEditorOrAdmin(), r.handleUpdate)
	authGroup.DELETE(pagesIdRoutePath, authmw.RequireEditorOrAdmin(), r.handleDelete)
	authGroup.PUT("/pages/:id/move", authmw.RequireEditorOrAdmin(), r.handleMove)
	authGroup.PUT("/pages/:id/sort", authmw.RequireEditorOrAdmin(), r.handleSort)
	authGroup.PUT("/pages/:id/pin", authmw.RequireEditorOrAdmin(), r.handlePin)
	// 个人收藏只使用认证组限制，不附加编辑角色要求，用户标识从认证上下文取得。
	authGroup.PUT("/pages/:id/favorite", r.handleAddFavorite)
	authGroup.DELETE("/pages/:id/favorite", r.handleRemoveFavorite)
	authGroup.GET("/favorites", r.handleListFavorites)
	authGroup.POST("/pages/ensure", authmw.RequireEditorOrAdmin(), r.handleEnsurePath)
	authGroup.POST("/pages/convert/:id", authmw.RequireEditorOrAdmin(), r.handleConvert)
	authGroup.POST("/pages/copy/:id", authmw.RequireEditorOrAdmin(), r.handleCopy)
	if opts.EnableLinkRefactor {
		authGroup.POST("/pages/:id/refactor/preview", authmw.RequireEditorOrAdmin(), r.handleRefactorPreview)
		authGroup.POST("/pages/:id/refactor/apply", authmw.RequireEditorOrAdmin(), r.handleRefactorApply)
	}
}

// handleGetByPath 要求非空路径，分区响应包含一层子节点，普通页面深度为零。
func (r *Routes) handleGetByPath(c *gin.Context) {
	routePath := strings.TrimSpace(c.Query("path"))
	if routePath == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingPath, "Missing path", "missing path")
		return
	}
	page, err := r.queries.FindByPath(routePath)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	depth := 0
	if page.Kind == tree.NodeKindSection {
		depth = 1
	}
	r.respondPageWithDepth(c, http.StatusOK, page, depth)
}

// handleFindByTitle 去除查询两端空白后按标题查找，空值返回明确的缺少标题错误。
func (r *Routes) handleFindByTitle(c *gin.Context) {
	title := strings.TrimSpace(c.Query("title"))
	if title == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingTitle, "Missing title query parameter", "missing title")
		return
	}
	out := r.queries.FindByTitle(title)
	c.JSON(http.StatusOK, out)
}

// handleLookupPath 将允许为空的路径交给逐段查询，返回存在与缺失段的解析结果。
func (r *Routes) handleLookupPath(c *gin.Context) {
	path := strings.TrimSpace(c.Query("path"))
	lookup, err := r.queries.LookupPath(path)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, lookup)
}

// handleResolvePermalink 从路由读取稳定页面标识，返回解析目标而不在此发起跳转。
func (r *Routes) handleResolvePermalink(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingID, "Page ID is required", "page id is required")
		return
	}
	target, err := r.queries.ResolvePermalink(id)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, target)
}

// handleSuggestSlug 拒绝空白标题，其余标题交给建议用例，包括纯标点标题。
// 建议为空仍以成功 JSON 返回，由输入校验流程处理最终创建名称。
func (r *Routes) handleSuggestSlug(c *gin.Context) {
	title := strings.TrimSpace(c.Query("title"))
	if title == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingTitle, "Title query param is required", "title query param is required")
		return
	}
	out, err := r.suggestSlug.Execute(c.Request.Context(), SuggestSlugInput{
		ParentID:  strings.TrimSpace(c.Query("parentId")),
		CurrentID: strings.TrimSpace(c.Query("currentId")),
		Title:     title,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"slug": out.Slug})
}

// handleCreate 绑定创建 JSON，取得认证用户后转换可选类型并调用创建用例。
// 未识别类型沿用 kindFromString 的页面默认值，成功响应使用 201。
func (r *Routes) handleCreate(c *gin.Context) {
	var req struct {
		ParentID *string `json:"parentId"`
		Title    string  `json:"title" binding:"required"`
		Slug     string  `json:"slug" binding:"required"`
		Kind     *string `json:"kind"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	kind := kindFromString(req.Kind)
	out, err := r.createPage.Execute(c.Request.Context(), CreatePageInput{
		UserID: user.ID, ParentID: req.ParentID, Title: req.Title, Slug: req.Slug, Kind: &kind,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusCreated, out.Page)
}

// handleUpdate 要求版本、标题和路径段，先校验结构化标签属性再进入保存用例。
// Content 指针区分省略正文与明确空正文，标签也保留 nil 与空列表的差别。
func (r *Routes) handleUpdate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version    string            `json:"version" binding:"required"`
		Title      string            `json:"title" binding:"required"`
		Slug       string            `json:"slug" binding:"required"`
		Content    *string           `json:"content"`
		Tags       []string          `json:"tags"`
		Properties map[string]string `json:"properties"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	if err := validatePageMetadataInput(req.Tags, req.Properties); err != nil {
		respondWithPageError(c, err)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	// 校验通过的标签再转换为小写，保存正文元信息与派生索引共用相同文字。
	// 未提供字段时保留 nil，明确空列表则用于清空标签。
	normalizedTags := req.Tags
	if req.Tags != nil {
		normalizedTags = normalizeTagInputs(req.Tags)
	}

	kind := tree.NodeKindPage
	out, err := r.updatePage.Execute(c.Request.Context(), UpdatePageInput{
		UserID: user.ID, ID: id, Version: req.Version, Title: req.Title, Slug: req.Slug,
		Content: req.Content, Kind: &kind,
		Tags: normalizedTags, Properties: req.Properties,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, out.Page)
}

// handleDelete 从查询串读取版本，recursive 仅在文字精确为 true 时启用。
// 用户身份取认证上下文，成功返回删除确认 JSON，不返回被删除的正文。
func (r *Routes) handleDelete(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	recursive := c.DefaultQuery("recursive", "false") == "true"
	version := c.Query("version")
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.deletePage.Execute(c.Request.Context(), DeletePageInput{
		UserID: user.ID, ID: id, Version: version, Recursive: recursive,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page deleted"})
}

// handleMove 要求版本字段，目标父标识和位置允许省略，直接传给移动用例。
func (r *Routes) handleMove(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version  string `json:"version" binding:"required"`
		ParentID string `json:"parentId"`
		Position *int   `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidPayload, "Invalid payload", "invalid payload")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.movePage.Execute(c.Request.Context(), MovePageInput{
		UserID: user.ID, ID: id, Version: req.Version, ParentID: req.ParentID, Position: req.Position,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page moved"})
}

// handleSort 将路由标识作为父节点，并提交完整子标识排列供树服务校验。
// JSON 绑定成功不代表列表有效，重复、遗漏和外部标识由下层拒绝。
func (r *Routes) handleSort(c *gin.Context) {
	parentID := strings.TrimSpace(c.Param("id"))
	var req struct {
		OrderedIDs []string `json:"orderedIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	if err := r.sortPages.Execute(c.Request.Context(), SortPagesInput{
		ParentID: parentID, OrderedIDs: req.OrderedIDs,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pages sorted successfully"})
}

// handlePin 要求版本，省略 pinned 时使用布尔零值 false，即提交取消置顶。
// 成功返回组装后的页面响应，不单独读取认证用户字段。
func (r *Routes) handlePin(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version string `json:"version" binding:"required"`
		Pinned  bool   `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.pinPage.Execute(c.Request.Context(), PinPageInput{
		ID:      id,
		Version: req.Version,
		Pinned:  req.Pinned,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, out.Page)
}

// handleAddFavorite 只为当前认证用户添加收藏，不从请求接收其他用户标识。
func (r *Routes) handleAddFavorite(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.addFavorite.Execute(c.Request.Context(), AddFavoriteInput{
		UserID: user.ID, PageID: id,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page favorited"})
}

// handleListFavorites 读取当前用户收藏后转换为 API 页面，并从原始元信息填入标签属性。
// 单页元信息读取失败沿用 enrichPageMetadata 的空列表回退，不使整个响应失败。
func (r *Routes) handleListFavorites(c *gin.Context) {
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	out, err := r.listFavorites.Execute(c.Request.Context(), ListFavoritesInput{UserID: user.ID})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	apiPages := make([]*dto.Page, 0, len(out.Pages))
	for _, p := range out.Pages {
		apiPage := dto.ToAPIPage(p, r.userResolver)
		r.enrichPageMetadata(apiPage)
		apiPages = append(apiPages, apiPage)
	}
	c.JSON(http.StatusOK, gin.H{"pages": apiPages})
}

// handleEnsurePath 要求路径和标题，类型由可选字符串转换，成功返回路径最终页面。
func (r *Routes) handleEnsurePath(c *gin.Context) {
	var req struct {
		Path  string  `json:"path" binding:"required"`
		Title string  `json:"title" binding:"required"`
		Kind  *string `json:"kind"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	kind := kindFromString(req.Kind)
	out, err := r.ensurePath.Execute(c.Request.Context(), EnsurePathInput{
		UserID: user.ID, TargetPath: req.Path, TargetTitle: req.Title, Kind: &kind,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, out.Page)
}

// handleConvert 要求版本和目标类型，并显式限制类型只能为 page 或 section。
// 转换完成返回 204，没有页面 JSON 正文。
func (r *Routes) handleConvert(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Kind    string `json:"targetKind" binding:"required"`
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	if req.Kind != "page" && req.Kind != "section" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidTargetKind, "Invalid targetKind", "invalid target kind")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.convertPage.Execute(c.Request.Context(), ConvertPageInput{
		UserID: user.ID, ID: id, Version: req.Version, TargetKind: tree.NodeKind(req.Kind),
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleCopy 从路由取得来源页，目标父标识、标题和路径段来自 JSON。
// 成功使用 201 返回复制得到的页面，来源内容读取和资源处理由复制用例完成。
func (r *Routes) handleCopy(c *gin.Context) {
	sourceID := strings.TrimSpace(c.Param("id"))
	var req struct {
		ParentID *string `json:"targetParentId"`
		Title    string  `json:"title" binding:"required"`
		Slug     string  `json:"slug" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	out, err := r.copyPage.Execute(c.Request.Context(), CopyPageInput{
		UserID: user.ID, SourcePageID: sourceID, TargetParentID: req.ParentID,
		Title: req.Title, Slug: req.Slug,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusCreated, out.Page)
}

// handleRefactorPreview 绑定重构方案输入后返回预览，不在处理器中应用正文写入。
// 预览不接收版本字段，具体可执行条件由预览用例检查。
func (r *Routes) handleRefactorPreview(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Kind        string  `json:"kind" binding:"required"`
		Title       string  `json:"title"`
		Slug        string  `json:"slug"`
		Content     *string `json:"content"`
		NewParentID *string `json:"parentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.previewRefactor.Execute(c.Request.Context(), RefactorPreviewInput{
		PageID: id, Kind: req.Kind, Title: req.Title, Slug: req.Slug,
		Content: req.Content, NewParentID: req.NewParentID,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// handleRefactorApply 接收版本、操作类型、链接重写选择和可选位置，取得用户后执行。
// 输入 JSON 只负责字段绑定，具体方案计算和保存由应用用例处理。
func (r *Routes) handleRefactorApply(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version      string  `json:"version" binding:"required"`
		Kind         string  `json:"kind" binding:"required"`
		Title        string  `json:"title"`
		Slug         string  `json:"slug"`
		Content      *string `json:"content"`
		NewParentID  *string `json:"parentId"`
		RewriteLinks bool    `json:"rewriteLinks"`
		Position     *int    `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	page, err := r.applyRefactor.Execute(c.Request.Context(), RefactorApplyInput{
		Version: req.Version,
		UserID:  user.ID,
		RefactorPreviewInput: RefactorPreviewInput{
			PageID: id, Kind: req.Kind, Title: req.Title, Slug: req.Slug,
			Content: req.Content, NewParentID: req.NewParentID,
		},
		RewriteLinks: req.RewriteLinks,
		Position:     req.Position,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, page)
}

// respondPage 将领域页面转换为 API 格式，解析作者信息并填充标签属性后按指定状态响应。
func (r *Routes) respondPage(c *gin.Context, status int, page *tree.Page) {
	apiPage := dto.ToAPIPage(page, r.userResolver)
	r.enrichPageMetadata(apiPage)
	c.JSON(status, apiPage)
}

// respondPageWithDepth 通过 DTO 转换限制子节点展开深度，当前页元信息仍独立读取。
func (r *Routes) respondPageWithDepth(c *gin.Context, status int, page *tree.Page, depth int) {
	apiPage := dto.ToAPIPageWithDepth(page, r.userResolver, depth)
	r.enrichPageMetadata(apiPage)
	c.JSON(status, apiPage)
}

// enrichPageMetadata 先设置空标签和属性，再读取当前页面的完整 Markdown 提取附加字段。
// 读取失败、解析失败或缺少附加字段均保留空集合，不在此输出额外错误响应。
func (r *Routes) enrichPageMetadata(page *dto.Page) {
	if page == nil {
		return
	}

	page.Tags = []string{}
	page.Properties = map[string]string{}

	raw, err := r.treeService.ReadPageRaw(page.ID)
	if err != nil {
		return
	}

	fm, _, has, err := markdown.ParseFrontmatter(raw)
	if err != nil || !has || len(fm.ExtraFields) == 0 {
		return
	}

	tags, properties := extractPageMetadata(fm.ExtraFields)
	page.Tags = tags
	page.Properties = properties
}

// extractPageMetadata 把不区分大小写的 tags 键单独处理，系统键不进入属性表。
// 其余字段递归展开为单行字符串属性，普通 title 键仍属于自定义属性。
func extractPageMetadata(fields map[string]interface{}) ([]string, map[string]string) {
	tags := []string{}
	properties := map[string]string{}

	for rawKey, value := range fields {
		key := strings.TrimSpace(rawKey)
		lower := strings.ToLower(key)

		if lower == "tags" {
			tags = normalizeMetadataTags(value)
			continue
		}
		if markdown.IsSystemKey(key) {
			continue
		}

		flattenMetadataEntry(key, value, properties)
	}

	return tags, properties
}

// flattenMetadataEntry 从深度零开始展开嵌套映射，父子键用点号连接。
// 例如 a 内部的 b 字符串值对应属性键 a.b，不把整个映射转成文字。
func flattenMetadataEntry(prefix string, value interface{}, properties map[string]string) {
	flattenMetadataEntryDepth(prefix, value, properties, 0)
}

// flattenMetadataEntryDepth 在到达深度上限前只处理字符串与字符串键映射。
// 字符串去空白后必须非空且没有换行，已有属性键不覆盖；其他类型直接忽略。
// 子键去除空白，空键和 leafdoc_ 前缀跳过，递归不改变原始字段映射。
func flattenMetadataEntryDepth(prefix string, value interface{}, properties map[string]string, depth int) {
	if depth >= maxFlattenDepth {
		return
	}
	switch v := value.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s != "" && !strings.ContainsRune(s, '\n') {
			if _, exists := properties[prefix]; !exists {
				properties[prefix] = s
			}
		}
	case map[string]interface{}:
		for childKey, childValue := range v {
			childKey = strings.TrimSpace(childKey)
			if childKey == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(childKey), "leafdoc_") {
				continue
			}
			flattenMetadataEntryDepth(prefix+"."+childKey, childValue, properties, depth+1)
		}
	}
}

// maxFlattenDepth 限制映射递归层数，深度达到此值的节点不再展开或保存。
const maxFlattenDepth = 20

// normalizeMetadataTags 仅接受通用切片形式，过滤非字符串元素再执行标签规范处理。
// 非列表值返回空列表，不把逗号分隔文本或其他类型自动转为标签。
func normalizeMetadataTags(value interface{}) []string {
	list, ok := value.([]interface{})
	if !ok {
		return []string{}
	}

	rawTags := make([]string, 0, len(list))
	for _, item := range list {
		tag, ok := item.(string)
		if !ok {
			continue
		}
		rawTags = append(rawTags, tag)
	}

	return normalizeTagInputs(rawTags)
}

// normalizeTagInputs 去两端空白并转换小写，移除空值和重复值，保留首次出现顺序。
func normalizeTagInputs(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result
}

// validatePageMetadataInput 累计标签与属性键的字段错误，不修改原输入。
// 标签禁止空白、首尾空格和不区分大小写的重复；属性键禁止空值、首尾空格及系统键。
// 属性值在此不做长度、换行或内容限制，错误路径保留具体下标或原始属性键。
func validatePageMetadataInput(tags []string, properties map[string]string) error {
	ve := sharederrors.NewValidationErrors()
	seenTags := map[string]struct{}{}

	for index, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		field := "tags[" + strconv.Itoa(index) + "]"
		if trimmed == "" {
			ve.Add(field, "Tag must not be empty")
			continue
		}
		if trimmed != tag {
			ve.Add(field, "Tag must not contain leading or trailing whitespace")
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seenTags[key]; exists {
			ve.Add(field, "Tag must be unique")
			continue
		}
		seenTags[key] = struct{}{}
	}

	for rawKey := range properties {
		key := strings.TrimSpace(rawKey)
		field := "properties." + rawKey
		switch {
		case key == "":
			ve.Add(field, "Property key must not be empty")
		case key != rawKey:
			ve.Add(field, "Property key must not contain leading or trailing whitespace")
		case markdown.IsSystemKey(key):
			ve.Add(field, "Property key is reserved")
		}
	}

	if ve.HasErrors() {
		return ve
	}

	return nil
}

// kindFromString 只将精确的 section 文字识别为分区，其余值和空指针统一返回页面类型。
func kindFromString(s *string) tree.NodeKind {
	if s != nil && *s == string(tree.NodeKindSection) {
		return tree.NodeKindSection
	}
	return tree.NodeKindPage
}
