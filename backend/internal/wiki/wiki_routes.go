package wiki

import (
	"path/filepath"

	coreimporter "github.com/NianBroken/LeafDoc/internal/core/importer"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	wikiapikeys "github.com/NianBroken/LeafDoc/internal/wiki/apikeys"
	wikiassets "github.com/NianBroken/LeafDoc/internal/wiki/assets"
	wikiauth "github.com/NianBroken/LeafDoc/internal/wiki/auth"
	wikiavatar "github.com/NianBroken/LeafDoc/internal/wiki/avatar"
	wikibranding "github.com/NianBroken/LeafDoc/internal/wiki/branding"
	wikihealth "github.com/NianBroken/LeafDoc/internal/wiki/health"
	wikiimporter "github.com/NianBroken/LeafDoc/internal/wiki/importer"
	wikiinstancesettings "github.com/NianBroken/LeafDoc/internal/wiki/instancesettings"
	wikilinks "github.com/NianBroken/LeafDoc/internal/wiki/links"
	wikipages "github.com/NianBroken/LeafDoc/internal/wiki/pages"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
	wikiproperties "github.com/NianBroken/LeafDoc/internal/wiki/properties"
	wikirestore "github.com/NianBroken/LeafDoc/internal/wiki/restore"
	wikiresync "github.com/NianBroken/LeafDoc/internal/wiki/resync"
	wikirevisions "github.com/NianBroken/LeafDoc/internal/wiki/revisions"
	wikisearch "github.com/NianBroken/LeafDoc/internal/wiki/search"
	wikisnapshot "github.com/NianBroken/LeafDoc/internal/wiki/snapshot"
	wikitags "github.com/NianBroken/LeafDoc/internal/wiki/tags"
)

// buildRoutes 为各领域构造用例与路由对象，此时不向 Gin 引擎注册具体 URL。
// 大部分路由共享已经初始化的服务，健康检查共享搜索状态，同步路由共享后台任务对象。
// 快照、恢复及实例设置路由由外部设置入口另行提供。
func (w *Wiki) buildRoutes(options *WikiOptions) {
	w.pagesRoutes = w.buildPagesRoutes()
	w.authRoutes = w.buildAuthRoutes()
	w.assetsRoutes = w.buildAssetsRoutes()
	w.revisionsRoutes = w.buildRevisionsRoutes()
	w.searchRoutes = w.buildSearchRoutes()
	w.linksRoutes = w.buildLinksRoutes()
	w.tagsRoutes = w.buildTagsRoutes()
	w.propertiesRoutes = w.buildPropertiesRoutes()
	w.brandingRoutes = w.buildBrandingRoutes()
	w.avatarRoutes = w.buildAvatarRoutes()

	w.apiKeysRoutes = w.buildAPIKeysRoutes()
	w.importerRoutes = w.buildImporterRoutes(options)
	w.healthRoutes = wikihealth.NewRoutes(wikihealth.RoutesConfig{
		Index:      w.searchIndex,
		Status:     w.status,
		StorageDir: w.storageDir,
	})
	w.resyncRoutes = wikiresync.NewRoutes(
		wikiresync.NewTriggerResyncUseCase(w.resyncJob, w.launchReloadWithProgress, w.metrics),
		wikiresync.NewGetResyncStatusUseCase(w.resyncJob),
		w.auth,
	)
}

// newPageOrchestrator 按搜索、链接、历史、标签、属性的顺序建立页面保存附带处理。
// 每个处理器引用已有服务并接入同一日志与指标，未启用的历史服务以空值传递。
func (w *Wiki) newPageOrchestrator() *pagesave.PageSaveOrchestrator {
	return pagesave.NewPageSaveOrchestrator(
		w.metrics,
		pagesave.NewSearchIndexSideEffect(w.searchIndex, w.tree, w.log, w.metrics),
		pagesave.NewLinkIndexSideEffect(w.links, w.log, w.metrics),
		pagesave.NewRevisionSideEffect(w.revision, w.log, w.metrics),
		pagesave.NewTagsSideEffect(w.tags, w.log, w.metrics),
		pagesave.NewPropertiesSideEffect(w.props, w.log, w.metrics),
	)
}

