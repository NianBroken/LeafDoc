// 本文件逐项检查用户文件响应头的生成结果，输入只提供文件路径和扩展名。
// HTML、SVG、图片与 PDF 的差别由响应头断言确认，不加载或执行实际文件内容。
package security

import (
	"net/http"
	"strings"
	"testing"
)

// TestSetUntrustedFileHeaders_ActiveContent_IsSandboxedDownload 遍历活动文档与无扩展名路径。
// 每个响应均应禁止嗅探、启用沙箱并要求下载，不通过真实浏览器执行文档。
func TestSetUntrustedFileHeaders_ActiveContent_IsSandboxedDownload(t *testing.T) {
	for _, name := range []string{"poc.html", "dir/poc.SVG", "poc.xml", "noext", "/assets/p1/"} {
		h := http.Header{}
		SetUntrustedFileHeaders(h, name)
		if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, want nosniff", name, got)
		}
		if got := h.Get("Content-Security-Policy"); got != "sandbox" {
			t.Errorf("%s: Content-Security-Policy = %q, want sandbox", name, got)
		}
		if got := h.Get("Content-Disposition"); !strings.HasPrefix(got, "attachment") {
			t.Errorf("%s: Content-Disposition = %q, want attachment", name, got)
		}
	}
}

// TestSetUntrustedFileHeaders_Media_IsInlineAndSandboxed 验证大写图片扩展名仍内联展示。
// 文件名只保留路径末段，图片响应同样带有沙箱策略。
func TestSetUntrustedFileHeaders_Media_IsInlineAndSandboxed(t *testing.T) {
	h := http.Header{}
	SetUntrustedFileHeaders(h, "/assets/p1/Pic.PNG")
	if got := h.Get("Content-Disposition"); got != "inline; filename=Pic.PNG" {
		t.Errorf("Content-Disposition = %q, want inline with filename", got)
	}
	if got := h.Get("Content-Security-Policy"); got != "sandbox" {
		t.Errorf("Content-Security-Policy = %q, want sandbox", got)
	}
}

// TestSetUntrustedFileHeaders_PDF_IsInlineWithoutSandbox 从空响应头开始验证 PDF 内联及沙箱豁免。
// 禁止内容类型嗅探仍生效，本用例不测试调用方预先设置的沙箱头。
func TestSetUntrustedFileHeaders_PDF_IsInlineWithoutSandbox(t *testing.T) {
	h := http.Header{}
	SetUntrustedFileHeaders(h, "doc.pdf")
	if got := h.Get("Content-Disposition"); got != "inline; filename=doc.pdf" {
		t.Errorf("Content-Disposition = %q, want inline with filename", got)
	}
	if got := h.Get("Content-Security-Policy"); got != "" {
		t.Errorf("Content-Security-Policy = %q, want none (Chrome won't render sandboxed PDFs)", got)
	}
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}
