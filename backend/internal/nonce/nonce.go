package nonce

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrMissingNonce  = errors.New("nonce is required for ID Token replay protection")
	ErrNonceConsumed = errors.New("nonce has already been consumed; possible replay attack")
	ErrNonceUnknown  = errors.New("nonce was not issued by this server")
)

const defaultTTL = 10 * time.Minute

type nonceEntry struct {
	consumed  bool
	createdAt time.Time
}

// Store manages nonce parameters for ID Token replay protection.
type Store struct {
	mu     sync.RWMutex
	nonces map[string]*nonceEntry
	ttl    time.Duration
}

// NewStore creates a new nonce store with TTL-based eviction.
func NewStore() *Store {
	s := &Store{
		nonces: make(map[string]*nonceEntry),
		ttl:    defaultTTL,
	}
	go s.evictLoop()
	return s
}

// evictLoop periodically removes expired entries.
func (s *Store) evictLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.evict()
	}
}

func (s *Store) evict() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, entry := range s.nonces {
		if now.Sub(entry.createdAt) > s.ttl {
			delete(s.nonces, k)
		}
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
	s.nonces[n] = &nonceEntry{consumed: false, createdAt: time.Now()}
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

	entry, exists := s.nonces[n]
	if !exists {
		return ErrNonceUnknown
	}
	if entry.consumed {
		return ErrNonceConsumed
	}
	if time.Since(entry.createdAt) > s.ttl {
		delete(s.nonces, n)
		return ErrNonceUnknown
	}

	entry.consumed = true
	return nil
}

// IsConsumed checks if a nonce has been consumed.
func (s *Store) IsConsumed(n string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.nonces[n]
	return exists && entry.consumed
}
