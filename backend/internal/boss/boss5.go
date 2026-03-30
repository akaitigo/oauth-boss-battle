package boss

import (
	"encoding/json"
	"net/http"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/session"
)

// Boss5Handler handles the Logout Hell boss stage.
type Boss5Handler struct {
	sessionStore *session.Store
}

// Boss5LoginRequest simulates login and session creation.
type Boss5LoginRequest struct {
	UserID string `json:"user_id"`
}

// Boss5SessionsRequest queries session state.
type Boss5SessionsRequest struct {
	SIDPrefix string `json:"sid_prefix"`
}

// Boss5LogoutRequest triggers logout.
type Boss5LogoutRequest struct {
	SIDPrefix string `json:"sid_prefix"`
}

// Boss5VerifyRequest checks boss defeat condition.
type Boss5VerifyRequest struct {
	SIDPrefix       string `json:"sid_prefix"`
	UsedBackChannel bool   `json:"used_back_channel"`
}

var defaultRPs = []session.RPInfo{
	{ID: "rp-email", Name: "Email App"},
	{ID: "rp-calendar", Name: "Calendar App"},
	{ID: "rp-files", Name: "Files App"},
}

// NewBoss5Handler creates a new Boss5Handler.
func NewBoss5Handler() *Boss5Handler {
	return &Boss5Handler{
		sessionStore: session.NewStore(),
	}
}

// Login simulates user login and creates sessions at multiple RPs.
func (h *Boss5Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req Boss5LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	userID := req.UserID
	if userID == "" {
		userID = "demo-user"
	}

	sidPrefix, sessions, err := h.sessionStore.CreateSessions(userID, defaultRPs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BossResult{
			Success: false,
			Message: "Failed to create sessions: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"sid_prefix": sidPrefix,
		"sessions":   sessions,
		"message":    "User logged in. Sessions created at 3 Relying Parties.",
		"explanation": "The IdP has created sessions at multiple RPs. " +
			"Each RP has an independent session tied together by the session ID (sid). " +
			"When the user logs out from the IdP, ALL RP sessions should be invalidated.",
	})
}

// Sessions returns the current session state for all RPs.
func (h *Boss5Handler) Sessions(w http.ResponseWriter, r *http.Request) {
	var req Boss5SessionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	sessions := h.sessionStore.GetSessions(req.SIDPrefix)
	if sessions == nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "No sessions found for this SID",
		})
		return
	}

	activeCount := h.sessionStore.ActiveCount(req.SIDPrefix)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"sessions":     sessions,
		"active_count": activeCount,
		"total_count":  len(sessions),
	})
}

// LogoutFrontChannel simulates front-channel logout (unreliable).
func (h *Boss5Handler) LogoutFrontChannel(w http.ResponseWriter, r *http.Request) {
	var req Boss5LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	sessions, err := h.sessionStore.FrontChannelLogout(req.SIDPrefix)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Logout failed: " + err.Error(),
		})
		return
	}

	activeCount := h.sessionStore.ActiveCount(req.SIDPrefix)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"sessions":     sessions,
		"active_count": activeCount,
		"message": "Front-channel logout attempted. Only 1 of 3 RP sessions was invalidated! " +
			"The other sessions remain active due to iframe/redirect failures.",
		"explanation": "Front-channel logout uses browser-based mechanisms (iframes, redirects) to notify RPs. " +
			"These often fail silently due to:\n" +
			"- Browser privacy restrictions (third-party cookie blocking)\n" +
			"- Network timeouts on iframe requests\n" +
			"- Pop-up blockers\n" +
			"- CSP (Content Security Policy) restrictions\n\n" +
			"Result: Users think they logged out, but sessions at other RPs remain active. " +
			"An attacker with access to the browser can continue using those sessions.",
		"rfc_link": "https://openid.net/specs/openid-connect-session-1_0.html",
	})
}

// LogoutBackChannel simulates back-channel logout (reliable).
func (h *Boss5Handler) LogoutBackChannel(w http.ResponseWriter, r *http.Request) {
	var req Boss5LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	sessions, err := h.sessionStore.BackChannelLogout(req.SIDPrefix)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Logout failed: " + err.Error(),
		})
		return
	}

	activeCount := h.sessionStore.ActiveCount(req.SIDPrefix)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"defeated":     true,
		"sessions":     sessions,
		"active_count": activeCount,
		"message": "Boss defeated! Back-channel logout succeeded. All 3 RP sessions were invalidated " +
			"via server-to-server communication.",
		"explanation": "Back-channel logout sends a logout_token directly from the IdP to each RP's " +
			"back_channel_logout_uri via HTTP POST. This is reliable because:\n" +
			"- Server-to-server: not dependent on browser state\n" +
			"- Includes sid claim to identify the session to invalidate\n" +
			"- Each RP confirms logout with HTTP 200\n" +
			"- Failures can be retried by the IdP\n\n" +
			"Combined with RP-Initiated Logout for the current RP, this ensures complete session termination.",
		"rfc_link": "https://openid.net/specs/openid-connect-backchannel-1_0.html",
	})
}

// Verify checks the boss defeat condition.
func (h *Boss5Handler) Verify(w http.ResponseWriter, r *http.Request) {
	var req Boss5VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.SIDPrefix == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Missing SID prefix. Log in first to create sessions.",
		})
		return
	}

	activeCount := h.sessionStore.ActiveCount(req.SIDPrefix)

	if activeCount > 0 {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Not all sessions are invalidated. There are still active sessions at some RPs.",
			Explanation: "To defeat this boss, you need to ensure ALL RP sessions are " +
				"invalidated when the user logs out from the IdP. " +
				"Front-channel logout alone is not sufficient.",
		})
		return
	}

	if !req.UsedBackChannel {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Sessions are invalidated, but you did not use back-channel logout. " +
				"In production, front-channel logout is unreliable.",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success:  true,
		Defeated: true,
		Message:  "Boss 5 Defeated! You correctly implemented back-channel logout to ensure complete session termination.",
		Explanation: "Proper logout in federated identity requires:\n" +
			"1. RP-Initiated Logout: redirect user to IdP's end_session_endpoint\n" +
			"2. Back-Channel Logout: IdP sends logout_token to each RP's back_channel_logout_uri\n" +
			"3. Session ID (sid): binds IdP and RP sessions for targeted invalidation\n" +
			"4. Confirmation: each RP responds HTTP 200 to confirm logout\n\n" +
			"This prevents the 'Logout Hell' where users think they're logged out " +
			"but their sessions remain active at RPs.",
		RFCLink: "https://openid.net/specs/openid-connect-backchannel-1_0.html",
	})
}
