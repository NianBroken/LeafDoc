// White-box test: package wikirestore so we can register the real handler methods.
package wikirestore

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/testutil"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/branding"
	"github.com/NianBroken/LeafDoc/internal/core/restore"
	"github.com/NianBroken/LeafDoc/internal/core/settings"
	snapshotSvc "github.com/NianBroken/LeafDoc/internal/core/snapshot"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "wiki/restore") }

// newTestManager wires a real restore.Manager (real snapshot manager, real
// AuthService, real BrandingService) against fresh temp dirs — the
// Manager-level behavior itself is covered exhaustively in
// internal/core/restore's own tests; this package only needs to verify the HTTP
// wiring (routing, status codes, response shapes) on top of it.
func newTestManager(t *testing.T) *restore.Manager {
	return newTestManagerWithMaxUploadSize(t, 0)
}

// newTestManagerWithMaxUploadSize is newTestManager with control over
// Config.MaxUploadSizeBytes, used to exercise the 413-too-large path with a
// small configured limit instead of the real (500 MiB) default.
func newTestManagerWithMaxUploadSize(t *testing.T, maxUploadSizeBytes int64) *restore.Manager {
	t.Helper()
	base := t.TempDir()

	pages := tree.NewTreeService(base)
	if err := pages.LoadTree(); err != nil {
		t.Fatal(err)
	}
	if _, err := pages.CreateNode("system", nil, "Hello", "hello", nil); err != nil {
		t.Fatal(err)
	}

	userStore, err := coreauth.NewUserStore(base)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}
	t.Cleanup(func() {
		if err := userStore.Close(); err != nil {
			t.Error(err)
		}
	})
	sessionStore, err := coreauth.NewSessionStore(base)
	if err != nil {
		t.Fatalf("NewSessionStore failed: %v", err)
	}
	t.Cleanup(func() {
		if err := sessionStore.Close(); err != nil {
			t.Error(err)
		}
	})
	sessions := coreauth.NewSessionManager(sessionStore, "test-secret-key-for-unit-tests-1", time.Hour, 24*time.Hour)
	authService := coreauth.NewAuthService(coreauth.NewUserService(userStore), sessions)
	t.Cleanup(func() { _ = authService.Close() })

	brandingService, err := branding.NewBrandingService(base)
	if err != nil {
		t.Fatalf("NewBrandingService failed: %v", err)
	}

	snapshotManager := snapshotSvc.NewManager(snapshotSvc.Config{
		BackupsDir:  filepath.Join(base, "backups"),
		DataDir:     base,
		WikiVersion: "v0.0.0-test",
	})

	return restore.NewManager(restore.Config{
		SnapshotManager:    snapshotManager,
		DataDir:            base,
		WikiVersion:        "v0.0.0-test",
		WriteGate:          restore.NewWriteGate(),
		AuthService:        authService,
		Reloadables:        []settings.Reloadable{brandingService},
		TriggerResync:      func() {},
		MaxUploadSizeBytes: maxUploadSizeBytes,
	})
}

func newTestRouter(routes *Routes) *gin.Engine {
	r := gin.New()
	r.POST("/restore/upload", routes.handleTriggerUpload)
	r.POST("/restore/:id", routes.handleTrigger)
	r.GET("/restore/status", routes.handleStatus)
	r.GET("/api/restore/progress", routes.handleProgress)
	r.POST("/restore/self-restart", routes.handleSelfRestart)
	return r
}

