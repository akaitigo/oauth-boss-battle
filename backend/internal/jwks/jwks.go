package jwks

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrKeyNotFound = errors.New("key with specified kid not found in JWKS")
	ErrSignature   = errors.New("token signature verification failed")
	ErrCacheStale  = errors.New("JWKS cache is stale; kid not found and cache not refreshed")
	ErrTokenFormat = errors.New("invalid signed token format")
)

// JWK represents a single JSON Web Key.
type JWK struct {
	KTY string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`

	privateKey *rsa.PrivateKey
	createdAt  time.Time
}

// JWKSet represents the JWKS endpoint response.
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// KeyStore manages JWKS keys with rotation support.
type KeyStore struct {
	mu       sync.RWMutex
	keys     []*JWK
	keyIndex int
}

// NewKeyStore creates a new key store with an initial key.
func NewKeyStore() (*KeyStore, error) {
	ks := &KeyStore{}
	if _, err := ks.Rotate(); err != nil {
		return nil, fmt.Errorf("generate initial key: %w", err)
	}
	return ks, nil
}

// Rotate generates a new key pair and adds it to the store.
func (ks *KeyStore) Rotate() (*JWK, error) {
	// Use 2048-bit RSA (smaller for speed in demo)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	ks.mu.Lock()
	ks.keyIndex++
	kid := fmt.Sprintf("key-%d", ks.keyIndex)
	ks.mu.Unlock()

	jwk := &JWK{
		KTY:        "RSA",
		Use:        "sig",
		Kid:        kid,
		Alg:        "RS256",
		N:          base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
		E:          base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
		privateKey: priv,
		createdAt:  time.Now(),
	}

	ks.mu.Lock()
	ks.keys = append(ks.keys, jwk)
	ks.mu.Unlock()

	return jwk, nil
}

// RevokeOldest removes the oldest key from the store.
func (ks *KeyStore) RevokeOldest() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if len(ks.keys) <= 1 {
		return errors.New("cannot revoke last key")
	}

	ks.keys = ks.keys[1:]
	return nil
}

// GetJWKS returns the current public key set.
func (ks *KeyStore) GetJWKS() JWKSet {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	publicKeys := make([]JWK, len(ks.keys))
	for i, k := range ks.keys {
		publicKeys[i] = JWK{
			KTY: k.KTY,
			Use: k.Use,
			Kid: k.Kid,
			Alg: k.Alg,
			N:   k.N,
			E:   k.E,
		}
	}
	return JWKSet{Keys: publicKeys}
}

// CurrentKid returns the kid of the most recent key.
func (ks *KeyStore) CurrentKid() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	if len(ks.keys) == 0 {
		return ""
	}
	return ks.keys[len(ks.keys)-1].Kid
}

// Sign signs data with the current (newest) key, returning kid + signature.
func (ks *KeyStore) Sign(data []byte) (kid string, signature string, err error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if len(ks.keys) == 0 {
		return "", "", errors.New("no keys available")
	}

	key := ks.keys[len(ks.keys)-1]
	hash := sha256.Sum256(data)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", "", fmt.Errorf("sign: %w", err)
	}

	return key.Kid, base64.RawURLEncoding.EncodeToString(sig), nil
}

// SignWithKid signs data with a specific key identified by kid.
func (ks *KeyStore) SignWithKid(data []byte, kid string) (string, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	for _, key := range ks.keys {
		if key.Kid == kid {
			hash := sha256.Sum256(data)
			sig, err := rsa.SignPKCS1v15(rand.Reader, key.privateKey, crypto.SHA256, hash[:])
			if err != nil {
				return "", fmt.Errorf("sign: %w", err)
			}
			return base64.RawURLEncoding.EncodeToString(sig), nil
		}
	}
	return "", ErrKeyNotFound
}

// VerifyWithKid verifies a signature against a specific key.
func (ks *KeyStore) VerifyWithKid(data []byte, kid, signature string) error {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	for _, key := range ks.keys {
		if key.Kid == kid {
			sigBytes, err := base64.RawURLEncoding.DecodeString(signature)
			if err != nil {
				return fmt.Errorf("decode signature: %w", err)
			}
			hash := sha256.Sum256(data)
			if err := rsa.VerifyPKCS1v15(&key.privateKey.PublicKey, crypto.SHA256, hash[:], sigBytes); err != nil {
				return ErrSignature
			}
			return nil
		}
	}
	return ErrKeyNotFound
}

// Cache simulates a JWKS client cache.
type Cache struct {
	mu      sync.RWMutex
	keys    JWKSet
	fetched time.Time
	ttl     time.Duration
	smart   bool // kid-based refresh enabled
}

// NewCache creates a new JWKS cache.
func NewCache(ttl time.Duration, smart bool) *Cache {
	return &Cache{
		ttl:   ttl,
		smart: smart,
	}
}

// Fetch fetches the JWKS and updates the cache.
func (c *Cache) Fetch(jwks JWKSet) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = jwks
	c.fetched = time.Now()
}

// Lookup finds a key by kid in the cache.
func (c *Cache) Lookup(kid string) (*JWK, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, k := range c.keys.Keys {
		if k.Kid == kid {
			return &k, true
		}
	}
	return nil, false
}

// IsStale returns true if the cache has expired.
func (c *Cache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.fetched) > c.ttl
}

// IsSmart returns whether kid-based refresh is enabled.
func (c *Cache) IsSmart() bool {
	return c.smart
}

// SetSmart enables or disables kid-based refresh.
func (c *Cache) SetSmart(smart bool) {
	c.smart = smart
}

// Info returns cache state information as JSON-serializable map.
func (c *Cache) Info() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	kids := make([]string, len(c.keys.Keys))
	for i, k := range c.keys.Keys {
		kids[i] = k.Kid
	}

	return map[string]interface{}{
		"cached_kids": kids,
		"fetched_at":  c.fetched.Format(time.RFC3339),
		"ttl_seconds": c.ttl.Seconds(),
		"is_stale":    time.Since(c.fetched) > c.ttl,
		"smart_mode":  c.smart,
	}
}

// MarshalToken creates a simple signed token with kid header.
func MarshalToken(payload []byte, kid, signature string) string {
	header, _ := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	})
	headerB64 := base64.RawURLEncoding.EncodeToString(header)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	return headerB64 + "." + payloadB64 + "." + signature
}

// ParseToken extracts kid, payload, and signature from a signed token.
func ParseToken(tokenStr string) (kid string, payload []byte, signature string, err error) {
	parts := splitToken(tokenStr)
	if len(parts) != 3 {
		return "", nil, "", ErrTokenFormat
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", nil, "", fmt.Errorf("decode header: %w", err)
	}

	var header map[string]string
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", nil, "", fmt.Errorf("parse header: %w", err)
	}

	payload, err = base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, "", fmt.Errorf("decode payload: %w", err)
	}

	return header["kid"], payload, parts[2], nil
}

func splitToken(s string) []string {
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
