package jwks_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/jwks"
)

func TestNewKeyStore(t *testing.T) {
	ks, err := jwks.NewKeyStore()
	if err != nil {
		t.Fatalf("NewKeyStore() error = %v", err)
	}
	jwkSet := ks.GetJWKS()
	if len(jwkSet.Keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(jwkSet.Keys))
	}
}

func TestRotate(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	_, err := ks.Rotate()
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	jwkSet := ks.GetJWKS()
	if len(jwkSet.Keys) != 2 {
		t.Errorf("expected 2 keys after rotation, got %d", len(jwkSet.Keys))
	}
}

func TestSignAndVerify(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	data := []byte("test payload")

	kid, sig, err := ks.Sign(data)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if err := ks.VerifyWithKid(data, kid, sig); err != nil {
		t.Errorf("VerifyWithKid() error = %v", err)
	}
}

func TestSignAndVerify_AfterRotation(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	data := []byte("test payload")

	// Sign with first key
	kid1, sig1, _ := ks.Sign(data)

	// Rotate
	_, _ = ks.Rotate()

	// Sign with new key
	kid2, sig2, _ := ks.Sign(data)

	// Both should verify
	if err := ks.VerifyWithKid(data, kid1, sig1); err != nil {
		t.Errorf("old key verification failed: %v", err)
	}
	if err := ks.VerifyWithKid(data, kid2, sig2); err != nil {
		t.Errorf("new key verification failed: %v", err)
	}

	// Kids should be different
	if kid1 == kid2 {
		t.Error("expected different kids after rotation")
	}
}

func TestVerify_AfterRevocation(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	data := []byte("test payload")

	kid1, sig1, _ := ks.Sign(data)
	_, _ = ks.Rotate()
	_ = ks.RevokeOldest()

	err := ks.VerifyWithKid(data, kid1, sig1)
	if err != jwks.ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound after revocation, got %v", err)
	}
}

func TestVerify_KeyNotFound(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	err := ks.VerifyWithKid([]byte("data"), "nonexistent-kid", "sig")
	if err != jwks.ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestCache_Lookup(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	cache := jwks.NewCache(time.Hour, false)
	cache.Fetch(ks.GetJWKS())

	kid := ks.CurrentKid()
	key, found := cache.Lookup(kid)
	if !found {
		t.Error("expected to find key in cache")
	}
	if key.Kid != kid {
		t.Errorf("Kid = %q, want %q", key.Kid, kid)
	}
}

func TestCache_Stale(t *testing.T) {
	cache := jwks.NewCache(0, false) // Zero TTL = immediately stale
	ks, _ := jwks.NewKeyStore()
	cache.Fetch(ks.GetJWKS())

	// Wait briefly to ensure staleness
	time.Sleep(time.Millisecond)

	if !cache.IsStale() {
		t.Error("expected cache to be stale with zero TTL")
	}
}

func TestCache_Smart(t *testing.T) {
	cache := jwks.NewCache(time.Hour, true)
	if !cache.IsSmart() {
		t.Error("expected smart mode to be enabled")
	}
	cache.SetSmart(false)
	if cache.IsSmart() {
		t.Error("expected smart mode to be disabled")
	}
}

func TestMarshalAndParseToken(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	payload := []byte(`{"sub":"user-123"}`)
	kid := ks.CurrentKid()

	signer := func(data []byte) (string, error) {
		return ks.SignWithKid(data, kid)
	}

	tok, err := jwks.MarshalToken(payload, kid, signer)
	if err != nil {
		t.Fatalf("MarshalToken() error = %v", err)
	}

	parsedKid, parsedPayload, signingInput, parsedSig, err := jwks.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if parsedKid != kid {
		t.Errorf("kid = %q, want %q", parsedKid, kid)
	}
	if string(parsedPayload) != string(payload) {
		t.Errorf("payload = %q, want %q", parsedPayload, payload)
	}
	if parsedSig == "" {
		t.Error("expected non-empty signature")
	}

	// Signature must verify against the signing input (header.payload)
	if err := ks.VerifyWithKid([]byte(signingInput), kid, parsedSig); err != nil {
		t.Errorf("signature verification failed: %v", err)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	_, _, _, _, err := jwks.ParseToken("not-a-token")
	if err != jwks.ErrTokenFormat {
		t.Errorf("expected ErrTokenFormat, got %v", err)
	}
}

func TestMarshalToken_HeaderTamperDetected(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	payload := []byte(`{"sub":"victim"}`)
	kid := ks.CurrentKid()

	signer := func(data []byte) (string, error) {
		return ks.SignWithKid(data, kid)
	}

	tok, err := jwks.MarshalToken(payload, kid, signer)
	if err != nil {
		t.Fatalf("MarshalToken() error = %v", err)
	}

	// Tamper with the header: replace alg with "none"
	_, _, _, originalSig, err := jwks.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	// Build a tampered token with alg=none but keep original signature
	tamperedHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT","kid":"` + kid + `"}`))
	originalParts := splitTestToken(tok)
	tamperedToken := tamperedHeader + "." + originalParts[1] + "." + originalSig

	// Parse the tampered token
	tamperedKid, _, tamperedSigningInput, tamperedSig, err := jwks.ParseToken(tamperedToken)
	if err != nil {
		t.Fatalf("ParseToken(tampered) error = %v", err)
	}

	// Verification MUST fail because header was tampered
	err = ks.VerifyWithKid([]byte(tamperedSigningInput), tamperedKid, tamperedSig)
	if err == nil {
		t.Fatal("expected signature verification to fail after header tampering, but it succeeded")
	}
	if err != jwks.ErrSignature {
		t.Errorf("expected ErrSignature, got %v", err)
	}
}

func splitTestToken(s string) []string {
	var parts []string
	start := 0
	for i := range len(s) {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func TestRevokeOldest_LastKey(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	err := ks.RevokeOldest()
	if err == nil {
		t.Error("expected error when revoking last key")
	}
}

func TestCurrentKid(t *testing.T) {
	ks, _ := jwks.NewKeyStore()
	kid1 := ks.CurrentKid()
	if kid1 == "" {
		t.Error("expected non-empty kid")
	}
	_, _ = ks.Rotate()
	kid2 := ks.CurrentKid()
	if kid1 == kid2 {
		t.Error("expected different kid after rotation")
	}
}
