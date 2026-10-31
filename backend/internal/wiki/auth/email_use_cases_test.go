// 本文件验证密码重置用例在邮件能力未配置时的明确失败分支，不启动邮件连接。
package auth

import (
	"context"
	"testing"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
)

// TestRequestPasswordResetUseCase_Execute_EmailDisabled_ReturnsErrEmailDisabled 使用空邮件入口结果。
// 确认未配置功能返回确切哨兵错误，不启动后台发送也不把配置缺失包装为成功。
func TestRequestPasswordResetUseCase_Execute_EmailDisabled_ReturnsErrEmailDisabled(t *testing.T) {
	uc := NewRequestPasswordResetUseCase(func() *coreauth.EmailTokenService { return nil })

	if err := uc.Execute(context.Background(), RequestPasswordResetInput{Identifier: "alice"}); err != coreauth.ErrEmailDisabled {
		t.Fatalf("expected ErrEmailDisabled, got %v", err)
	}
}
