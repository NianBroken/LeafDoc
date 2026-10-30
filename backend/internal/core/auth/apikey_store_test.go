package auth

import (
	"testing"
	"time"
)

func TestAPIKey_IsActive_ExpiredIsInactive(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	key := &APIKey{ExpiresAt: &past}
	if key.IsActive(time.Now()) {
		t.Fatalf("expected expired key to be inactive")
	}
}

func TestAPIKey_IsActive_NoExpiryIsActive(t *testing.T) {
	key := &APIKey{}
	if !key.IsActive(time.Now()) {
		t.Fatalf("expected key with no expiry and no revocation to be active")
	}
}
