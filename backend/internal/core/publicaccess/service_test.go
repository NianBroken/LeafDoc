// 本文件验证公开访问固定来源的读取和拒绝更新行为，不依赖外部数据库。
package publicaccess

import (
	"testing"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// TestService_EnvManaged_ReturnsFixedValueAndRejectsSetEnabled 先确认开关与来源，再尝试关闭。
// 拒绝必须表现为可提取的本地化错误和固定错误码，且原先开启状态保持不变。
func TestService_EnvManaged_ReturnsFixedValueAndRejectsSetEnabled(t *testing.T) {
	svc := NewEnvManaged(true)

	if !svc.Enabled() {
		t.Fatal("expected Enabled() true for NewEnvManaged(true)")
	}
	if !svc.EnvManaged() {
		t.Fatal("expected EnvManaged() true")
	}

	err := svc.SetEnabled(false)
	if err == nil {
		t.Fatal("expected SetEnabled to fail on an env-managed service")
	}
	loc, ok := sharederrors.AsLocalizedError(err)
	if !ok {
		t.Fatalf("expected a *LocalizedError, got %T", err)
	}
	if loc.Code != ErrCodeEnvManaged {
		t.Fatalf("expected error code %q, got %q", ErrCodeEnvManaged, loc.Code)
	}
	if !svc.Enabled() {
		t.Fatal("expected Enabled() still true after a rejected SetEnabled")
	}
}
