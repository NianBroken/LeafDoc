// Package instancesettings exposes administrator-managed runtime preferences,
// anonymous access and table-of-contents defaults through authenticated routes.
package instancesettings

import (
	"log/slog"
	"net/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tocdisplay"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes is the RouteRegistrar for runtime instance settings.
type Routes struct {
	publicAccess *publicaccess.Service
	tocDisplay   *tocdisplay.Service
	authService  *coreauth.AuthService
	log          *slog.Logger
}

// NewRoutes constructs the instance-settings RouteRegistrar.
func NewRoutes(publicAccess *publicaccess.Service, tocDisplay *tocdisplay.Service, authService *coreauth.AuthService, log *slog.Logger) *Routes {
	if log == nil {
		log = slog.Default()
	}
	return &Routes{publicAccess: publicAccess, tocDisplay: tocDisplay, authService: authService, log: log}
}

// RegisterRoutes implements RouteRegistrar.
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	adminGroup := ctx.APIAuthGroup(r.authService).Group("/admin/settings")
	if ctx.Opts.Preferences != nil {
		adminGroup.GET("/preferences", authmw.RequireAdmin(), func(c *gin.Context) { c.JSON(200, ctx.Opts.Preferences.Get()) })
		adminGroup.PUT("/preferences", authmw.RequireAdmin(), authmw.RequireCookieSession(), func(c *gin.Context) {
			var patch map[string]any
			if err := c.ShouldBindJSON(&patch); err != nil {
				c.JSON(400, gin.H{"error": localization.Message("invalid payload")})
				return
			}
			value, err := ctx.Opts.Preferences.Patch(patch)
			if err != nil {
				if loc, ok := sharederrors.AsLocalizedError(err); ok {
					httpinternal.RespondError(c, http.StatusBadRequest, loc.Code, loc.Message, loc.Template, loc.Args...)
				} else {
					r.log.Error("failed to save preferences", "error", err)
					httpinternal.RespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to save preferences", "failed to save preferences")
				}
				return
			}
			c.JSON(200, value)
		})
	}
	// Flipping instance-wide anonymous access is a browser-session-only
	// action, like API-key management — an API key must not be able to do it
	// (CSRF no longer blocks key-authenticated writes, so RequireCookieSession
	// is what enforces this).
	adminGroup.PUT("/public-access",
		authmw.RequireAdmin(),
		authmw.RequireCookieSession(),
		r.handleSetPublicAccess,
	)
	adminGroup.PUT("/toc-display",
		authmw.RequireAdmin(),
		r.handleSetTocDisplay,
	)
}

// ─── Handlers ───────────────────────────────────────────────────────────────

func (r *Routes) handleSetPublicAccess(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")
		return
	}

	if err := r.publicAccess.SetEnabled(*req.Enabled); err != nil {
		// Env-managed instances land here with publicaccess.ErrCodeEnvManaged
		// → 409, so the frontend knows to render its status-only view.
		respondWithError(c, err)
		return
	}

	r.log.Info("public access mode changed",
		"enabled", r.publicAccess.Enabled(),
		"actor_id", actorID(c),
	)
	c.JSON(http.StatusOK, gin.H{"enabled": r.publicAccess.Enabled()})
}

func (r *Routes) handleSetTocDisplay(c *gin.Context) {
	var req struct {
		AlwaysShow *bool  `json:"alwaysShow"`
		Mode       string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.AlwaysShow == nil && req.Mode == "") {
		respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "auto"
		if *req.AlwaysShow {
			mode = "always"
		}
	}
	if err := r.tocDisplay.SetMode(mode); err != nil {
		respondWithError(c, err)
		return
	}

	r.log.Info("toc always-show setting changed",
		"always_show", r.tocDisplay.AlwaysShow(),
		"actor_id", actorID(c),
	)
	c.JSON(http.StatusOK, gin.H{"alwaysShow": r.tocDisplay.AlwaysShow(), "mode": r.tocDisplay.Mode()})
}

func actorID(c *gin.Context) string {
	if u := authmw.TryGetUser(c); u != nil {
		return u.ID
	}
	return ""
}
