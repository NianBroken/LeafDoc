package http

import (
	"fmt"

	"net/http/httptest"

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
