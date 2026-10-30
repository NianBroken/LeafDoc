package http

import (
	"encoding/json"
	"fmt"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"html"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func frontendTemplateRouter(tb testing.TB, base string) *gin.Engine {
	tb.Helper()
	old := EmbedFrontend
	EmbedFrontend = "true"
	tb.Cleanup(func() { EmbedFrontend = old })
	return NewRouter(nil, FrontendConfig{}, RouterOptions{BasePath: base, DisableRequestLog: true})
}
func TestFrontendTemplate_DynamicBrandingAndDeepRoute_PreserveValues(t *testing.T) {
	testutil.ResetMySQL(t)
	if _, err := frontend.Open("dist/index.html"); err != nil {
		t.Skip("requires built frontend (make ui)")
	}
	old := EmbedFrontend
	EmbedFrontend = "true"
	defer func() { EmbedFrontend = old }()
	name, favicon := "中文 <Wiki> \"名称\" $& </title><script>test</script>", "first.png"
	router := NewRouter(nil, FrontendConfig{GetSiteName: func() string { return name }, GetFaviconFile: func() string { return favicon }}, RouterOptions{BasePath: "/wiki", DisableRequestLog: true})
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("GET", "/wiki/deep/page", nil))
		body := rec.Body.String()
		if rec.Code != 200 || !strings.Contains(body, html.EscapeString(name)) || !strings.Contains(body, "/wiki/branding/"+favicon) || strings.Contains(body, "{{__") {
			t.Fatalf("status=%d body=%s", rec.Code, body)
		}
		metadata := regexp.MustCompile(`<meta name="leafdoc-branding" content="([^"]*)">`).FindStringSubmatch(body)
		var initial map[string]string
		if len(metadata) != 2 {
			t.Fatal("missing initial branding")
		}
		if err := json.Unmarshal([]byte(html.UnescapeString(metadata[1])), &initial); err != nil {
			t.Fatal(err)
		}
		if initial["siteName"] != name || initial["faviconFile"] != favicon || strings.Contains(body, "<script>test</script>") || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("unsafe or stale initial branding")
		}
		for _, match := range regexp.MustCompile(`(?:src|href)="([^"]+\.(?:js|css))"`).FindAllStringSubmatch(body, -1) {
			asset := httptest.NewRecorder()
			router.ServeHTTP(asset, httptest.NewRequest("GET", match[1], nil))
			if asset.Code != 200 {
				t.Fatalf("%s status %d", match[1], asset.Code)
			}
		}
		name = "Second Wiki"
		favicon = "second.png"
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/outside", nil))
	if rec.Code != 404 {
		t.Fatal(rec.Code)
	}
}
func BenchmarkFrontendHTMLTemplate(b *testing.B) {
	if _, err := frontend.Open("dist/index.html"); err != nil {
		b.Skip("requires built frontend (make ui)")
	}
	for _, base := range []string{"", "/wiki"} {
		b.Run(fmt.Sprintf("base=%s", base), func(b *testing.B) {
			router := frontendTemplateRouter(b, base)
			b.ReportAllocs()
			for b.Loop() {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/deep/page", nil))
				if rec.Code != 200 {
					b.Fatal(rec.Code)
				}
			}
		})
	}
}
