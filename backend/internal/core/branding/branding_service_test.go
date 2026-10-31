// 本文件核验 Windows 卷路径识别，非 Windows 平台按平台条件跳过该用例。
package branding

import (
	"runtime"
	"testing"
)

// C:logo.png 含卷标但不必带斜杠，Windows 上仍须按受限路径拒绝。
func TestContainsPathTraversal_RejectsWindowsVolumePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("filepath volume paths are only recognized on Windows")
	}

	if !containsPathTraversal("C:logo.png") {
		t.Fatalf("expected Windows volume path to be rejected")
	}
}
