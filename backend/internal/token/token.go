package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken  = errors.New("invalid token format")
	ErrTokenExpired  = errors.New("token has expired")
	ErrInvalidSig    = errors.New("invalid token signature")
	ErrNonceMismatch = errors.New("nonce in token does not match expected nonce")
)

// Claims represents the ID Token claims.
type Claims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	Nonce     string `json:"nonce,omitempty"`
}

// Signer creates and verifies simplified JWT-like ID Tokens using HS256.
// This is for educational purposes only; production should use RS256.
type Signer struct {
	secret []byte
	issuer string
}

// NewSigner creates a new token signer.
func NewSigner(secret, issuer string) *Signer {
	return &Signer{
		secret: []byte(secret),
		issuer: issuer,
	}
}

// Issue creates a signed ID Token with the given claims.
func (s *Signer) Issue(subject, audience, n string) (string, error) {
	now := time.Now()
	claims := Claims{
		Issuer:    s.issuer,
		Subject:   subject,
		Audience:  audience,
		ExpiresAt: now.Add(time.Hour).Unix(),
		IssuedAt:  now.Unix(),
		Nonce:     n,
	}

	header := base64Encode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payloadB64 := base64Encode(payload)

	sigInput := header + "." + payloadB64
	sig := s.sign(sigInput)

	return sigInput + "." + sig, nil
}

// Verify parses and verifies an ID Token, returning its claims.
func (s *Signer) Verify(tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	sigInput := parts[0] + "." + parts[1]
	expectedSig := s.sign(sigInput)
	if parts[2] != expectedSig {
		return nil, ErrInvalidSig
	}

	payload, err := base64Decode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

// VerifyWithNonce verifies a token and checks the nonce claim.
func (s *Signer) VerifyWithNonce(tokenStr, expectedNonce string) (*Claims, error) {
	claims, err := s.Verify(tokenStr)
	if err != nil {
		return nil, err
	}

	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return nil, ErrNonceMismatch
	}

	return claims, nil
}

func (s *Signer) sign(input string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(input))
	return base64Encode(mac.Sum(nil))
}

func base64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
