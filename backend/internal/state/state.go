package state

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrMissingState  = errors.New("state parameter is required for CSRF protection")
	ErrStateMismatch = errors.New("state parameter does not match; possible CSRF attack")
	ErrStateExpired  = errors.New("state parameter has already been consumed or expired")
)

const (
	defaultTTL    = 10 * time.Minute
	defaultMaxCap = 10000
)

var ErrStoreFull = errors.New("state store is at capacity; try again later")

type stateEntry struct {
	consumed  bool
	createdAt time.Time
}

// Store manages state parameters for CSRF protection in the OAuth flow.
type Store struct {
	mu     sync.RWMutex
	states map[string]*stateEntry
	ttl    time.Duration
	maxCap int
}

// NewStore creates a new state store with TTL-based eviction and a max entry cap.
func NewStore() *Store {
	s := &Store{
		states: make(map[string]*stateEntry),
		ttl:    defaultTTL,
		maxCap: defaultMaxCap,
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
	for k, entry := range s.states {
		if now.Sub(entry.createdAt) > s.ttl {
			delete(s.states, k)
		}
	}
}

// Generate creates a cryptographically random state parameter and stores it.
func (s *Store) Generate() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	state := hex.EncodeToString(b)

	s.mu.Lock()
	if len(s.states) >= s.maxCap {
		s.mu.Unlock()
		// Try eviction first, then re-check
		s.evict()
		s.mu.Lock()
		if len(s.states) >= s.maxCap {
			s.mu.Unlock()
			return "", ErrStoreFull
		}
	}
	s.states[state] = &stateEntry{consumed: false, createdAt: time.Now()}
	s.mu.Unlock()

	return state, nil
}

// Validate checks that the returned state matches an issued state and consumes it.
func (s *Store) Validate(expected, actual string) error {
	if expected == "" || actual == "" {
		return ErrMissingState
	}

	if expected != actual {
		return ErrStateMismatch
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.states[expected]
	if !exists {
		return ErrStateExpired
	}
	if entry.consumed {
		return ErrStateExpired
	}
	if time.Since(entry.createdAt) > s.ttl {
		delete(s.states, expected)
		return ErrStateExpired
	}

	// Mark as consumed (one-time use)
	entry.consumed = true
	return nil
}

// IsIssued checks if a state was issued (without consuming it).
func (s *Store) IsIssued(state string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.states[state]
	if !exists {
		return false
	}
	return time.Since(entry.createdAt) <= s.ttl
}
