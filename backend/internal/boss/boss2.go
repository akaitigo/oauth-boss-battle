package boss

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/state"
)

// Boss2Handler handles the State Mismatch (CSRF) boss stage.
type Boss2Handler struct {
	stateStore *state.Store
	mu         sync.RWMutex
	sessions   map[string]*csrfSession // authorization_code -> session
}

type csrfSession struct {
	State string `json:"state"`
}

// Boss2AuthorizeRequest is the authorization request for Boss 2.
type Boss2AuthorizeRequest struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

// Boss2CallbackRequest simulates the callback with state validation.
type Boss2CallbackRequest struct {
	Code          string `json:"code"`
	ReturnedState string `json:"returned_state"`
	OriginalState string `json:"original_state"`
}

// Boss2AttackRequest simulates the CSRF attack scenario.
type Boss2AttackRequest struct {
	AttackerCode string `json:"attacker_code"`
	VictimState  string `json:"victim_state"`
}

// Boss2VerifyRequest is the verify request for boss defeat check.
type Boss2VerifyRequest struct {
	State         string `json:"state"`
	ReturnedState string `json:"returned_state"`
}

// NewBoss2Handler creates a new Boss2Handler.
func NewBoss2Handler() *Boss2Handler {
	return &Boss2Handler{
		stateStore: state.NewStore(),
		sessions:   make(map[string]*csrfSession),
	}
}

// Authorize simulates the /authorize endpoint.
func (h *Boss2Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss2AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	code, err := generateCode()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BossResult{
			Success: false,
			Message: "Failed to generate authorization code",
		})
		return
	}

	session := &csrfSession{
		State: req.State,
	}

	h.mu.Lock()
	h.sessions[code] = session
	h.mu.Unlock()

	if req.State == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: true,
			Code:    code,
			Message: "Authorization code issued WITHOUT state parameter. This request is vulnerable to CSRF attacks!",
			Explanation: "Without a state parameter, an attacker can craft a malicious URL with their own authorization code. " +
				"When the victim clicks this URL, their account gets linked to the attacker's identity. " +
				"This is a Cross-Site Request Forgery (CSRF) attack on the OAuth callback.",
			RFCLink: "https://datatracker.ietf.org/doc/html/rfc6749#section-10.12",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success: true,
		Code:    code,
		Message: "Authorization code issued with state parameter for CSRF protection.",
	})
}

// Callback simulates the OAuth callback with state validation.
func (h *Boss2Handler) Callback(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss2CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	h.mu.RLock()
	session, exists := h.sessions[req.Code]
	h.mu.RUnlock()

	if !exists {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid or expired authorization code",
		})
		return
	}

	h.mu.Lock()
	delete(h.sessions, req.Code)
	h.mu.Unlock()

	// No state was used
	if session.State == "" && req.OriginalState == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: true,
			Message: "Callback accepted WITHOUT state validation. CSRF attack succeeded! An attacker could have swapped their authorization code into this callback.",
			Explanation: "The OAuth flow completed without any CSRF protection. An attacker can: " +
				"1) Start an OAuth flow and get their own authorization code, " +
				"2) Craft a callback URL with their code, " +
				"3) Trick the victim into clicking it, " +
				"4) The victim's session is now linked to the attacker's account.",
			RFCLink: "https://datatracker.ietf.org/doc/html/rfc6749#section-10.12",
		})
		return
	}

	// State validation: use the server-side session state (not the client-supplied OriginalState)
	// to prevent bypass attacks where an attacker supplies their own state as both values.
	if err := h.stateStore.Validate(session.State, req.ReturnedState); err != nil {
		writeJSON(w, http.StatusOK, BossResult{
			Success:  true,
			Defeated: false,
			Message:  "State validation failed: " + err.Error() + ". CSRF attack was detected and blocked!",
			Explanation: "The state parameter mismatch indicates a potential CSRF attack. " +
				"The server correctly rejected the callback because the returned state " +
				"does not match the original state stored in the user's session.",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success:  true,
		Defeated: true,
		Message:  "Boss defeated! State validation succeeded. The CSRF attack was prevented.",
		Explanation: "By binding a cryptographically random state parameter to the user's session " +
			"and validating it on the callback, you prevent attackers from injecting their " +
			"authorization codes into a victim's OAuth flow. " +
			"The state parameter acts as a CSRF token for the OAuth authorization endpoint.",
		RFCLink: "https://datatracker.ietf.org/doc/html/rfc6749#section-10.12",
	})
}

// Attack simulates the attacker's perspective of the CSRF attack.
func (h *Boss2Handler) Attack(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss2AttackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.VictimState != "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Attack failed! The victim's session has a state parameter that the attacker cannot predict.",
			Explanation: "Since the attacker doesn't know the victim's state parameter, " +
				"they cannot craft a valid callback URL. The CSRF protection works!",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success: true,
		Message: "CSRF attack succeeded! The attacker's authorization code was accepted by the victim's browser because there was no state parameter to validate.",
		Explanation: "Attack flow:\n" +
			"1. Attacker starts OAuth flow, gets authorization code\n" +
			"2. Attacker crafts URL: /callback?code=ATTACKER_CODE\n" +
			"3. Victim clicks the malicious link\n" +
			"4. Victim's account is now linked to attacker's identity\n\n" +
			"This works because the callback endpoint has no way to verify " +
			"that the authorization code belongs to the current user's session.",
		RFCLink: "https://datatracker.ietf.org/doc/html/rfc6749#section-10.12",
	})
}

// Verify checks the boss defeat condition.
func (h *Boss2Handler) Verify(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss2VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.State == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Missing state parameter. CSRF protection requires a random state bound to the user's session.",
			Explanation: "To defeat this boss, generate a cryptographically random state parameter, " +
				"store it in the session, send it with the authorization request, " +
				"and validate it when the callback is received.",
		})
		return
	}

	if req.ReturnedState == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Missing returned_state. The callback must include the state parameter for validation.",
		})
		return
	}

	if req.State != req.ReturnedState {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "State mismatch! The returned state does not match the original. This correctly detects a potential CSRF attack.",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success:  true,
		Defeated: true,
		Message:  "Boss 2 Defeated! You correctly implemented state-based CSRF protection.",
		Explanation: "The state parameter must be:\n" +
			"1. Cryptographically random (unpredictable)\n" +
			"2. Bound to the user's session\n" +
			"3. Validated on the callback (exact match)\n" +
			"4. One-time use (consumed after validation)\n\n" +
			"This prevents CSRF attacks on the OAuth authorization flow " +
			"as described in RFC 6749 Section 10.12.",
		RFCLink: "https://datatracker.ietf.org/doc/html/rfc6749#section-10.12",
	})
}

// GenerateState generates a state parameter and registers it in the store.
func (h *Boss2Handler) GenerateState(w http.ResponseWriter, r *http.Request) {
	s, err := h.stateStore.Generate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BossResult{
			Success: false,
			Message: "Failed to generate state parameter",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"state": s,
	})
}
