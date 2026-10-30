package apikeys

import (
	"errors"
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

const (
	ErrCodeAPIKeyInvalidRequest  = "api_key_invalid_request"
	ErrCodeAPIKeyInvalidExpiry   = "api_key_invalid_expiry"
	ErrCodeAPIKeyInvalidRole     = "api_key_invalid_role"
	ErrCodeAPIKeyUserNotFound    = "api_key_user_not_found"
	ErrCodeAPIKeyNotFound        = "api_key_not_found"
	ErrCodeAPIKeyPrefixCollision = "api_key_prefix_collision"
	ErrCodeAPIKeysDisabled       = "api_keys_disabled"
	ErrCodeAPIKeyInternalError   = "api_key_internal_error"
)

func respondWithAPIKeyStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithAPIKeyError is the central error handler for api-key endpoints.
func respondWithAPIKeyError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithAPIKeyStatusError(c, apiKeyErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	var vErr *sharederrors.ValidationErrors
	if errors.As(err, &vErr) {
		httpinternal.RespondValidationErrors(c, vErr)
		return
	}

	switch {
	case errors.Is(err, coreauth.ErrAPIKeyNotFound):
		respondWithAPIKeyStatusError(c, http.StatusNotFound, ErrCodeAPIKeyNotFound, "API key not found", "api key not found")
	case errors.Is(err, coreauth.ErrUserNotFound):
		respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyUserNotFound, "Owning user not found", "owning user not found")
	case errors.Is(err, coreauth.ErrUserInvalidRole):
		respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyInvalidRole, "Invalid role", "invalid role")
	case errors.Is(err, coreauth.ErrAPIKeyPrefixCollision):
		respondWithAPIKeyStatusError(c, http.StatusConflict, ErrCodeAPIKeyPrefixCollision, "Could not generate a unique key, please try again", "could not generate a unique api key, please try again")
	case errors.Is(err, ErrAPIKeysDisabled):
		respondWithAPIKeyStatusError(c, http.StatusForbidden, ErrCodeAPIKeysDisabled, "API key management is disabled", "api key management is disabled")
	default:
		respondWithAPIKeyStatusError(c, http.StatusInternalServerError, ErrCodeAPIKeyInternalError, "API key request failed", "api key request failed")
	}
}

func apiKeyErrorStatus(code string) int {
	switch code {
	case ErrCodeAPIKeyNotFound:
		return http.StatusNotFound
	case ErrCodeAPIKeyInvalidRequest, ErrCodeAPIKeyInvalidExpiry, ErrCodeAPIKeyInvalidRole, ErrCodeAPIKeyUserNotFound:
		return http.StatusBadRequest
	case ErrCodeAPIKeyPrefixCollision:
		return http.StatusConflict
	case ErrCodeAPIKeysDisabled:
		return http.StatusForbidden
	case coreauth.ErrCodeAPIKeyStoreUnavailable, coreauth.ErrCodeAuthUserStoreUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
