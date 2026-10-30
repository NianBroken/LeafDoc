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

// ─── Subsystem initializers ───────────────────────────────────────────────────

func (w *Wiki) initAuth(options *WikiOptions) error {
	// Database connection and schema initialization complete before admin creation.
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

	// API key authentication is available only when the feature is enabled.
	if options.EnableAPIKeyManagement {
		apiKeyStore, err := auth.NewAPIKeyStore(w.storageDir)
		if err != nil {
			return err
		}
		w.apiKeys = auth.NewAPIKeyService(apiKeyStore, w.auth)
	}
	return nil
}

// initEmail constructs password reset and invitation services when SMTP is configured.
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

func (w *Wiki) initCoreServices(options *WikiOptions) error {
	// Create a shared ignore cache for multi-level .leafdocignore resolution.
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

func (w *Wiki) initTagsService() error {
	tagsStore, err := tags.NewTagsStore(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init tags store: %w", err)
	}
	w.tags = tags.NewTagsService(tagsStore)
	return nil
}

func (w *Wiki) initPropertiesService() error {
	propsStore, err := properties.NewPropertiesStore(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init properties store: %w", err)
	}
	w.props = properties.NewPropertiesService(propsStore)
	return nil
}

func (w *Wiki) initFavoritesService() error {
	store, err := favorites.NewFavoritesStore(w.storageDir, w.log)
	if err != nil {
		return fmt.Errorf("failed to init favorites store: %w", err)
	}
	w.favorites = store
	return nil
}

// bootstrapTagsAndProperties clears and rebuilds tag and property indexes in a single
// parallel GetPages pass — avoids two sequential ReadPageRaw loops at startup.
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

func (w *Wiki) initSearch() error {
	var err error
	w.searchIndex, err = search.NewMySQLIndex(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init search index: %w", err)
	}
	w.status = search.NewIndexingStatus()
	searchEffect := pagesave.NewSearchIndexSideEffect(w.searchIndex, w.tree, w.log, w.metrics)
	w.log.Info("search indexing started")
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

func (w *Wiki) initBranding() error {
	var err error
	w.branding, err = branding.NewBrandingService(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init branding service: %w", err)
	}
	return nil
}

func (w *Wiki) initAvatarService() error {
	var err error
	w.avatar, err = avatar.NewAvatarService(w.storageDir)
	if err != nil {
		return fmt.Errorf("failed to init avatar service: %w", err)
	}
	return nil
}

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
