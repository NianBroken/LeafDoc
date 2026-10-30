package auth

import (
	"context"
	"testing"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
)

func TestRequestPasswordResetUseCase_Execute_EmailDisabled_ReturnsErrEmailDisabled(t *testing.T) {
	uc := NewRequestPasswordResetUseCase(func() *coreauth.EmailTokenService { return nil })

	if err := uc.Execute(context.Background(), RequestPasswordResetInput{Identifier: "alice"}); err != coreauth.ErrEmailDisabled {
		t.Fatalf("expected ErrEmailDisabled, got %v", err)
	}
}
