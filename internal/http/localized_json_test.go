// Response translation preserves validation protocol and the actual rejection reason.
package http

import (
	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"testing"
)

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

func TestUnknownDiagnosticIsNotInvented(t *testing.T) {
	localization.SetLanguageProvider(func() string { return "zh" })
	response := map[string]any{"error": "Storage is temporarily unavailable"}
	translateResponse(response, 503)
	if response["error"] != "Storage is temporarily unavailable" {
		t.Fatal(response)
	}
}