// buildPagesRoutes 将页面查询、修改、重构、收藏和作者解析组装到页面领域路由。
// 常规写用例共用本方法建立的编排器，重构与转换按各自用例直接接收所需服务。
func (w *Wiki) buildPagesRoutes() *wikipages.Routes {
	o := w.newPageOrchestrator()
	return wikipages.NewRoutes(wikipages.RoutesConfig{
		TreeService:     w.tree,
		CreatePage:      wikipages.NewCreatePageUseCase(w.tree, w.slug, o, w.log, w.metrics),
		UpdatePage:      wikipages.NewUpdatePageUseCase(w.tree, w.slug, o, w.log, w.metrics),
		DeletePage:      wikipages.NewDeletePageUseCase(w.tree, w.revision, w.asset, w.favorites, o, w.log, w.metrics),
		MovePage:        wikipages.NewMovePageUseCase(w.tree, o, w.log, w.metrics),
		ConvertPage:     wikipages.NewConvertPageUseCase(w.tree, w.revision, w.log),
		CopyPage:        wikipages.NewCopyPageUseCase(w.tree, w.slug, o, w.asset, w.log),
		Queries:         wikipages.NewPageQueries(w.tree),
		SortPages:       wikipages.NewSortPagesUseCase(w.tree),
		EnsurePath:      wikipages.NewEnsurePathUseCase(w.tree, w.slug, o, w.log),
		SuggestSlug:     wikipages.NewSuggestSlugUseCase(w.tree, w.slug),
		PreviewRefactor: wikipages.NewPreviewPageRefactorUseCase(w.tree, w.slug, w.links, w.log),
		ApplyRefactor:   wikipages.NewApplyPageRefactorUseCase(w.tree, w.slug, w.revision, w.links, w.log, w.metrics),
		PinPage:         wikipages.NewPinPageUseCase(w.tree, w.log),
		AddFavorite:     wikipages.NewAddFavoriteUseCase(w.tree, w.favorites),
		RemoveFavorite:  wikipages.NewRemoveFavoriteUseCase(w.favorites),
		ListFavorites:   wikipages.NewListFavoritesUseCase(w.tree, w.favorites, w.log),
		UserResolver:    w.userResolver,
		AuthService:     w.auth,
	})
}

// buildAuthRoutes 将登录会话、用户管理和邮件凭据流程交给各自用例。
// 用户与邮件服务使用方法入口动态获取，路由对象不固定保存构造时的用户存储。
func (w *Wiki) buildAuthRoutes() *wikiauth.Routes {
	return wikiauth.NewRoutes(wikiauth.RoutesConfig{
		Login:             wikiauth.NewLoginUseCase(w.auth, w.metrics),
		Logout:            wikiauth.NewLogoutUseCase(w.auth, w.metrics),
		RefreshToken:      wikiauth.NewRefreshTokenUseCase(w.auth, w.metrics),
		CreateUser:        wikiauth.NewCreateUserUseCase(w.UserService, w.userResolver, w.log),
		UpdateUser:        wikiauth.NewUpdateUserUseCase(w.UserService, w.userResolver, w.auth, w.log),
		ChangeOwnPassword: wikiauth.NewChangeOwnPasswordUseCase(w.UserService, w.auth, w.log),
		DeleteUser:        wikiauth.NewDeleteUserUseCase(w.UserService, w.userResolver, w.favorites, w.apiKeys, w.log),
		GetUsers:          wikiauth.NewGetUsersUseCase(w.UserService),
		GetUserByID:       wikiauth.NewGetUserByIDUseCase(w.UserService),
		AuthService:       w.auth,

		RequestPasswordReset: wikiauth.NewRequestPasswordResetUseCase(w.EmailTokenService),
		ConfirmPasswordReset: wikiauth.NewConfirmPasswordResetUseCase(w.EmailTokenService),
		InviteUser:           wikiauth.NewInviteUserUseCase(w.UserService, w.EmailTokenService, w.userResolver, w.log),
		ResendInvite:         wikiauth.NewResendInviteUseCase(w.UserService, w.EmailTokenService),
		ConfirmInvite:        wikiauth.NewConfirmInviteUseCase(w.EmailTokenService, w.auth),
	})
}

// buildAssetsRoutes 接入页面存在性检查、资源读写和可选历史记录。
// 文件服务目录直接取资源服务目录，确保文件响应与上传使用同一位置。
func (w *Wiki) buildAssetsRoutes() *wikiassets.Routes {
	return wikiassets.NewRoutes(wikiassets.RoutesConfig{
		Upload:      wikiassets.NewUploadAssetUseCase(w.tree, w.asset, w.revision, w.log),
		List:        wikiassets.NewListAssetsUseCase(w.tree, w.asset),
		Rename:      wikiassets.NewRenameAssetUseCase(w.tree, w.asset, w.revision, w.log),
		Delete:      wikiassets.NewDeleteAssetUseCase(w.tree, w.asset, w.revision, w.log),
		AuthService: w.auth,
		AssetsDir:   w.asset.GetAssetsDir(),
		Log:         w.log,
	})
}

