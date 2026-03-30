package state

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrMissingState  = errors.New("state parameter is required for CSRF protection")
	ErrStateMismatch = errors.New("state parameter does not match; possible CSRF attack")
	ErrStateExpired  = errors.New("state parameter has already been consumed or expired")
)

// Store manages state parameters for CSRF protection in the OAuth flow.
type Store struct {
	mu     sync.RWMutex
	states map[string]bool // state -> consumed
}

// NewStore creates a new state store.
func NewStore() *Store {
	return &Store{
		states: make(map[string]bool),
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
	s.states[state] = false
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

	consumed, exists := s.states[expected]
	if !exists {
		return ErrStateExpired
	}
	if consumed {
		return ErrStateExpired
	}

	// Mark as consumed (one-time use)
	s.states[expected] = true
	return nil
}

// IsIssued checks if a state was issued (without consuming it).
func (s *Store) IsIssued(state string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.states[state]
	return exists
}
