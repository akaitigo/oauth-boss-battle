package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

const defaultTTL = 10 * time.Minute

// RPSession represents a session at a Relying Party.
type RPSession struct {
	SID       string    `json:"sid"`
	RPID      string    `json:"rp_id"`
	RPName    string    `json:"rp_name"`
	UserID    string    `json:"user_id"`
	Active    bool      `json:"active"`
	LogoutBy  string    `json:"logout_by,omitempty"` // "frontchannel", "backchannel", ""
	CreatedAt time.Time `json:"created_at"`
}

// Store manages sessions across multiple RPs.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*RPSession // sid -> session
	bySID    map[string][]string   // sid_prefix -> all session sids for that login
	ttl      time.Duration
}

// NewStore creates a new session store with TTL-based eviction.
func NewStore() *Store {
	s := &Store{
		sessions: make(map[string]*RPSession),
		bySID:    make(map[string][]string),
		ttl:      defaultTTL,
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
	for sid, session := range s.sessions {
		if now.Sub(session.CreatedAt) > s.ttl {
			delete(s.sessions, sid)
		}
	}
	// Clean up bySID entries that have no remaining sessions
	for prefix, sids := range s.bySID {
		remaining := sids[:0]
		for _, sid := range sids {
			if _, exists := s.sessions[sid]; exists {
				remaining = append(remaining, sid)
			}
		}
		if len(remaining) == 0 {
			delete(s.bySID, prefix)
		} else {
			s.bySID[prefix] = remaining
		}
	}
}

// CreateSessions creates sessions for a user across multiple RPs.
// Returns the shared session ID prefix.
func (s *Store) CreateSessions(userID string, rps []RPInfo) (string, []*RPSession, error) {
	sidPrefix, err := generateSID()
	if err != nil {
		return "", nil, fmt.Errorf("generate sid: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var sessions []*RPSession
	var sids []string

	for i, rp := range rps {
		sid := fmt.Sprintf("%s-%d", sidPrefix, i)
		session := &RPSession{
			SID:       sid,
			RPID:      rp.ID,
			RPName:    rp.Name,
			UserID:    userID,
			Active:    true,
			CreatedAt: now,
		}
		s.sessions[sid] = session
		sessions = append(sessions, session)
		sids = append(sids, sid)
	}

	s.bySID[sidPrefix] = sids
	return sidPrefix, sessions, nil
}

// GetSessions returns all sessions for a given SID prefix.
func (s *Store) GetSessions(sidPrefix string) []*RPSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sids, exists := s.bySID[sidPrefix]
	if !exists {
		return nil
	}

	var sessions []*RPSession
	for _, sid := range sids {
		if session, ok := s.sessions[sid]; ok {
			cpy := *session
			sessions = append(sessions, &cpy)
		}
	}
	return sessions
}

// FrontChannelLogout simulates front-channel logout (unreliable).
// Returns which RPs were "notified" (in practice, some fail silently).
func (s *Store) FrontChannelLogout(sidPrefix string) ([]*RPSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sids, exists := s.bySID[sidPrefix]
	if !exists {
		return nil, ErrSessionNotFound
	}

	var sessions []*RPSession
	for i, sid := range sids {
		session, ok := s.sessions[sid]
		if !ok {
			continue
		}

		cpy := *session
		// Simulate: first RP logout succeeds, others fail silently
		// (simulating iframe/redirect failures in real browsers)
		if i == 0 {
			session.Active = false
			session.LogoutBy = "frontchannel"
			cpy = *session
		}
		// Other RPs: session remains active (iframe failure)
		sessions = append(sessions, &cpy)
	}

	return sessions, nil
}

// BackChannelLogout invalidates all sessions for a SID prefix via server-to-server call.
func (s *Store) BackChannelLogout(sidPrefix string) ([]*RPSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sids, exists := s.bySID[sidPrefix]
	if !exists {
		return nil, ErrSessionNotFound
	}

	var sessions []*RPSession
	for _, sid := range sids {
		session, ok := s.sessions[sid]
		if !ok {
			continue
		}

		session.Active = false
		session.LogoutBy = "backchannel"
		cpy := *session
		sessions = append(sessions, &cpy)
	}

	return sessions, nil
}

// ActiveCount returns the number of active sessions for a SID prefix.
func (s *Store) ActiveCount(sidPrefix string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sids, exists := s.bySID[sidPrefix]
	if !exists {
		return 0
	}

	count := 0
	for _, sid := range sids {
		if session, ok := s.sessions[sid]; ok && session.Active {
			count++
		}
	}
	return count
}

// RPInfo describes a Relying Party.
type RPInfo struct {
	ID   string
	Name string
}

func generateSID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
