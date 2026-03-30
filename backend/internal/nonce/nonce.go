package nonce

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrMissingNonce  = errors.New("nonce is required for ID Token replay protection")
	ErrNonceConsumed = errors.New("nonce has already been consumed; possible replay attack")
	ErrNonceUnknown  = errors.New("nonce was not issued by this server")
)

// Store manages nonce parameters for ID Token replay protection.
type Store struct {
	mu     sync.RWMutex
	nonces map[string]bool // nonce -> consumed
}

// NewStore creates a new nonce store.
func NewStore() *Store {
	return &Store{
		nonces: make(map[string]bool),
	}
}

// Generate creates a cryptographically random nonce and registers it.
func (s *Store) Generate() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	n := hex.EncodeToString(b)

	s.mu.Lock()
	s.nonces[n] = false
	s.mu.Unlock()

	return n, nil
}

// Consume validates and consumes a nonce (one-time use).
func (s *Store) Consume(n string) error {
	if n == "" {
		return ErrMissingNonce
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	consumed, exists := s.nonces[n]
	if !exists {
		return ErrNonceUnknown
	}
	if consumed {
		return ErrNonceConsumed
	}

	s.nonces[n] = true
	return nil
}

// IsConsumed checks if a nonce has been consumed.
func (s *Store) IsConsumed(n string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	consumed, exists := s.nonces[n]
	return exists && consumed
}
