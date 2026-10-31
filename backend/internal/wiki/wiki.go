package wiki

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/avatar"
	"github.com/NianBroken/LeafDoc/internal/core/branding"
	"github.com/NianBroken/LeafDoc/internal/core/email"
	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	"github.com/NianBroken/LeafDoc/internal/core/ignore"
	"github.com/NianBroken/LeafDoc/internal/core/links"
	"github.com/NianBroken/LeafDoc/internal/core/properties"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
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
	wikiproperties "github.com/NianBroken/LeafDoc/internal/wiki/properties"
	wikirestore "github.com/NianBroken/LeafDoc/internal/wiki/restore"
	wikiresync "github.com/NianBroken/LeafDoc/internal/wiki/resync"
	wikirevisions "github.com/NianBroken/LeafDoc/internal/wiki/revisions"
	wikisearch "github.com/NianBroken/LeafDoc/internal/wiki/search"
	wikisnapshot "github.com/NianBroken/LeafDoc/internal/wiki/snapshot"
	wikitags "github.com/NianBroken/LeafDoc/internal/wiki/tags"
)

// Wiki 汇集当前实例的核心服务、业务路由及后台重建任务生命周期。
// 构造过程按依赖顺序初始化服务，路由层通过这些共享实例读取或修改内容。
// reloadMu 串行化显式重建流程，reloadWG 跟踪后台任务，关闭上下文通知支持取消的阶段退出。
type Wiki struct {
	tree              *tree.TreeService
	slug              *tree.SlugService
	auth              *auth.AuthService
	userResolver      *auth.UserResolver
	user              *auth.UserService
	apiKeys           *auth.APIKeyService
	emailTokenStore   *auth.EmailTokenStore
	emailTokenService *auth.EmailTokenService
	asset             *assets.AssetService
	branding          *branding.BrandingService
	avatar            *avatar.AvatarService
	searchIndex       *search.MySQLIndex
	status            *search.IndexingStatus
	storageDir        string

	// 领域路由由启动阶段装配；快照、恢复和实例设置路由另由外部入口注入。
	pagesRoutes            *wikipages.Routes
	authRoutes             *wikiauth.Routes
	assetsRoutes           *wikiassets.Routes
	revisionsRoutes        *wikirevisions.Routes
	searchRoutes           *wikisearch.Routes
	linksRoutes            *wikilinks.Routes
	tagsRoutes             *wikitags.Routes
	propertiesRoutes       *wikiproperties.Routes
	brandingRoutes         *wikibranding.Routes
	avatarRoutes           *wikiavatar.Routes
	apiKeysRoutes          *wikiapikeys.Routes
	importerRoutes         *wikiimporter.Routes
	healthRoutes           *wikihealth.Routes
	revision               *revision.Service
	links                  *links.LinkService
	tags                   *tags.TagsService
	props                  *properties.PropertiesService
	favorites              *favorites.FavoritesStore
	snapshotRoutes         *wikisnapshot.Routes
	restoreRoutes          *wikirestore.Routes
	instanceSettingsRoutes *wikiinstancesettings.Routes
	resyncRoutes           *wikiresync.Routes
	resyncJob              *wikiresync.ResyncJob
	ignoreCache            *ignore.Cache
	reloadMu               sync.Mutex
	reloadWG               sync.WaitGroup
	shutdownCtx            context.Context
	shutdownCancel         context.CancelFunc
	log                    *slog.Logger
	metrics                *httpmetrics.HTTPMetrics
}

// SYSTEM_USER_ID 是系统建立基线历史时记录的作者标识，不表示当前登录用户。
const SYSTEM_USER_ID = "system"

// WikiOptions 提供核心服务构造参数，不在该结构体内读取环境或设置默认值。
// 各服务分别执行自己的初始化及约束检查，已有管理员不因构造参数而直接重置。
type WikiOptions struct {
	StorageDir              string        // 存储根目录，供各服务定位数据及资源。
	AdminUsername           string        // 初始化管理员时使用的用户名。
	AdminEmail              string        // 初始化管理员时使用的邮箱。
	AdminPassword           string        // 初始化管理员时使用的密码，不在这里写入日志。
	JWTSecret               string        // 会话管理器签发与验证令牌所用的密钥。
	AccessTokenTimeout      time.Duration // 访问令牌有效期。
	RefreshTokenTimeout     time.Duration // 刷新令牌有效期。
	EnableRevision          bool          // 是否构造页面历史服务并建立基线。
	EnableAPIKeyManagement  bool          // 是否构造 API 密钥存储与鉴权服务。
	MaxRevisionHistory      int           // 交给历史服务的每页保留数量限制。
	EditorLimit             int           // 交给用户服务的管理员及编辑者数量限制。
	MaxAssetUploadSizeBytes int64         // 交给导入服务的资源及导入上传字节限制。
	RevisionCoalesceWindow  time.Duration // 交给历史服务的连续保存合并窗口。
	SMTP                    email.Config  // 邮件配置未启用时不构造邮件令牌存储与服务。
	Metrics                 *httpmetrics.HTTPMetrics
}

// NewWiki 建立关闭上下文并执行启动流程，成功后返回可供路由装配的实例。
// options 必须非空；任一步失败时调用 Close 清理已经初始化的资源，并返回原始启动错误。
func NewWiki(options *WikiOptions) (*Wiki, error) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	w := &Wiki{
		storageDir:     options.StorageDir,
		log:            slog.Default().With("component", "Wiki"),
		resyncJob:      wikiresync.NewResyncJob(),
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
		metrics:        options.Metrics,
	}
	if err := w.boot(options); err != nil {
		// 启动中断时仍需关闭前面步骤取得的存储句柄，清理失败单独记录而不替换启动错误。
		if closeErr := w.Close(); closeErr != nil {
			w.log.Error("error closing partially initialized wiki", "error", closeErr)
		}
		return nil, err
	}
	return w, nil
}

