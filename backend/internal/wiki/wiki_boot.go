package wiki

import (
	"fmt"
	"path/filepath"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/avatar"
	"github.com/NianBroken/LeafDoc/internal/core/branding"
	"github.com/NianBroken/LeafDoc/internal/core/email"
	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	"github.com/NianBroken/LeafDoc/internal/core/ignore"
	"github.com/NianBroken/LeafDoc/internal/core/links"
	"github.com/NianBroken/LeafDoc/internal/core/properties"
	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// initAuth 先建立用户存储及管理员，再建立会话与可选密钥服务。
// 作者解析器保存动态用户服务入口，供恢复后的查询继续使用当前存储。
func (w *Wiki) initAuth(options *WikiOptions) error {
	// 用户存储构造完成后才创建默认管理员，存储错误在写管理员之前返回。
	store, err := auth.NewUserStore(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init user store: %w", err)
	}
	w.user = auth.NewUserService(store)
	w.user.SetEditorLimit(options.EditorLimit)
	if err := w.user.InitDefaultAdmin(options.AdminUsername, options.AdminEmail, options.AdminPassword); err != nil {
		return err
	}
	w.userResolver, err = auth.NewUserResolver(w.UserService)
	if err != nil {
		return err
	}
	sessionStore, err := auth.NewSessionStore(w.storageDir)
	if err != nil {
		return err
	}
	sessions := auth.NewSessionManager(sessionStore, options.JWTSecret, options.AccessTokenTimeout, options.RefreshTokenTimeout)
	w.auth = auth.NewAuthService(w.user, sessions)

	// 开关关闭时不打开密钥存储，字段保持为空，供路由跳过对应鉴权中间件。
	if options.EnableAPIKeyManagement {
		apiKeyStore, err := auth.NewAPIKeyStore(w.storageDir)
		if err != nil {
			return err
		}
		w.apiKeys = auth.NewAPIKeyService(apiKeyStore, w.auth)
	}
	return nil
}

// initEmail 仅在邮件配置启用时构造令牌存储、邮件发送器与重置及邀请服务。
// 存储保存到实例后由 Close 负责清理，邮件服务使用配置中的公开访问地址生成链接。
func (w *Wiki) initEmail(options *WikiOptions) error {
	if !options.SMTP.Enabled() {
		return nil
	}

	store, err := auth.NewEmailTokenStore(w.storageDir)
	if err != nil {
		return err
	}
	w.emailTokenStore = store

	mailer := email.NewService(options.SMTP)
	w.emailTokenService = auth.NewEmailTokenService(store, w.user, w.auth, mailer, options.SMTP.PublicURL)
	return nil
}

// initCoreServices 将树与资源服务接入同一个忽略规则缓存，加载树后再构造资源入口。
func (w *Wiki) initCoreServices(options *WikiOptions) error {
	// 缓存以 root 内容目录为边界，供多层 .leafdocignore 规则查询复用。
	rootDir := filepath.Join(w.storageDir, "root")
	w.ignoreCache = ignore.NewCache(rootDir)

	w.tree = tree.NewTreeService(w.storageDir)
	w.tree.SetIgnoreCache(w.ignoreCache)
	if err := w.tree.LoadTree(); err != nil {
		return err
	}
	w.slug = tree.NewSlugService()
	w.asset = assets.NewAssetService(w.storageDir, w.slug)
	w.asset.SetIgnoreCache(w.ignoreCache)

	return nil
}

// initLinkService 建立链接存储并同步索引当前页面，索引失败只记录警告。
// 存储构造失败则阻断启动，已构造的链接服务交给 Wiki.Close 管理。
func (w *Wiki) initLinkService() error {
	linksStore, err := links.NewLinksStore(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init links store: %w", err)
	}
	w.links = links.NewLinkService(w.storageDir, w.tree, linksStore)
	if err := w.links.IndexAllPages(); err != nil {
		w.log.Warn("failed to index links on startup", "error", err)
	}
	return nil
}

// initTagsService 将标签存储接入服务，此步骤不读取页面，索引由共享引导流程建立。
func (w *Wiki) initTagsService() error {
	tagsStore, err := tags.NewTagsStore(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init tags store: %w", err)
	}
	w.tags = tags.NewTagsService(tagsStore)
	return nil
}

// initPropertiesService 打开属性存储并保存服务，失败时带上初始化上下文返回。
func (w *Wiki) initPropertiesService() error {
	propsStore, err := properties.NewPropertiesStore(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init properties store: %w", err)
	}
	w.props = properties.NewPropertiesService(propsStore)
	return nil
}

// initFavoritesService 构造用户收藏存储，供页面收藏与账号删除流程共同使用。
func (w *Wiki) initFavoritesService() error {
	store, err := favorites.NewFavoritesStore(w.storageDir, w.log)
	if err != nil {
		return fmt.Errorf("failed to init favorites store: %w", err)
	}
	w.favorites = store
	return nil
}

// bootstrapTagsAndProperties 顺序清空标签和属性索引，再统一取得页面并逐页写入两种索引。
// 清理或枚举失败直接结束，单页读取或单项索引失败记录警告后继续其他处理。
// 该方法不返回错误，也不接收取消上下文，因此调用方不能据此判断每项索引均已成功。
func (w *Wiki) bootstrapTagsAndProperties() {
	if err := w.tags.ClearIndex(); err != nil {
		w.log.Warn("failed to clear tags index before bootstrap", "error", err)
		return
	}
	if err := w.props.ClearIndex(); err != nil {
		w.log.Warn("failed to clear properties index before bootstrap", "error", err)
		return
	}
	var ids []string
	if err := w.tree.WalkNodes(func(id string) error {
		ids = append(ids, id)
		return nil
	}); err != nil {
		w.log.Warn("failed to walk pages for tags/properties bootstrap", "error", err)
		return
	}
	// GetPages 的页面结果与错误按输入标识对齐，一次读取供两种索引使用。
	pages, errs := w.tree.GetPages(ids)
	for i, page := range pages {
		if errs[i] != nil {
			w.log.Warn("skipping page during bootstrap", "pageID", ids[i], "error", errs[i])
			continue
		}
		if err := w.tags.IndexPageContent(page.ID, page.RawContent); err != nil {
			w.log.Warn("failed to index tags", "pageID", page.ID, "error", err)
		}
		if err := w.props.IndexPageContent(page.ID, page.RawContent); err != nil {
			w.log.Warn("failed to index properties", "pageID", page.ID, "error", err)
		}
	}
}

// initSearch 同步打开搜索存储，后台完成全量索引并更新共享状态。
// 后台任务登记到关闭等待组，索引过程使用实例关闭上下文，失败记录状态而不回退启动结果。
func (w *Wiki) initSearch() error {
	var err error
	w.searchIndex, err = search.NewMySQLIndex(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init search index: %w", err)
	}
	w.status = search.NewIndexingStatus()
	searchEffect := pagesave.NewSearchIndexSideEffect(w.searchIndex, w.tree, w.log, w.metrics)
	w.log.Info("search indexing started")
	// 启动协程之前登记，确保 Close 能等待该任务退出后关闭搜索存储。
	w.reloadWG.Add(1)
	go func() {
		defer w.reloadWG.Done()
		w.status.Start()
		defer w.status.Finish()
		if err := searchEffect.IndexAllPagesContext(w.shutdownCtx); err != nil {
			w.log.Warn("search bootstrap failed", "error", err)
			w.status.Fail()
		} else {
			w.log.Info("search indexing completed")
			w.status.Success()
		}
	}()
	return nil
}

// initBranding 读取品牌配置并保存服务，构造错误阻断启动。
func (w *Wiki) initBranding() error {
	var err error
	w.branding, err = branding.NewBrandingService(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init branding service: %w", err)
	}
	return nil
}

// initAvatarService 为当前存储目录建立用户头像服务，失败时返回带上下文的错误。
func (w *Wiki) initAvatarService() error {
	var err error
	w.avatar, err = avatar.NewAvatarService(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init avatar service: %w", err)
	}
	return nil
}

// ensureBaselineRevisions 收集可读取的当前页面，以系统作者和 baseline 原因批量记录历史。
// 枚举失败结束处理，页面读取失败跳过该页，批量记录的每项错误分别写入日志。
// 历史服务决定实际是否需要写入，此方法不强制为已有页面重复创建版本。
func (w *Wiki) ensureBaselineRevisions() {
	var ids []string
	if err := w.tree.WalkNodes(func(id string) error {
		ids = append(ids, id)
		return nil
	}); err != nil {
		w.log.Warn("failed to enumerate pages for baseline revisions", "error", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	// 过滤读取错误与空页面，保持 valid 与后续批量返回错误的下标一一对应。
	pages, pageErrs := w.tree.GetPages(ids)
	var valid []*tree.Page
	for i, p := range pages {
		if pageErrs[i] != nil {
			w.log.Warn("failed to load page for baseline revision", "pageID", ids[i], "error", pageErrs[i])
			continue
		}
		if p != nil {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return
	}
	errs := w.revision.RecordContentUpdates(valid, SYSTEM_USER_ID, "baseline")
	for i, err := range errs {
		if err != nil {
			w.log.Warn("baseline revision failed", "pageID", valid[i].ID, "error", err)
		}
	}
}
