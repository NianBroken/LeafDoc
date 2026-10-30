package http_test

import (
	"strings"
	"testing"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
)

func TestBuildCustomStylesheetTag(t *testing.T) {
	tag := httpinternal.BuildCustomStylesheetTag("/wiki", "/tmp/custom.css")

	expected := `<link rel="stylesheet" href="/wiki/custom.css">`
	if tag != expected {
		t.Fatalf("expected %q, got %q", expected, tag)
	}
}

func TestBuildCustomStylesheetTag_EmptyPath(t *testing.T) {
	tag := httpinternal.BuildCustomStylesheetTag("", "")
	if tag != "" {
		t.Fatalf("expected empty tag, got %q", tag)
	}
}

func TestInjectIntoHead(t *testing.T) {
	html := "<html><head></head><body></body></html>"
	got := httpinternal.InjectIntoHead(html, `<link rel="stylesheet" href="/custom.css">`)

	if !strings.Contains(got, `<link rel="stylesheet" href="/custom.css">`) {
		t.Fatalf("expected stylesheet link to be injected, got %q", got)
	}
}

func TestBuildFrontendFaviconHref(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		faviconFile string
		want        string
	}{
		{
			name:     "default favicon without base path",
			want:     "/defaults/leafdoc-favicon.ico",
			basePath: "",
		},
		{
			name:     "default favicon with base path",
			basePath: "/wiki",
			want:     "/wiki/defaults/leafdoc-favicon.ico",
		},
		{
			name:        "custom favicon without base path",
			faviconFile: "favicon.ico",
			want:        "/branding/favicon.ico",
		},
		{
			name:        "custom favicon with base path",
			basePath:    "/wiki",
			faviconFile: "favicon.webp",
			want:        "/wiki/branding/favicon.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := httpinternal.BuildFrontendFaviconHref(tt.basePath, tt.faviconFile)
			if got != tt.want {
				t.Fatalf("BuildFrontendFaviconHref(%q, %q) = %q, want %q", tt.basePath, tt.faviconFile, got, tt.want)
			}
		})
	}
}

func TestBuildCustomStylesheetTag_WhitespacePath(t *testing.T) {
	tag := httpinternal.BuildCustomStylesheetTag("/wiki", "   ")
	if tag != "" {
		t.Fatalf("expected empty tag for whitespace path, got %q", tag)
	}
}
