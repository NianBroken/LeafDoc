package apikeys

import (
	"context"
	"testing"
)

func TestCreateAPIKey_NilServiceReturnsAPIKeysDisabled(t *testing.T) {
	uc := NewCreateAPIKeyUseCase(nil)
	_, err := uc.Execute(context.Background(), CreateAPIKeyInput{Name: "k", UserID: "u1", CreatedBy: "admin1"})
	if err != ErrAPIKeysDisabled {
		t.Fatalf("expected ErrAPIKeysDisabled, got %v", err)
	}
}

func TestListAPIKeys_NilServiceReturnsAPIKeysDisabled(t *testing.T) {
	uc := NewListAPIKeysUseCase(nil)
	_, err := uc.Execute(context.Background())
	if err != ErrAPIKeysDisabled {
		t.Fatalf("expected ErrAPIKeysDisabled, got %v", err)
	}
}

func TestRevokeAPIKey_NilServiceReturnsAPIKeysDisabled(t *testing.T) {
	uc := NewRevokeAPIKeyUseCase(nil)
	err := uc.Execute(context.Background(), RevokeAPIKeyInput{ID: "k1"})
	if err != ErrAPIKeysDisabled {
		t.Fatalf("expected ErrAPIKeysDisabled, got %v", err)
	}
}
