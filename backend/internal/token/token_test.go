package token_test

import (
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/token"
)

func newTestSigner() *token.Signer {
	return token.NewSigner("test-secret-at-least-32-chars-long!!", "http://localhost:4444")
}

func TestIssue_And_Verify(t *testing.T) {
	s := newTestSigner()
	tok, err := s.Issue("user-123", "demo-client", "test-nonce")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
	if claims.Audience != "demo-client" {
		t.Errorf("Audience = %q, want %q", claims.Audience, "demo-client")
	}
	if claims.Nonce != "test-nonce" {
		t.Errorf("Nonce = %q, want %q", claims.Nonce, "test-nonce")
	}
	if claims.Issuer != "http://localhost:4444" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "http://localhost:4444")
	}
}

func TestIssue_WithoutNonce(t *testing.T) {
	s := newTestSigner()
	tok, err := s.Issue("user-123", "demo-client", "")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims.Nonce != "" {
		t.Errorf("Nonce = %q, want empty", claims.Nonce)
	}
}

func TestVerify_InvalidToken(t *testing.T) {
	s := newTestSigner()
	_, err := s.Verify("not-a-valid-token")
	if err != token.ErrInvalidToken {
		t.Errorf("Verify() error = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_InvalidSignature(t *testing.T) {
	s := newTestSigner()
	tok, _ := s.Issue("user-123", "demo-client", "nonce")

	// Tamper with signature
	_, err := s.Verify(tok + "tampered")
	if err != token.ErrInvalidSig {
		t.Errorf("Verify() error = %v, want ErrInvalidSig", err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	s1 := newTestSigner()
	s2 := token.NewSigner("different-secret-also-32-chars!!!!", "http://localhost:4444")

	tok, _ := s1.Issue("user-123", "demo-client", "nonce")
	_, err := s2.Verify(tok)
	if err != token.ErrInvalidSig {
		t.Errorf("Verify() error = %v, want ErrInvalidSig", err)
	}
}

func TestVerifyWithNonce_Match(t *testing.T) {
	s := newTestSigner()
	tok, _ := s.Issue("user-123", "demo-client", "expected-nonce")

	claims, err := s.VerifyWithNonce(tok, "expected-nonce")
	if err != nil {
		t.Fatalf("VerifyWithNonce() error = %v", err)
	}
	if claims.Nonce != "expected-nonce" {
		t.Errorf("Nonce = %q, want %q", claims.Nonce, "expected-nonce")
	}
}

func TestVerifyWithNonce_Mismatch(t *testing.T) {
	s := newTestSigner()
	tok, _ := s.Issue("user-123", "demo-client", "original-nonce")

	_, err := s.VerifyWithNonce(tok, "different-nonce")
	if err != token.ErrNonceMismatch {
		t.Errorf("VerifyWithNonce() error = %v, want ErrNonceMismatch", err)
	}
}
