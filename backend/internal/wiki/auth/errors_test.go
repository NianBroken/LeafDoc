package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// TestRespondWithAuthError_UserStoreUnavailable 确认恢复暂停的用户存储返回 503。
func TestRespondWithAuthError_UserStoreUnavailable(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	respondWithAuthError(c, sharederrors.NewLocalizedError(
		ErrCodeAuthUserStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"user store is suspended for an in-progress restore",
		nil,
	))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestRespondWithAuthError_EditorLimitReached 核对编辑者配额错误的 403 与完整翻译字段。
func TestRespondWithAuthError_EditorLimitReached(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	respondWithAuthError(c, coreauth.ErrEditorLimitReached)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got, want := rec.Body.String(), `{"error":{"code":"auth_editor_limit_reached","message":"Editor limit reached for this plan","template":"editor limit reached for this plan"}}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}
