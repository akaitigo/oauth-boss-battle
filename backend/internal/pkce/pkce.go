package pkce

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// MethodS256 is the SHA-256 code challenge method (required by RFC 7636).
	MethodS256 = "S256"
	// MethodPlain is the plain code challenge method (not recommended).
	MethodPlain = "plain"

	verifierMinLen = 43
	verifierMaxLen = 128
)

var (
	ErrMissingVerifier  = errors.New("code_verifier is required")
	ErrMissingChallenge = errors.New("code_challenge is required")
	ErrInvalidMethod    = errors.New("unsupported code_challenge_method; use S256")
	ErrVerifierLength   = fmt.Errorf("code_verifier must be %d-%d characters", verifierMinLen, verifierMaxLen)
	ErrChallengeFailed  = errors.New("code_verifier does not match code_challenge")
)

// GenerateVerifier creates a cryptographically random code_verifier.
func GenerateVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// S256Challenge computes the S256 code_challenge from a code_verifier.
func S256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// Verify checks that a code_verifier matches the stored code_challenge.
func Verify(method, challenge, verifier string) error {
	if verifier == "" {
		return ErrMissingVerifier
	}
	if challenge == "" {
		return ErrMissingChallenge
	}

	if len(verifier) < verifierMinLen || len(verifier) > verifierMaxLen {
		return ErrVerifierLength
	}

	switch method {
	case MethodS256:
		computed := S256Challenge(verifier)
		if computed != challenge {
			return ErrChallengeFailed
		}
		return nil
	case MethodPlain:
		if verifier != challenge {
			return ErrChallengeFailed
		}
		return nil
	default:
		return ErrInvalidMethod
	}
}
