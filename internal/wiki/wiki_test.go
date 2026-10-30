package wiki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/testutil"

	"encoding/json"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"

	wikipages "github.com/NianBroken/LeafDoc/internal/wiki/pages"
	wikiresync "github.com/NianBroken/LeafDoc/internal/wiki/resync"
)

func createWikiTestInstance(t *testing.T) *Wiki {
	return createWikiTestInstanceWithMetrics(t, nil)
}

func createWikiTestInstanceWithMetrics(t *testing.T, metrics *httpmetrics.HTTPMetrics) *Wiki {
	wikiInstance, err := NewWiki(&WikiOptions{
		StorageDir:          t.TempDir(),
		AdminPassword:       "adminpassword",
		JWTSecret:           "secretkey",
		AccessTokenTimeout:  15 * time.Minute,
		RefreshTokenTimeout: 7 * 24 * time.Hour,
		EnableRevision:      true,
		Metrics:             metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create wiki instance: %v", err)
	}
	return wikiInstance
}

func metricsBody(t *testing.T, metrics *httpmetrics.HTTPMetrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	metrics.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics endpoint to return 200, got %d", rec.Code)
	}
	return rec.Body.String()
}

func pageNodeKind() *tree.NodeKind {
	kind := tree.NodeKindPage
	return &kind
}

func createPageForTest(t *testing.T, w *Wiki, userID string, parentID *string, title, slug string, kind *tree.NodeKind) *tree.Page {
	t.Helper()

	out, err := wikipages.NewCreatePageUseCase(w.tree, w.slug, w.newPageOrchestrator(), w.log, nil).Execute(
		context.Background(),
		wikipages.CreatePageInput{UserID: userID, ParentID: parentID, Title: title, Slug: slug, Kind: kind},
	)
	if err != nil {
		t.Fatalf("CreatePage failed: %v", err)
	}
	return out.Page
}

func updatePageForTest(t *testing.T, w *Wiki, userID, id, title, slug string, content *string, kind *tree.NodeKind) *tree.Page {
	t.Helper()

	current, err := w.tree.GetPage(id)
	if err != nil {
		t.Fatalf("GetPage before update failed: %v", err)
	}

	out, err := wikipages.NewUpdatePageUseCase(w.tree, w.slug, w.newPageOrchestrator(), w.log, nil).Execute(
		context.Background(),
		wikipages.UpdatePageInput{UserID: userID, ID: id, Version: current.Version(), Title: title, Slug: slug, Content: content, Kind: kind},
	)
	if err != nil {
		t.Fatalf("UpdatePage failed: %v", err)
	}
	return out.Page
}

func deletePageForTest(t *testing.T, w *Wiki, userID, id string, recursive bool) {
	t.Helper()

	current, err := w.tree.GetPage(id)
	if err != nil {
		t.Fatalf("GetPage before delete failed: %v", err)
	}

	if err := wikipages.NewDeletePageUseCase(w.tree, w.revision, w.asset, w.favorites, w.newPageOrchestrator(), w.log, nil).Execute(
		context.Background(),
		wikipages.DeletePageInput{UserID: userID, ID: id, Version: current.Version(), Recursive: recursive},
	); err != nil {
		t.Fatalf("DeletePage failed: %v", err)
	}
}

func TestWiki_DeletePage_Simple(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)
	page := createPageForTest(t, w, "system", nil, "Trash", "trash", pageNodeKind())
	deletePageForTest(t, w, "system", page.ID, false)
	if _, err := w.tree.GetPage(page.ID); err == nil {
		t.Fatalf("expected deleted page to be gone")
	}
}

func TestWiki_DeletePage_WithChildren(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)
	parent := createPageForTest(t, w, "system", nil, "Parent", "parent", pageNodeKind())
	createPageForTest(t, w, "system", &parent.ID, "Child", "child", pageNodeKind())

	err := wikipages.NewDeletePageUseCase(w.tree, w.revision, w.asset, w.favorites, w.newPageOrchestrator(), w.log, nil).Execute(
		context.Background(),
		wikipages.DeletePageInput{UserID: "system", ID: parent.ID, Version: parent.Version(), Recursive: false},
	)
	if err == nil {
		t.Error("Expected error when deleting parent with children")
	}
}