// buildFixtureZipPath exports real MySQL records and resource checksums into
// the same versioned ZIP format accepted by the HTTP restore endpoint.
func buildFixtureZipPath(t *testing.T) string {
	t.Helper()
	src := t.TempDir()

	pages := tree.NewTreeService(src)
	if err := pages.LoadTree(); err != nil {
		t.Fatal(err)
	}
	if _, err := pages.CreateNode("system", nil, "Uploaded backup content", "welcome", nil); err != nil {
		t.Fatal(err)
	}

	userStore, err := coreauth.NewUserStore(src)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}
	t.Cleanup(func() {
		if err := userStore.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := coreauth.NewUserService(userStore).CreateUser("upload-admin", "upload-admin@example.com", "upload-password-123", "admin"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if err := userStore.Close(); err != nil {
		t.Fatalf("failed to close user store: %v", err)
	}

	snapshotManager := snapshotSvc.NewManager(snapshotSvc.Config{
		BackupsDir:  filepath.Join(src, "backups"),
		DataDir:     src,
		WikiVersion: "v0.0.0-test",
	})
	if err := snapshotManager.RunOnce(context.Background()); err != nil {
		t.Fatalf("failed to build fixture snapshot: %v", err)
	}
	entries, err := snapshotManager.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 fixture snapshot, got %v (err=%v)", entries, err)
	}
	zipPath, err := snapshotManager.SnapshotZipPath(entries[0].ID)
	if err != nil {
		t.Fatalf("SnapshotZipPath failed: %v", err)
	}
	return zipPath
}

// newMultipartUploadRequest builds a POST request whose body is a multipart
// form carrying zipPath's contents under the "file" field, mirroring what the
// frontend's FormData-based upload sends.
func newMultipartUploadRequest(t *testing.T, url, zipPath string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filepath.Base(zipPath))
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	src, err := os.Open(zipPath)
	if err != nil {
		t.Fatalf("failed to open fixture zip: %v", err)
	}
	defer func() { _ = src.Close() }()
	if _, err := io.Copy(fw, src); err != nil {
		t.Fatalf("failed to copy fixture zip into multipart body: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestHandleTrigger_NotEnabledWhenManagerNil(t *testing.T) {
	testutil.ResetMySQL(t)
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/restore/snapshot-20260101-000000", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatus_NotEnabledWhenManagerNil(t *testing.T) {
	testutil.ResetMySQL(t)
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/restore/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleTrigger_UnknownSnapshotID_AsyncJobReportsError(t *testing.T) {
	testutil.ResetMySQL(t)
	m := newTestManager(t)
	routes := &Routes{manager: m}
	router := newTestRouter(routes)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/restore/snapshot-does-not-exist", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 (validation runs async), got %d: %s", w.Code, w.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status().Done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if m.Status().Error == "" {
		t.Error("expected the async job to report an error for an unknown snapshot id")
	}
}

func TestHandleStatus_ReturnsJobStatus(t *testing.T) {
	testutil.ResetMySQL(t)
	m := newTestManager(t)
	routes := &Routes{manager: m}
	router := newTestRouter(routes)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/restore/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var status restore.JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if status.Running {
		t.Error("expected a fresh manager to report running=false")
	}
}

func TestHandleSelfRestart_NotEnabledWhenManagerNil(t *testing.T) {
	testutil.ResetMySQL(t)
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/restore/self-restart", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleTriggerUpload_NotEnabledWhenManagerNil(t *testing.T) {
	testutil.ResetMySQL(t)
	router := newTestRouter(&Routes{})

	zipPath := buildFixtureZipPath(t)
	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/restore/upload", zipPath)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleTriggerUpload_HappyPath_AsyncJobCompletes(t *testing.T) {
	testutil.ResetMySQL(t)
	m := newTestManager(t)
	routes := &Routes{manager: m}
	router := newTestRouter(routes)

	zipPath := buildFixtureZipPath(t)
	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/restore/upload", zipPath)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var accepted struct {
		StatusToken string `json:"statusToken"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if token, err := hex.DecodeString(accepted.StatusToken); err != nil || len(token) != 32 {
		t.Fatal("upload response omitted valid progress credential")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status().Done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	status := m.Status()
	if status.Error != "" {
		t.Fatalf("expected successful restore, got error: %s", status.Error)
	}
	progressRequest := httptest.NewRequest(http.MethodGet, "/api/restore/progress", nil)
	progressRequest.Header.Set("X-Restore-Token", accepted.StatusToken)
	progressResponse := httptest.NewRecorder()
	router.ServeHTTP(progressResponse, progressRequest)
	if progressResponse.Code != http.StatusOK {
		t.Fatalf("upload progress unavailable: %d", progressResponse.Code)
	}

}

func TestHandleTriggerUpload_ExceedsConfiguredMaxSize(t *testing.T) {
	testutil.ResetMySQL(t)
	m := newTestManagerWithMaxUploadSize(t, 8) // smaller than any real fixture zip
	routes := &Routes{manager: m}
	router := newTestRouter(routes)

	zipPath := buildFixtureZipPath(t)
	w := httptest.NewRecorder()
	req := newMultipartUploadRequest(t, "/restore/upload", zipPath)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", w.Code, w.Body.String())
	}
	var resp RestoreErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Error.Code != ErrCodeRestoreUploadTooLarge {
		t.Errorf("expected code %q, got %q", ErrCodeRestoreUploadTooLarge, resp.Error.Code)
	}
}

// TestHandleTriggerUpload_MalformedMultipartBody_DistinctFromTooLarge is the
// regression test for a review finding: every ParseMultipartForm error used
// to be reported as ErrCodeRestoreUploadTooLarge (413), even when the body
// wasn't too large at all — e.g. malformed multipart framing from a broken
// client or a mangled proxy. That masked the real cause. A malformed-but-not-
// oversized body should surface as ErrCodeRestoreUploadInvalid (400) instead.
func TestHandleTriggerUpload_MalformedMultipartBody_DistinctFromTooLarge(t *testing.T) {
	testutil.ResetMySQL(t)
	m := newTestManager(t)
	routes := &Routes{manager: m}
	router := newTestRouter(routes)

	req, _ := http.NewRequest(http.MethodPost, "/restore/upload", bytes.NewBufferString("not a multipart body"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=broken")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp RestoreErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Error.Code != ErrCodeRestoreUploadInvalid {
		t.Errorf("expected code %q, got %q", ErrCodeRestoreUploadInvalid, resp.Error.Code)
	}
}

func TestHandleTriggerUpload_MissingFile(t *testing.T) {
	testutil.ResetMySQL(t)
	m := newTestManager(t)
	routes := &Routes{manager: m}
	router := newTestRouter(routes)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.Close(); err != nil {
		t.Fatalf("failed to close empty multipart writer: %v", err)
	}
	req, _ := http.NewRequest(http.MethodPost, "/restore/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp RestoreErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Error.Code != ErrCodeRestoreMissingFile {
		t.Errorf("expected code %q, got %q", ErrCodeRestoreMissingFile, resp.Error.Code)
	}
}

// TestHandleSelfRestart_RejectedWithoutNeedsIntervention is the only
// self-restart HTTP test: it deliberately never reaches restore.SelfRestart()
// (which would syscall.Exec / os.Exit the *test process itself*). A fresh
// Manager never has NeedsIntervention set, so this only exercises the guard.
func TestHandleSelfRestart_RejectedWithoutNeedsIntervention(t *testing.T) {
	testutil.ResetMySQL(t)
	m := newTestManager(t)
	routes := &Routes{manager: m}
	router := newTestRouter(routes)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/restore/self-restart", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 when the job hasn't reported NeedsIntervention, got %d", w.Code)
	}
}

// TestProgressCredentialRoute separates the narrow bearer capability from
// authenticated administrator status and accepts credentials only in headers.
func TestProgressCredentialRoute(t *testing.T) {
	testutil.ResetMySQL(t)
	manager := newTestManager(t)
	routes := &Routes{manager: manager}
	triggerRouter := newTestRouter(routes)
	registered := gin.New()
	routes.RegisterRoutes(httpinternal.RouterContext{Engine: registered, Base: registered, AuthCookies: authmw.NewAuthCookies(true, time.Hour, time.Hour), CSRFCookie: security.NewCSRFCookie(true, time.Hour), Opts: httpinternal.RouterOptions{AllowInsecure: true}})
	request := func(path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			req.Header.Set("X-Restore-Token", token)
		}
		response := httptest.NewRecorder()
		registered.ServeHTTP(response, req)
		return response
	}
	if response := request("/api/restore/progress", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous idle progress=%d", response.Code)
	}
	trigger := func() string {
		response := httptest.NewRecorder()
		triggerRouter.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/restore/not-found", nil))
		if response.Code != http.StatusAccepted {
			t.Fatalf("trigger=%d %s", response.Code, response.Body.String())
		}
		var body struct {
			StatusToken string `json:"statusToken"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		decoded, err := hex.DecodeString(body.StatusToken)
		if err != nil || len(decoded) != 32 {
			t.Fatal("invalid 256-bit credential")
		}
		manager.Wait()
		return body.StatusToken
	}
	first := trigger()
	for _, item := range []struct{ path, token string }{{"/api/restore/progress", "wrong"}, {"/api/restore/progress?statusToken=" + first, ""}, {"/api/restore/progress?token=" + first, ""}, {"/api/admin/restore/status", first}} {
		if response := request(item.path, item.token); response.Code != http.StatusUnauthorized {
			t.Fatalf("unauthorized route accepted: %s %d", item.path, response.Code)
		}
	}
	response := request("/api/restore/progress", first)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("valid credential=%d cache=%s", response.Code, response.Header().Get("Cache-Control"))
	}
	if strings.Contains(response.Body.String(), first) {
		t.Fatal("credential leaked into status body")
	}
	var state restore.JobStatus
	if err := json.Unmarshal(response.Body.Bytes(), &state); err != nil || !state.Done {
		t.Fatalf("completed status unavailable: %v", err)
	}
	second := trigger()
	if first == second {
		t.Fatal("credential did not rotate")
	}
	if response := request("/api/restore/progress", first); response.Code != http.StatusUnauthorized {
		t.Fatal("previous job credential accepted")
	}
	if response := request("/api/restore/progress", second); response.Code != http.StatusOK {
		t.Fatal("current job credential rejected")
	}
}
