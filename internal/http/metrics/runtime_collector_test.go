package metrics

import (
	"errors"
	"strings"
	"testing"
)

type fakeRuntimeSource struct {
	pages    PageStats
	pagesErr error
	users    UserStats
	usersErr error
}

func TestHTTPMetrics_RegisterRuntimeStats_SkipsGaugesWhenSourceErrors(t *testing.T) {
	m := NewHTTPMetrics("test")
	m.RegisterRuntimeStats(fakeRuntimeSource{
		pagesErr: errors.New("tree not loaded"),
		usersErr: errors.New("user store unavailable"),
	})

	// A source error must not break the whole endpoint: /metrics still
	// serves 200 with the other metrics, just without the runtime gauges.
	body := getMetricsBody(t, m.HTTPHandler())
	if !strings.Contains(body, "leafwiki_build_info") {
		t.Fatalf("expected the rest of /metrics to keep working, got:\n%s", body)
	}
	if strings.Contains(body, "leafwiki_pages{") || strings.Contains(body, "leafwiki_users{") {
		t.Errorf("expected no runtime gauge samples when source errors, got:\n%s", body)
	}
}

func TestHTTPMetrics_RegisterRuntimeStats_NilReceiverIsNoop(t *testing.T) {
	var m *HTTPMetrics
	m.RegisterRuntimeStats(fakeRuntimeSource{}) // must not panic
}

func (f fakeRuntimeSource) PageStats() (PageStats, error) { return f.pages, f.pagesErr }
func (f fakeRuntimeSource) UserStats() (UserStats, error) { return f.users, f.usersErr }
