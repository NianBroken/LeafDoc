// 本文件分别验证密钥创建、列表与撤销在服务为空时返回同一个功能关闭哨兵错误。
package apikeys

import (
	"context"
	"testing"
)

// TestCreateAPIKey_NilServiceReturnsAPIKeysDisabled 确认服务缺失优先于输入处理且返回同一哨兵错误。
func TestCreateAPIKey_NilServiceReturnsAPIKeysDisabled(t *testing.T) {
	uc := NewCreateAPIKeyUseCase(nil)
	_, err := uc.Execute(context.Background(), CreateAPIKeyInput{Name: "k", UserID: "u1", CreatedBy: "admin1"})
	if err != ErrAPIKeysDisabled {
		t.Fatalf("expected ErrAPIKeysDisabled, got %v", err)
	}
}

// TestListAPIKeys_NilServiceReturnsAPIKeysDisabled 确认列表入口不会解引用空服务。
func TestListAPIKeys_NilServiceReturnsAPIKeysDisabled(t *testing.T) {
	uc := NewListAPIKeysUseCase(nil)
	_, err := uc.Execute(context.Background())
	if err != ErrAPIKeysDisabled {
		t.Fatalf("expected ErrAPIKeysDisabled, got %v", err)
	}
}

// TestRevokeAPIKey_NilServiceReturnsAPIKeysDisabled 确认撤销入口与创建、列表采用相同关闭语义。
func TestRevokeAPIKey_NilServiceReturnsAPIKeysDisabled(t *testing.T) {
	uc := NewRevokeAPIKeyUseCase(nil)
	err := uc.Execute(context.Background(), RevokeAPIKeyInput{ID: "k1"})
	if err != ErrAPIKeysDisabled {
		t.Fatalf("expected ErrAPIKeysDisabled, got %v", err)
	}
}
