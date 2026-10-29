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
	"github.com/NianBroken/LeafDoc/internal/core/restore"
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
	searchIndex       *search.SQLiteIndex
	status            *search.IndexingStatus
	storageDir        string

	// Domain route registrars (populated by NewWiki).
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

const SYSTEM_USER_ID = "system"

type WikiOptions struct {
	StorageDir              string        // Path to storage directory
	AdminUsername           string        // Initial admin username (optional; defaults to "admin")
	AdminEmail              string        // Initial admin email (optional; defaults to "admin@localhost")
	AdminPassword           string        // Initial admin password
	JWTSecret               string        // JWT secret for authentication
	AccessTokenTimeout      time.Duration // Access token timeout duration
	RefreshTokenTimeout     time.Duration // Refresh token timeout duration
	AuthDisabled            bool          // Whether authentication is disabled
	EnableRevision          bool          // Whether revision recording/storage is enabled
	EnableAPIKeyManagement  bool          // Whether the experimental API key management feature is enabled
	MaxRevisionHistory      int           // Max revisions kept per page; 0 = unlimited
	EditorLimit             int           // Max admin+editor users allowed; 0 = unlimited
	MaxAssetUploadSizeBytes int64         // Maximum allowed size in bytes for asset/import uploads; 0 = default
	RevisionCoalesceWindow  time.Duration // Window for coalescing rapid successive saves; 0 = disabled
	SMTP                    email.Config  // SMTP config for password-reset/invite email; SMTP.Enabled()==false disables the feature entirely
	Metrics                 *httpmetrics.HTTPMetrics
}

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
		// Close what the steps before the failing one opened: a leaked
		// database handle keeps the file locked on Windows.
		if closeErr := w.Close(); closeErr != nil {
			w.log.Error("error closing partially initialized wiki", "error", closeErr)
		}
		return nil, err
	}
	return w, nil
}

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
	// Default documents exist before the revision service records their baseline.
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

// AuthService returns the authentication service.
func (w *Wiki) AuthService() *auth.AuthService {
	return w.auth
}

// BrandingService returns the branding service.
func (w *Wiki) BrandingService() *branding.BrandingService {
	return w.branding
}

// ─── Service getters (test infrastructure) ───────────────────────────────────

func (w *Wiki) GetStorageDir() string {
	return w.storageDir
}

// UserService returns the current *auth.UserService: AuthService's live,
// restore-swap-tracking one when auth is enabled (see AuthService.UserService),
// falling back to the UserService constructed in initAuth when auth is
// disabled — in that mode there's no AuthService and, per
// internal/core/restore/manager.go's AuthService-nil guards, a live restore never
// swaps this store either, so the fallback is never stale.
func (w *Wiki) UserService() *auth.UserService {
	if w.auth != nil {
		return w.auth.UserService()
	}
	return w.user
}

// UserResolver returns the author-label resolver shared across page/tag/
// property routes and the user-management use cases.
func (w *Wiki) UserResolver() *auth.UserResolver {
	return w.userResolver
}

// APIKeyService returns the API key service used for Bearer authentication.
func (w *Wiki) APIKeyService() *auth.APIKeyService {
	return w.apiKeys
}

// Favorites returns the favorites store, e.g. for wiring into restore.Config.
func (w *Wiki) Favorites() *favorites.FavoritesStore {
	return w.favorites
}

// StateStores returns the state stores (other than users.db) that exist in
// this process, for restore.Config.Stores. A store whose feature is off is
// not listed; a nil pointer must never be wrapped in the interface.
func (w *Wiki) StateStores() []restore.LiveStore {
	var stores []restore.LiveStore
	if w.apiKeys != nil {
		stores = append(stores, restore.LiveStore{File: restore.APIKeysDBFile, Store: w.apiKeys})
	}
	if w.favorites != nil {
		stores = append(stores, restore.LiveStore{File: restore.FavoritesDBFile, Store: w.favorites})
	}
	return stores
}

// EmailTokenService returns the password-reset/invite token service, or nil
// if SMTP isn't configured or auth is disabled (see initEmail). Every use
// case that resolves this via the func()-based pattern (like UserService)
// nil-checks it and returns ErrEmailDisabled.
func (w *Wiki) EmailTokenService() *auth.EmailTokenService {
	return w.emailTokenService
}

func (w *Wiki) Close() error {
	w.shutdownCancel() // signal in-flight reloads to abort
	w.reloadWG.Wait()  // drain goroutines before closing stores
	if w.status != nil {
		w.status.Finish()
	}

	// Close every store even when an earlier one fails, so no handle is
	// leaked. Fields stay nil when NewWiki failed before reaching them.
	var errs []error
	if w.auth != nil {
		// When auth is enabled, AuthService owns both the session store and user store.
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
		w.emailTokenService.Close() // drains in-flight fire-and-forget password-reset sends
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
