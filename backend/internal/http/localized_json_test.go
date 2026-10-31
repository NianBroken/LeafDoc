// 本文件验证响应翻译保留字段校验协议和未知诊断的原始原因，不运行 HTTP 服务。
package http

import (
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/localization"
)

// 指定中文语言后，校验标识不变，已知字段消息与直接模板查询返回中文。
func TestValidationTranslationPreservesFields(t *testing.T) {
	localization.SetLanguageProvider(func() string { return "zh" })
	response := map[string]any{"error": "validation_error", "fields": []any{map[string]any{"field": "siteName", "message": "Site name must not be empty"}}}
	translateResponse(response, 400)
	if response["error"] != "validation_error" {
		t.Fatal(response)
	}
	field := response["fields"].([]any)[0].(map[string]any)
	if field["message"] != "站点名称不能为空" {
		t.Fatal(field)
	}
	if got := localization.Message("Site name must not exceed 100 characters"); got != "站点名称不能超过 100 个字符" {
		t.Fatal(got)
	}
}

// 未收录的非空诊断保持原文，不按状态码强行替换成另一种失败原因。
func TestUnknownDiagnosticIsNotInvented(t *testing.T) {
	localization.SetLanguageProvider(func() string { return "zh" })
	response := map[string]any{"error": "Storage is temporarily unavailable"}
	translateResponse(response, 503)
	if response["error"] != "Storage is temporarily unavailable" {
		t.Fatal(response)
	}
}