// buildRevisionsRoutes 将历史查询、比较、资源读取、完整性检查与恢复用例集中交给历史路由。
// 单页恢复使用独立保存编排器，使恢复内容继续经过派生索引及其他保存处理。
func (w *Wiki) buildRevisionsRoutes() *wikirevisions.Routes {
	return wikirevisions.NewRoutes(wikirevisions.RoutesConfig{
		ListRevisions:    wikirevisions.NewListRevisionsUseCase(w.revision),
		GetRevision:      wikirevisions.NewGetRevisionUseCase(w.revision),
		CompareRevisions: wikirevisions.NewCompareRevisionsUseCase(w.revision),
		GetRevisionAsset: wikirevisions.NewGetRevisionAssetUseCase(w.revision),
		GetLatest:        wikirevisions.NewGetLatestRevisionUseCase(w.revision),
		RestoreRevision:  wikirevisions.NewRestoreRevisionUseCase(w.revision, w.tree, w.newPageOrchestrator(), w.log, w.metrics),
		CheckIntegrity:   wikirevisions.NewCheckIntegrityUseCase(w.revision),
		UserResolver:     w.userResolver,
		AuthService:      w.auth,
	})
}

// buildSearchRoutes 将搜索索引、标签及树交给查询用例，状态用例读取后台索引共享状态。
func (w *Wiki) buildSearchRoutes() *wikisearch.Routes {
	return wikisearch.NewRoutes(wikisearch.RoutesConfig{
		Search:            wikisearch.NewSearchUseCase(w.searchIndex, w.tags, w.tree),
		GetIndexingStatus: wikisearch.NewGetIndexingStatusUseCase(w.status),
		AuthService:       w.auth,
	})
}

// buildLinksRoutes 为链接状态查询提供链接服务与树，为失效链接查询只提供链接服务。
func (w *Wiki) buildLinksRoutes() *wikilinks.Routes {
	return wikilinks.NewRoutes(wikilinks.RoutesConfig{
		GetLinkStatus:  wikilinks.NewGetLinkStatusUseCase(w.links, w.tree),
		GetBrokenLinks: wikilinks.NewGetBrokenLinksUseCase(w.links),
		AuthService:    w.auth,
	})
}

// buildTagsRoutes 复用标签索引查询列表，按标签取页面时同时接入树和作者名称解析。
func (w *Wiki) buildTagsRoutes() *wikitags.Routes {
	return wikitags.NewRoutes(wikitags.RoutesConfig{
		GetTags:        wikitags.NewGetTagsUseCase(w.tags),
		GetPagesByTags: wikitags.NewGetPagesByTagsUseCase(w.tags, w.tree, w.userResolver),
		AuthService:    w.auth,
	})
}

// buildPropertiesRoutes 接入属性键查询及按属性取页面的用例，页面结果共用作者解析器。
func (w *Wiki) buildPropertiesRoutes() *wikiproperties.Routes {
	return wikiproperties.NewRoutes(wikiproperties.RoutesConfig{
		GetPropertyKeys:    wikiproperties.NewGetPropertyKeysUseCase(w.props),
		GetPagesByProperty: wikiproperties.NewGetPagesByPropertyUseCase(w.props, w.tree, w.userResolver),
		AuthService:        w.auth,
	})
}

// buildBrandingRoutes 将品牌配置及标识、图标文件的读写统一绑定同一品牌服务。
// 路由同时取得服务供品牌文件响应使用，写权限仍由路由注册时的鉴权规则决定。
func (w *Wiki) buildBrandingRoutes() *wikibranding.Routes {
	return wikibranding.NewRoutes(wikibranding.RoutesConfig{
		GetBranding:     wikibranding.NewGetBrandingUseCase(w.branding),
		UpdateBranding:  wikibranding.NewUpdateBrandingUseCase(w.branding),
		UploadLogo:      wikibranding.NewUploadLogoUseCase(w.branding),
		DeleteLogo:      wikibranding.NewDeleteLogoUseCase(w.branding),
		UploadFavicon:   wikibranding.NewUploadFaviconUseCase(w.branding),
		DeleteFavicon:   wikibranding.NewDeleteFaviconUseCase(w.branding),
		BrandingService: w.branding,
		AuthService:     w.auth,
		Log:             w.log,
	})
}

// buildAvatarRoutes 让上传与删除共用头像服务，账号查询使用动态用户服务入口。
func (w *Wiki) buildAvatarRoutes() *wikiavatar.Routes {
	return wikiavatar.NewRoutes(wikiavatar.RoutesConfig{
		UserService:   w.UserService,
		UploadAvatar:  wikiavatar.NewUploadAvatarUseCase(w.avatar),
		DeleteAvatar:  wikiavatar.NewDeleteAvatarUseCase(w.avatar),
		AvatarService: w.avatar,
		AuthService:   w.auth,
	})
}