func TestWiki_DeletePage_Recursive(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)
	parent := createPageForTest(t, w, "system", nil, "Parent", "parent", pageNodeKind())
	child := createPageForTest(t, w, "system", &parent.ID, "Child", "child", pageNodeKind())

	deletePageForTest(t, w, "system", parent.ID, true)
	if _, err := w.tree.GetPage(parent.ID); err == nil {
		t.Fatalf("expected deleted parent to be gone")
	}
	if _, err := w.tree.GetPage(child.ID); err == nil {
		t.Fatalf("expected deleted child to be gone")
	}
}

func TestWiki_TriggerResyncAsync_EmitsMetrics(t *testing.T) {
	testutil.ResetMySQL(t)
	metrics := httpmetrics.NewHTTPMetrics("test")
	w := createWikiTestInstanceWithMetrics(t, metrics)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)

	w.TriggerResyncAsync()

	deadline := time.Now().Add(5 * time.Second)
	for {
		status := w.resyncJob.Status()
		if status.Done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for resync to finish")
		}
		time.Sleep(10 * time.Millisecond)
	}

	body := metricsBody(t, metrics)
	if !strings.Contains(body, `leafdoc_resync_runs_total{result="success"} 1`) {
		t.Fatalf("expected resync run metric, got: %s", body)
	}
	if !strings.Contains(body, `leafdoc_resync_duration_seconds_bucket{result="success"`) {
		t.Fatalf("expected resync duration metric, got: %s", body)
	}
}

func TestWiki_DeletePage_PurgesRevisionData(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)

	page := createPageForTest(t, w, "system", nil, "Page", "page", pageNodeKind())
	content := "updated"
	updatePageForTest(t, w, "system", page.ID, page.Title, page.Slug, &content, pageNodeKind())

	deletePageForTest(t, w, "system", page.ID, false)

	revisions, err := w.revision.ListRevisions(page.ID)
	if err != nil {
		t.Fatalf("ListRevisions failed: %v", err)
	}
	if len(revisions) != 0 {
		t.Fatalf("expected revisions to be purged, got %#v", revisions)
	}
}

func TestWiki_InitDefaultAdmin_UsesGivenPassword(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)

	_, err := w.user.GetUserByEmailOrUsernameAndPassword("admin", "adminpassword")
	if err != nil {
		t.Fatalf("Admin user not found: %v", err)
	}
}

