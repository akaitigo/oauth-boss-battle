package session_test

import (
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/session"
)

var testRPs = []session.RPInfo{
	{ID: "rp-1", Name: "Email App"},
	{ID: "rp-2", Name: "Calendar App"},
	{ID: "rp-3", Name: "Files App"},
}

func TestCreateSessions(t *testing.T) {
	store := session.NewStore()
	sid, sessions, err := store.CreateSessions("user-1", testRPs)
	if err != nil {
		t.Fatalf("CreateSessions() error = %v", err)
	}
	if sid == "" {
		t.Error("expected non-empty SID prefix")
	}
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
	for _, s := range sessions {
		if !s.Active {
			t.Error("expected new session to be active")
		}
	}
}

func TestGetSessions(t *testing.T) {
	store := session.NewStore()
	sid, _, _ := store.CreateSessions("user-1", testRPs)

	sessions := store.GetSessions(sid)
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestGetSessions_Unknown(t *testing.T) {
	store := session.NewStore()
	sessions := store.GetSessions("unknown")
	if sessions != nil {
		t.Error("expected nil for unknown SID")
	}
}

func TestFrontChannelLogout(t *testing.T) {
	store := session.NewStore()
	sid, _, _ := store.CreateSessions("user-1", testRPs)

	sessions, err := store.FrontChannelLogout(sid)
	if err != nil {
		t.Fatalf("FrontChannelLogout() error = %v", err)
	}

	// First RP should be logged out, others should remain active
	loggedOutCount := 0
	activeCount := 0
	for _, s := range sessions {
		if !s.Active {
			loggedOutCount++
		} else {
			activeCount++
		}
	}

	if loggedOutCount != 1 {
		t.Errorf("expected 1 logged out session, got %d", loggedOutCount)
	}
	if activeCount != 2 {
		t.Errorf("expected 2 active sessions remaining, got %d", activeCount)
	}
}

func TestFrontChannelLogout_Unknown(t *testing.T) {
	store := session.NewStore()
	_, err := store.FrontChannelLogout("unknown")
	if err != session.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestBackChannelLogout(t *testing.T) {
	store := session.NewStore()
	sid, _, _ := store.CreateSessions("user-1", testRPs)

	sessions, err := store.BackChannelLogout(sid)
	if err != nil {
		t.Fatalf("BackChannelLogout() error = %v", err)
	}

	for _, s := range sessions {
		if s.Active {
			t.Errorf("expected all sessions to be logged out, but %s is active", s.RPName)
		}
		if s.LogoutBy != "backchannel" {
			t.Errorf("expected logout_by=backchannel, got %q", s.LogoutBy)
		}
	}
}

func TestBackChannelLogout_Unknown(t *testing.T) {
	store := session.NewStore()
	_, err := store.BackChannelLogout("unknown")
	if err != session.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestActiveCount(t *testing.T) {
	store := session.NewStore()
	sid, _, _ := store.CreateSessions("user-1", testRPs)

	if count := store.ActiveCount(sid); count != 3 {
		t.Errorf("expected 3 active, got %d", count)
	}

	_, _ = store.FrontChannelLogout(sid)
	if count := store.ActiveCount(sid); count != 2 {
		t.Errorf("expected 2 active after front-channel, got %d", count)
	}

	_, _ = store.BackChannelLogout(sid)
	if count := store.ActiveCount(sid); count != 0 {
		t.Errorf("expected 0 active after back-channel, got %d", count)
	}
}

func TestActiveCount_Unknown(t *testing.T) {
	store := session.NewStore()
	if count := store.ActiveCount("unknown"); count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}