// buildAPIKeysRoutes 将创建、列出和撤销用例绑定同一密钥服务。
// 服务未启用时仍构造路由对象，功能是否挂载由对应路由的配置规则处理。
func (w *Wiki) buildAPIKeysRoutes() *wikiapikeys.Routes {
	return wikiapikeys.NewRoutes(wikiapikeys.RoutesConfig{
		CreateAPIKey: wikiapikeys.NewCreateAPIKeyUseCase(w.apiKeys),
		ListAPIKeys:  wikiapikeys.NewListAPIKeysUseCase(w.apiKeys),
		RevokeAPIKey: wikiapikeys.NewRevokeAPIKeyUseCase(w.apiKeys),
		AuthService:  w.auth,
	})
}

// buildImporterRoutes 在存储根目录的 .importer 下保存当前计划及临时工作区。
// 规划器通过适配器访问现有页面与资源，并复用忽略规则缓存；执行由导入服务统一协调。
func (w *Wiki) buildImporterRoutes(options *WikiOptions) *wikiimporter.Routes {
	importerDir := filepath.Join(options.StorageDir, ".importer")
	adapter := NewWikiImportAdapter(w)
	planner := coreimporter.NewPlanner(adapter, w.slug, options.StorageDir)
	planner.SetIgnoreCache(w.ignoreCache)

	store := coreimporter.NewPlanStore(filepath.Join(importerDir, "current-plan.json"))
	svc := coreimporter.NewImporterService(planner, store, filepath.Join(importerDir, "workspaces"), options.MaxAssetUploadSizeBytes)
	return wikiimporter.NewRoutes(wikiimporter.RoutesConfig{
		CreatePlan:  wikiimporter.NewCreateImportPlanUseCase(svc),
		GetPlan:     wikiimporter.NewGetImportPlanUseCase(svc),
		Execute:     wikiimporter.NewExecuteImportUseCase(svc),
		ClearPlan:   wikiimporter.NewClearImportPlanUseCase(svc),
		AuthService: w.auth,
		Svc:         svc,
		Log:         w.log,
	})
}

// Registrars 按声明顺序返回基础领域路由，再追加已配置的快照、恢复与实例设置路由。
// 返回的是当次建立的切片，设置可选路由后需要重新取得该列表才能看到对应对象。
func (w *Wiki) Registrars() []httpinternal.RouteRegistrar {
	registrars := []httpinternal.RouteRegistrar{
		w.authRoutes,
		w.pagesRoutes,
		w.assetsRoutes,
		w.revisionsRoutes,
		w.searchRoutes,
		w.linksRoutes,
		w.tagsRoutes,
		w.propertiesRoutes,
		w.brandingRoutes,
		w.avatarRoutes,

		w.apiKeysRoutes,
		w.importerRoutes,
		w.healthRoutes,
		w.resyncRoutes,
	}
	if w.snapshotRoutes != nil {
		registrars = append(registrars, w.snapshotRoutes)
	}
	if w.restoreRoutes != nil {
		registrars = append(registrars, w.restoreRoutes)
	}
	if w.instanceSettingsRoutes != nil {
		registrars = append(registrars, w.instanceSettingsRoutes)
	}
	return registrars
}

// SetSnapshotRoutes 保存完整快照路由，应在取得注册器并构造 HTTP 路由之前调用。
func (w *Wiki) SetSnapshotRoutes(r *wikisnapshot.Routes) {
	w.snapshotRoutes = r
}

// SetRestoreRoutes 保存在线恢复路由，不自动向已经构造的 HTTP 引擎注册地址。
func (w *Wiki) SetRestoreRoutes(r *wikirestore.Routes) {
	w.restoreRoutes = r
}

// SetInstanceSettingsRoutes 保存运行期设置路由，供下一次取得注册器列表时追加。
func (w *Wiki) SetInstanceSettingsRoutes(r *wikiinstancesettings.Routes) {
	w.instanceSettingsRoutes = r
}

// FrontendConfig 提供存储位置与三个动态品牌回调，供 HTML 首屏渲染使用。
// 各回调分别读取品牌配置，错误或空结果返回空字符串，由 HTTP 层采用自己的回退值。
func (w *Wiki) FrontendConfig() httpinternal.FrontendConfig {
	return httpinternal.FrontendConfig{
		StorageDir: w.storageDir,
		GetSiteName: func() string {
			cfg, err := w.branding.GetBranding()
			if err != nil || cfg == nil {
				return ""
			}
			return cfg.SiteName
		},
		GetLogoFile: func() string {
			cfg, err := w.branding.GetBranding()
			if err != nil || cfg == nil {
				return ""
			}
			return cfg.LogoFile
		},
		GetFaviconFile: func() string {
			cfg, err := w.branding.GetBranding()
			if err != nil || cfg == nil {
				return ""
			}
			return cfg.FaviconFile
		},
	}
}
