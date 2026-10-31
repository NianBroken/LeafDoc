package http

import (
	"fmt"

	"net/http/httptest"

	"testing"

	"github.com/gin-gonic/gin"
)

// frontendTemplateRouter 临时启用嵌入前端，构造指定基础路径且不记录请求日志的路由。
// 清理函数恢复包级开关，基准调用期间依赖该开关与其他测试不并发修改。
func frontendTemplateRouter(tb testing.TB, base string) *gin.Engine {
	tb.Helper()
	old := EmbedFrontend
	EmbedFrontend = "true"
	tb.Cleanup(func() { EmbedFrontend = old })
	return NewRouter(nil, FrontendConfig{}, RouterOptions{BasePath: base, DisableRequestLog: true})
}

// BenchmarkFrontendHTMLTemplate 对根部署与 wiki 前缀部署的深层页面请求测量分配及耗时。
// 未嵌入构建后的 index.html 时跳过，测试内存响应状态而非实际浏览器渲染。
func BenchmarkFrontendHTMLTemplate(b *testing.B) {
	if _, err := frontend.Open("dist/index.html"); err != nil {
		b.Skip("requires built frontend (make ui)")
	}
	for _, base := range []string{"", "/wiki"} {
		b.Run(fmt.Sprintf("base=%s", base), func(b *testing.B) {
			router := frontendTemplateRouter(b, base)
			b.ReportAllocs()
			// 每轮单独创建请求与记录器，测量包含这些对象及路由处理的开销。
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
