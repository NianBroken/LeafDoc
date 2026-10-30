package auth

import (
	"testing"
)

func TestHashSecret_DeterministicAndDistinct(t *testing.T) {
	h1 := hashSecret("secret-a")
	h2 := hashSecret("secret-a")
	h3 := hashSecret("secret-b")
	if h1 != h2 {
		t.Fatalf("expected hashSecret to be deterministic, got %q vs %q", h1, h2)
	}
	if h1 == h3 {
		t.Fatalf("expected different secrets to hash differently")
	}
	if h1 == dummySecretHash {
		t.Fatalf("real secret hash must not collide with the dummy hash used for timing equalization")
	}
}

func TestGenerateKeyToken_ProducesNonEmptyDistinctValues(t *testing.T) {
	prefix1, secret1, err := generateKeyToken()
	if err != nil {
		t.Fatalf("generateKeyToken err: %v", err)
	}
	prefix2, secret2, err := generateKeyToken()
	if err != nil {
		t.Fatalf("generateKeyToken err: %v", err)
	}
	if prefix1 == "" || secret1 == "" {
		t.Fatalf("expected non-empty prefix/secret")
	}
	if prefix1 == prefix2 || secret1 == secret2 {
		t.Fatalf("expected distinct values across calls")
	}
}

func TestParseKeyToken_RoundTrip(t *testing.T) {
	prefix, secret, err := generateKeyToken()
	if err != nil {
		t.Fatalf("generateKeyToken err: %v", err)
	}
	token := apiKeyTokenPrefix + prefix + "_" + secret

	gotPrefix, gotSecret, ok := parseKeyToken(token)
	if !ok {
		t.Fatalf("expected token to parse")
	}
	if gotPrefix != prefix || gotSecret != secret {
		t.Fatalf("got (%q, %q), want (%q, %q)", gotPrefix, gotSecret, prefix, secret)
	}
}

func TestParseKeyToken_RejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"not-a-key-token",
		"lw_missingsecret",
		"lw__",
		"lw_prefixonly_",
		"lw__secretonly",
	}
	for _, tc := range cases {
		if _, _, ok := parseKeyToken(tc); ok {
			t.Fatalf("expected %q to be rejected as malformed", tc)
		}
	}
}

func TestIntersectRole_NeverWidensPermission(t *testing.T) {
	tests := []struct {
		userRole, keyRole, want string
	}{
		{RoleAdmin, RoleAdmin, RoleAdmin},
		{RoleAdmin, RoleEditor, RoleEditor},
		{RoleAdmin, RoleViewer, RoleViewer},
		{RoleEditor, RoleAdmin, RoleEditor},
		{RoleEditor, RoleEditor, RoleEditor},
		{RoleEditor, RoleViewer, RoleViewer},
		{RoleViewer, RoleAdmin, RoleViewer},
		{RoleViewer, RoleEditor, RoleViewer},
		{RoleViewer, RoleViewer, RoleViewer},
	}
	for _, tc := range tests {
		got := intersectRole(tc.userRole, tc.keyRole)
		if got != tc.want {
			t.Errorf("intersectRole(%q, %q) = %q, want %q", tc.userRole, tc.keyRole, got, tc.want)
		}
	}
}

func TestIntersectRole_UnknownRoleFailsSafeToViewer(t *testing.T) {
	if got := intersectRole("bogus", RoleAdmin); got != RoleViewer {
		t.Errorf("intersectRole(bogus, admin) = %q, want %q (fail safe)", got, RoleViewer)
	}
	if got := intersectRole(RoleAdmin, "bogus"); got != RoleViewer {
		t.Errorf("intersectRole(admin, bogus) = %q, want %q (fail safe)", got, RoleViewer)
	}
}

// generateKeyToken provides a matched random prefix and secret.
func generateKeyToken() (prefix string, secret string, err error) {
	prefix, err = generatePrefix()
	if err != nil {
		return "", "", err
	}
	secret, err = generateSecret()
	if err != nil {
		return "", "", err
	}
	return prefix, secret, nil
}
