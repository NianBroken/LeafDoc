package apikeys

import (
	"net/http"
	"net/http/httptest"
	"testing"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	wikiauth "github.com/NianBroken/LeafDoc/internal/wiki/auth"
	"github.com/gin-gonic/gin"
)

func TestRespondWithAPIKeyError_StoreUnavailable(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	respondWithAPIKeyError(c, sharederrors.NewLocalizedError(
		coreauth.ErrCodeAPIKeyStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"api key store is suspended for an in-progress restore",
		nil,
	))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestRespondWithAPIKeyError_UserStoreUnavailable(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	respondWithAPIKeyError(c, sharederrors.NewLocalizedError(
		wikiauth.ErrCodeAuthUserStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"user store is suspended for an in-progress restore",
		nil,
	))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
