package search

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRespondWithSearchError_ServiceUnavailable 精确检查服务缺失的 503 状态与完整结构化错误体。
func TestRespondWithSearchError_ServiceUnavailable(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	respondWithSearchError(c, ErrSearchUnavailable)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	if got, want := rec.Body.String(), `{"error":{"code":"search_unavailable","message":"Search is currently unavailable","template":"search is currently unavailable"}}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

// TestRespondWithSearchError_InternalErrorIsSanitized 注入数据库错误，确认响应只保留固定对外消息。
func TestRespondWithSearchError_InternalErrorIsSanitized(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	respondWithSearchError(c, errors.New("MySQL connection lost"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if got, want := rec.Body.String(), `{"error":{"code":"search_internal_error","message":"Failed to perform search","template":"failed to perform search"}}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}