func TestWiki_InitDefaultAdmin_UsesGivenUsernameAndEmail(t *testing.T) {
	testutil.ResetMySQL(t)
	wikiInstance, err := NewWiki(&WikiOptions{
		StorageDir:          t.TempDir(),
		AdminUsername:       "root",
		AdminEmail:          "root@example.com",
		AdminPassword:       "adminpassword",
		JWTSecret:           "secretkey",
		AccessTokenTimeout:  15 * time.Minute,
		RefreshTokenTimeout: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create wiki instance: %v", err)
	}
	defer testutil.WrapCloseWithErrorCheck(wikiInstance.Close, t)

	user, err := wikiInstance.user.GetUserByEmailOrUsernameAndPassword("root", "adminpassword")
	if err != nil {
		t.Fatalf("Admin user with custom username not found: %v", err)
	}
	if user.Email != "root@example.com" {
		t.Errorf("Expected admin email %q, got %q", "root@example.com", user.Email)
	}
}

// one, and InitDefaultAdmin then created a fresh admin from the startup
// flags. NewWiki must fail before that and leave the file alone (ADR-0013).
func TestWiki_Close_ClosesAllDatabases(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	w, err := NewWiki(&WikiOptions{
		StorageDir:             dir,
		AdminPassword:          "adminpassword",
		JWTSecret:              "secretkey",
		AccessTokenTimeout:     15 * time.Minute,
		RefreshTokenTimeout:    7 * 24 * time.Hour,
		EnableRevision:         true,
		EnableAPIKeyManagement: true,
	})
	if err != nil {
		t.Fatalf("NewWiki: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "users.db")); !os.IsNotExist(err) {
		t.Fatal("local database was generated")
	}
}

func TestWiki_Login_SuccessAndFailure(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)

	authSvc := w.auth
	if authSvc == nil {
		t.Fatal("expected auth service to be initialized")
	}

	token, err := authSvc.Login("admin", "adminpassword")
	if err != nil || token == nil {
		t.Error("Expected login to succeed with default admin password")
	}

	_, err = authSvc.Login("admin", "wrong")
	if err == nil {
		t.Error("Expected login to fail with wrong password")
	}
}

// TestWiki_APIKeyManagementDisabled_APIKeysUnavailable ensures the
// experimental API key feature stays off by default: with auth enabled but
// EnableAPIKeyManagement left false, APIKeyService() must be nil so the
// Bearer middleware never gets registered and the admin CRUD routes report
// the feature as disabled.
func TestWiki_APIKeyManagementDisabled_APIKeysUnavailable(t *testing.T) {
	testutil.ResetMySQL(t)
	wikiInstance, err := NewWiki(&WikiOptions{
		StorageDir:             t.TempDir(),
		AdminPassword:          "adminpassword",
		JWTSecret:              "secretkey",
		AccessTokenTimeout:     15 * time.Minute,
		RefreshTokenTimeout:    7 * 24 * time.Hour,
		EnableAPIKeyManagement: false,
	})
	if err != nil {
		t.Fatalf("Failed to create wiki instance: %v", err)
	}
	defer testutil.WrapCloseWithErrorCheck(wikiInstance.Close, t)

	if wikiInstance.APIKeyService() != nil {
		t.Error("Expected APIKeyService() to be nil when EnableAPIKeyManagement is false")
	}
}

// TestWiki_APIKeyManagementEnabled_APIKeysAvailable confirms opting in via
// EnableAPIKeyManagement actually constructs the service (auth must also be
// enabled, since API keys are meaningless without it).
func TestWiki_APIKeyManagementEnabled_APIKeysAvailable(t *testing.T) {
	testutil.ResetMySQL(t)
	wikiInstance, err := NewWiki(&WikiOptions{
		StorageDir:             t.TempDir(),
		AdminPassword:          "adminpassword",
		JWTSecret:              "secretkey",
		AccessTokenTimeout:     15 * time.Minute,
		RefreshTokenTimeout:    7 * 24 * time.Hour,
		EnableAPIKeyManagement: true,
	})
	if err != nil {
		t.Fatalf("Failed to create wiki instance: %v", err)
	}
	defer testutil.WrapCloseWithErrorCheck(wikiInstance.Close, t)

	if wikiInstance.APIKeyService() == nil {
		t.Error("Expected APIKeyService() to be non-nil when EnableAPIKeyManagement is true")
	}
}

func TestWiki_EnsureBaselineRevisions_SkipsUnreadablePages(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)

	okPage := createPageForTest(t, w, "system", nil, "Healthy", "healthy", pageNodeKind())
	brokenPage := createPageForTest(t, w, "system", nil, "Broken", "broken", pageNodeKind())

	if err := w.revision.DeletePageData(okPage.ID); err != nil {
		t.Fatalf("DeletePageData(okPage) failed: %v", err)
	}
	if err := w.revision.DeletePageData(brokenPage.ID); err != nil {
		t.Fatalf("DeletePageData(brokenPage) failed: %v", err)
	}

	brokenKey := brokenPage.ID
	if err := database.DeleteRecord(w.storageDir, "page_nodes", brokenKey); err != nil {
		t.Fatalf("Remove(%s) failed: %v", brokenKey, err)
	}

	w.ensureBaselineRevisions()

	okRevisions, err := w.revision.ListRevisions(okPage.ID)
	if err != nil {
		t.Fatalf("ListRevisions(okPage) failed: %v", err)
	}
	if len(okRevisions) == 0 {
		t.Fatalf("expected baseline revision for readable page")
	}

	brokenRevisions, err := w.revision.ListRevisions(brokenPage.ID)
	if err != nil {
		t.Fatalf("ListRevisions(brokenPage) failed: %v", err)
	}
	if len(brokenRevisions) != 0 {
		t.Fatalf("expected no baseline revision for unreadable page, got %d", len(brokenRevisions))
	}
}

