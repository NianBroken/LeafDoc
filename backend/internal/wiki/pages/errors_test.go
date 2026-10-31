package pages

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// TestRespondWithPageError_FavoritesStoreUnavailable_Returns503
// 构造带收藏存储不可用代码的本地化错误，通过真实页面错误响应入口检查 503 状态。
// 断言只验证状态映射，不触发备份恢复或实际暂停收藏存储。
func TestRespondWithPageError_FavoritesStoreUnavailable_Returns503(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	respondWithPageError(c, sharederrors.NewLocalizedError(
		favorites.ErrCodeFavoritesStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"favorites store is suspended for an in-progress restore",
		nil,
	))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
