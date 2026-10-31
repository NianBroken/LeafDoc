package assets

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newAssetServeRouter 将样本写入自动清理的临时目录，再挂载真实静态文件处理与响应头中间件。
// 此路由只验证文件响应规则，不包含生产路由的登录或公开读取门禁。
func newAssetServeRouter(t *testing.T, files map[string]string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	for name, content := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	engine := gin.New()
	g := engine.Group("/assets")
	g.Use(assetServeHeaders())
	g.StaticFS("/", gin.Dir(dir, false))
	return engine
}

// getAsset 发出内存 GET 请求，保留状态、内容及响应头供调用方逐项断言。
func getAsset(engine *gin.Engine, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// TestAssetServeHeaders_ActiveContent_IsForcedDownload 检查实际文件响应对各类活动文档强制下载。
// 样本中的脚本只是文件内容，不由测试执行；检查范围是 200 状态与三个安全响应头。
func TestAssetServeHeaders_ActiveContent_IsForcedDownload(t *testing.T) {
	const payload = `<script>alert(document.domain)</script>`
	engine := newAssetServeRouter(t, map[string]string{
		"p1/poc.html":  payload,
		"p1/poc.HTML":  payload,
		"p1/poc.xhtml": payload,
		"p1/poc.xml":   payload,
		"p1/x.svg":     `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`,
		"p1/noext":     "<html>" + payload + "</html>", // 无扩展名但内容包含 HTML，仍按下载规则处理。
	})

	for _, p := range []string{"poc.html", "poc.HTML", "poc.xhtml", "poc.xml", "x.svg", "noext"} {
		rec := getAsset(engine, "/assets/p1/"+p)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", p, rec.Code)
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, want nosniff", p, got)
		}
		if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "attachment") {
			t.Errorf("%s: Content-Disposition = %q, want attachment", p, got)
		}
		if got := rec.Header().Get("Content-Security-Policy"); got != "sandbox" {
			t.Errorf("%s: Content-Security-Policy = %q, want sandbox", p, got)
		}
	}
}

// TestAssetServeHeaders_SafeMedia_StaysInline 确认集合内媒体扩展名维持内联及禁止嗅探。
// 样本只用于文件发送与响应头，不要求具备能够解码播放的完整媒体内容。
func TestAssetServeHeaders_SafeMedia_StaysInline(t *testing.T) {
	engine := newAssetServeRouter(t, map[string]string{
		"p1/doc.pdf":  "%PDF-1.4",
		"p1/pic.png":  "\x89PNG",
		"p1/pic.JPG":  "x",
		"p1/vid.mp4":  "x",
		"p1/song.mp3": "x",
	})

	for _, p := range []string{"doc.pdf", "pic.png", "pic.JPG", "vid.mp4", "song.mp3"} {
		rec := getAsset(engine, "/assets/p1/"+p)
		if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "inline") {
			t.Errorf("%s: Content-Disposition = %q, want inline", p, got)
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, want nosniff", p, got)
		}
	}
}
