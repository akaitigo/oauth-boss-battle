package pkce_test

import (
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/pkce"
)

func TestGenerateVerifier(t *testing.T) {
	v, err := pkce.GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier() error = %v", err)
	}
	if len(v) < 43 {
		t.Errorf("GenerateVerifier() length = %d, want >= 43", len(v))
	}
}

func TestS256Challenge(t *testing.T) {
	// Known test vector from RFC 7636 Appendix B
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	got := pkce.S256Challenge(verifier)
	if got != want {
		t.Errorf("S256Challenge() = %q, want %q", got, want)
	}
}

func TestVerify_S256_Success(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := pkce.S256Challenge(verifier)

	if err := pkce.Verify(pkce.MethodS256, challenge, verifier); err != nil {
		t.Errorf("Verify() unexpected error = %v", err)
	}
}

func TestVerify_S256_Failure(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := pkce.S256Challenge(verifier)

	wrongVerifier := "wrong-verifier-that-is-at-least-43-characters-long-xxxxx"
	err := pkce.Verify(pkce.MethodS256, challenge, wrongVerifier)
	if err == nil {
		t.Error("Verify() expected error for wrong verifier")
	}
}

func TestVerify_MissingVerifier(t *testing.T) {
	err := pkce.Verify(pkce.MethodS256, "some-challenge", "")
	if err != pkce.ErrMissingVerifier {
		t.Errorf("Verify() error = %v, want ErrMissingVerifier", err)
	}
}

func TestVerify_MissingChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	err := pkce.Verify(pkce.MethodS256, "", verifier)
	if err != pkce.ErrMissingChallenge {
		t.Errorf("Verify() error = %v, want ErrMissingChallenge", err)
	}
}

func TestVerify_InvalidMethod(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	err := pkce.Verify("unsupported", "some-challenge", verifier)
	if err != pkce.ErrInvalidMethod {
		t.Errorf("Verify() error = %v, want ErrInvalidMethod", err)
	}
}

func TestVerify_VerifierTooShort(t *testing.T) {
	err := pkce.Verify(pkce.MethodS256, "some-challenge", "short")
	if err != pkce.ErrVerifierLength {
		t.Errorf("Verify() error = %v, want ErrVerifierLength", err)
	}
}