func TestWiki_ReloadWithProgress_PicksUpDiskChangesAndFinishesInSearchPhase(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)

	external := tree.NewTreeService(w.storageDir)
	if err := external.LoadTree(); err != nil {
		t.Fatal(err)
	}
	if _, err := external.CreateNode("system", nil, "From MySQL", "from-disk", pageNodeKind()); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if !w.resyncJob.Start() {
		t.Fatal("expected resync job to start")
	}

	w.reloadWithProgress(context.Background())

	status := w.resyncJob.Status()
	want := wikiresync.JobStatus{Running: false, Phase: string(wikiresync.PhaseSearch), Done: true}
	if status != want {
		t.Fatalf("status = %+v, want %+v", status, want)
	}
	found := false
	if err := w.tree.WalkNodes(func(id string) error {
		if page, err := w.tree.GetPage(id); err == nil && page.Slug == "from-disk" {
			found = true
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkNodes failed: %v", err)
	}
	if !found {
		t.Fatal("expected reload to pick up the page written to disk")
	}
}

func TestWiki_ReloadWithProgress_CanceledContext_FinishesWithErrorAndFailureMetric(t *testing.T) {
	testutil.ResetMySQL(t)
	metrics := httpmetrics.NewHTTPMetrics("test")
	w := createWikiTestInstanceWithMetrics(t, metrics)
	defer testutil.WrapCloseWithErrorCheck(w.Close, t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !w.resyncJob.Start() {
		t.Fatal("expected resync job to start")
	}

	w.reloadWithProgress(ctx)

	status := w.resyncJob.Status()
	want := wikiresync.JobStatus{
		Running: false,
		Phase:   string(wikiresync.PhaseTree),
		Done:    true,
		Error:   "tree reconstruction failed: context canceled",
	}
	if status != want {
		t.Fatalf("status = %+v, want %+v", status, want)
	}
	body := metricsBody(t, metrics)
	for _, metric := range []string{
		`leafdoc_resync_runs_total{result="error"} 1`,
		`leafdoc_resync_failures_total{result="error"} 1`,
	} {
		if !strings.Contains(body, metric) {
			t.Fatalf("expected %s in metrics, got: %s", metric, body)
		}
	}
}

func newWikiOnDir(dir string) (*Wiki, error) {
	return NewWiki(&WikiOptions{
		StorageDir:          dir,
		AdminPassword:       "adminpassword",
		JWTSecret:           "secretkey",
		AccessTokenTimeout:  15 * time.Minute,
		RefreshTokenTimeout: 7 * 24 * time.Hour,
	})
}

// seedWikiWithUser boots a wiki on dir, adds the user alice with a favorite
// and closes it again, as a previous run of the server would. Like
// cmd/leafdoc, it creates the data dir first.

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "wiki") }

// An unavailable database must fail startup instead of creating local state.
func TestMySQLWikiConnectionFailure(t *testing.T) {
	testutil.ResetMySQL(t)
	t.Setenv("LEAFDOC_MYSQL_PASSWORD", "invalid-test-password")
	if w, err := newWikiOnDir(t.TempDir()); err == nil {
		w.Close()
		t.Fatal("startup succeeded with invalid database credentials")
	}
}

// All durable stores are covered by the credential-free database dump.
func TestMySQLWikiBackupCoversBusinessTables(t *testing.T) {
	testutil.ResetMySQL(t)
	w := createWikiTestInstance(t)
	defer w.Close()
	raw, err := database.Export(context.Background(), w.storageDir)
	if err != nil {
		t.Fatal(err)
	}
	var dump database.Dump
	if err = json.Unmarshal(raw, &dump); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, table := range dump.Tables {
		names[table.Name] = true
	}
	for _, name := range []string{"users", "sessions", "favorites", "links", "page_tags", "page_properties", "content_records"} {
		if !names[name] {
			t.Errorf("backup missing %s", name)
		}
	}
}