// boot 先建立账号与内容服务，再构建派生索引、品牌资源、默认文档与可选历史。
// 同步初始化错误立即返回，服务内部明确记录后继续的索引错误不在这里转换成失败。
// 搜索初始索引在后台运行，boot 返回不表示该索引已经完成。
func (w *Wiki) boot(options *WikiOptions) error {
	if err := w.initAuth(options); err != nil {
		return err
	}
	if err := w.initEmail(options); err != nil {
		return err
	}
	if err := w.initCoreServices(options); err != nil {
		return err
	}
	if err := w.initLinkService(); err != nil {
		return err
	}
	if err := w.initTagsService(); err != nil {
		return err
	}
	if err := w.initPropertiesService(); err != nil {
		return err
	}
	if err := w.initFavoritesService(); err != nil {
		return err
	}
	w.bootstrapTagsAndProperties()
	if err := w.initSearch(); err != nil {
		return err
	}
	if err := w.initBranding(); err != nil {
		return err
	}
	if err := w.initAvatarService(); err != nil {
		return err
	}
	// 默认文档先落入树中，再建立历史服务并为当前页面记录基线。
	if err := w.EnsureDefaultDocuments(); err != nil {
		return err
	}
	if options.EnableRevision {
		w.revision = revision.NewService(w.storageDir, w.tree, w.log,
			revision.ServiceOptions{
				MaxRevisions:   options.MaxRevisionHistory,
				CoalesceWindow: options.RevisionCoalesceWindow,
			})
		w.ensureBaselineRevisions()
	}
	w.metrics.RegisterRuntimeStats(runtimeStatsSource{w: w})
	w.buildRoutes(options)
	return nil
}

// AuthService 返回当前鉴权服务，供路由和恢复配置复用同一会话入口。
func (w *Wiki) AuthService() *auth.AuthService {
	return w.auth
}

// BrandingService 返回当前品牌服务，供首屏配置及品牌设置路由使用。
func (w *Wiki) BrandingService() *branding.BrandingService {
	return w.branding
}

// GetStorageDir 返回构造时保存的存储根目录，不进行清理或绝对路径转换。
func (w *Wiki) GetStorageDir() string {
	return w.storageDir
}

// UserService 优先从鉴权服务获取当前用户服务，跟随恢复过程替换后的用户存储。
// 鉴权服务尚未建立时返回初始化阶段保存的用户服务，供启动期解析器使用。
func (w *Wiki) UserService() *auth.UserService {
	if w.auth != nil {
		return w.auth.UserService()
	}
	return w.user
}

// UserResolver 返回各页面、标签、属性及账号用例共享的作者名称解析器。
func (w *Wiki) UserResolver() *auth.UserResolver {
	return w.userResolver
}

// APIKeyService 返回 Bearer 密钥服务，未启用相应功能时为空。
func (w *Wiki) APIKeyService() *auth.APIKeyService {
	return w.apiKeys
}

// Favorites 返回收藏存储，供页面用例与恢复配置引用同一存储实例。
func (w *Wiki) Favorites() *favorites.FavoritesStore {
	return w.favorites
}

// EmailTokenService 返回邮件令牌服务，邮件功能未启用时为空。
func (w *Wiki) EmailTokenService() *auth.EmailTokenService {
	return w.emailTokenService
}

// Close 通知后台重建退出并等待其结束，然后逐项关闭当前持有的存储。
// 某个关闭操作失败不妨碍其余存储继续关闭，最终通过 errors.Join 汇总全部关闭错误。
// 实例必须具备构造阶段建立的关闭函数，不支持未初始化的 Wiki 零值。
func (w *Wiki) Close() error {
	w.shutdownCancel() // 通知使用关闭上下文的索引或重建阶段取消。
	w.reloadWG.Wait()  // 后台任务结束后再关闭其使用的存储。
	if w.status != nil {
		w.status.Finish()
	}

	// 初始化尚未到达的字段保持为空，按字段检查使失败启动也可使用同一清理路径。
	var errs []error
	if w.auth != nil {
		// 鉴权服务负责关闭会话与用户存储，避免再通过 w.user 重复关闭。
		if err := w.auth.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close auth: %w", err))
		}
	} else if w.user != nil {
		if err := w.user.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close user store: %w", err))
		}
	}

	if w.apiKeys != nil {
		if err := w.apiKeys.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close api keys: %w", err))
		}
	}

	if w.emailTokenService != nil {
		w.emailTokenService.Close() // 等待异步密码重置邮件发送结束，再关闭邮件令牌存储。
	}
	if w.emailTokenStore != nil {
		if err := w.emailTokenStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close email token store: %w", err))
		}
	}

	if w.links != nil {
		if err := w.links.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close links: %w", err))
		}
	}

	if w.tags != nil {
		if err := w.tags.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close tags: %w", err))
		}
	}

	if w.props != nil {
		if err := w.props.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close properties: %w", err))
		}
	}

	if w.favorites != nil {
		if err := w.favorites.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close favorites store: %w", err))
		}
	}

	if w.searchIndex != nil {
		if err := w.searchIndex.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close search index: %w", err))
		}
	}

	return errors.Join(errs...)
}

// WaitForStartup 等待已登记的初始索引完成，供安装校验在关闭服务前确认搜索可用。
func (w *Wiki) WaitForStartup() error {
	w.reloadWG.Wait()
	if w.status != nil && w.status.IsFailed() {
		return fmt.Errorf("搜索初始化失败")
	}
	return nil
}
