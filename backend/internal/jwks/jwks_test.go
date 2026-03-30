package jwks_test

import (
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
	payload := []byte(`{"sub":"user-123"}`)
	kid := "test-key-1"
	sig := "test-signature"

	tok := jwks.MarshalToken(payload, kid, sig)

	parsedKid, parsedPayload, parsedSig, err := jwks.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if parsedKid != kid {
		t.Errorf("kid = %q, want %q", parsedKid, kid)
	}
	if string(parsedPayload) != string(payload) {
		t.Errorf("payload = %q, want %q", parsedPayload, payload)
	}
	if parsedSig != sig {
		t.Errorf("sig = %q, want %q", parsedSig, sig)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	_, _, _, err := jwks.ParseToken("not-a-token")
	if err != jwks.ErrTokenFormat {
		t.Errorf("expected ErrTokenFormat, got %v", err)
	}
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
