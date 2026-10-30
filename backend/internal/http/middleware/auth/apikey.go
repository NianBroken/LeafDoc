package auth

import (
	"log/slog"
	"net/http"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// APIKeyConfig holds the configuration for Bearer API-key authentication.
type APIKeyConfig struct {
	Service *coreauth.APIKeyService
	// RateLimiter throttles repeated failed Bearer-auth attempts per client
	// IP (a valid key resets its own count — see NotifyResult). Optional;
	// nil disables rate limiting for this path.
	RateLimiter *security.KeyedLimiter
}

// InjectAPIKeyUser reads an "Authorization: Bearer <token>" header carrying a
// LeafDoc API key and, on success, stores the resolved user in the Gin
// context so that RequireAuth can short-circuit JWT/cookie validation — the
// same contract InjectRemoteUser uses for reverse-proxy header auth.
//
// Behaviour by case:
//   - service not configured (feature unused)      → no-op, normal auth applies
//   - user already set upstream                     → no-op, don't override
//   - no Authorization header, or the bearer value
//     isn't shaped like a LeafDoc key               → no-op (cookie/proxy auth still works)
//   - LeafDoc-shaped token, rate limited            → 429
//   - LeafDoc-shaped token, valid                   → user set in context
//   - LeafDoc-shaped token, invalid/revoked/expired → 401
//   - LeafDoc-shaped token, store suspended for a
//     live restore                                   → 503, not counted as a failed attempt
//
// Note on writes: a successfully resolved key sets security.APIKeyAuthContextKey,
// which CSRFMiddleware treats as proof the request is not browser-forgeable and
// skips its cookie check for. Writes are then limited only by the usual role
// guards (role(user) ∩ role(key)). Routes that manage credentials, sessions or
// users must opt out with RequireCookieSession.
func InjectAPIKeyUser(cfg APIKeyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Service == nil {
			c.Next()
			return
		}

		if _, exists := c.Get("user"); exists {
			c.Next()
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" || !coreauth.LooksLikeAPIKeyToken(token) {
			c.Next()
			return
		}

		var limiterKey string
		if cfg.RateLimiter != nil {
			limiterKey = security.ClientKey(c)
			if !cfg.RateLimiter.Allow(limiterKey) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many API key attempts, please try again later"})
				return
			}
		}

		user, err := cfg.Service.Resolve(token)
		if loc, ok := coreauth.AsStoreUnavailableErr(err); ok {
			// An in-progress live restore is an operational condition, not a
			// failed auth attempt. Allow() above already recorded a hit for
			// this request, so undo it the same way a successful request
			// would (NotifyResult(key, true)) rather than leaving it counted
			// — otherwise a sequential, backed-off retry doing exactly what
			// the 503 asks could still trip the limiter. A burst that already
			// exceeded Allow()'s own capacity is unaffected either way, same
			// as it would be for any other error.
			if cfg.RateLimiter != nil {
				cfg.RateLimiter.NotifyResult(limiterKey, true)
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": loc.Message})
			return
		}
		if cfg.RateLimiter != nil {
			cfg.RateLimiter.NotifyResult(limiterKey, err == nil)
		}
		if err != nil {
			slog.Default().Warn("api key auth: rejected", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired API key"})
			return
		}

		c.Set("user", user)
		c.Set(security.APIKeyAuthContextKey, true)
		c.Next()
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
