package publicaccess

import (
	"testing"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

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
