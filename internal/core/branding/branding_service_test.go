package branding

import (
	"runtime"
	"testing"
)

func TestContainsPathTraversal_RejectsWindowsVolumePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("filepath volume paths are only recognized on Windows")
	}

	if !containsPathTraversal("C:logo.png") {
		t.Fatalf("expected Windows volume path to be rejected")
	}
}
