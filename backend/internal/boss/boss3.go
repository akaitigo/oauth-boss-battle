package boss

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/nonce"
	"github.com/akaitigo/oauth-boss-battle/backend/internal/token"
)

// Boss3Handler handles the Nonce Replay Attack boss stage.
type Boss3Handler struct {
	nonceStore *nonce.Store
	signer     *token.Signer
	mu         sync.RWMutex
	sessions   map[string]*nonceSession // authorization_code -> session
}

type nonceSession struct {
	Nonce   string `json:"nonce"`
	Subject string `json:"subject"`
}

// Boss3AuthorizeRequest is the authorization request for Boss 3.
type Boss3AuthorizeRequest struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Nonce       string `json:"nonce"`
}

// Boss3TokenRequest is the token exchange request for Boss 3.
type Boss3TokenRequest struct {
	Code          string `json:"code"`
	RedirectURI   string `json:"redirect_uri"`
	ExpectedNonce string `json:"expected_nonce"`
}

// Boss3ReplayRequest simulates replaying a captured ID Token.
type Boss3ReplayRequest struct {
	IDToken       string `json:"id_token"`
	ExpectedNonce string `json:"expected_nonce"`
}

// Boss3VerifyRequest is the verify request for boss defeat check.
type Boss3VerifyRequest struct {
	IDToken       string `json:"id_token"`
	Nonce         string `json:"nonce"`
	ExpectedNonce string `json:"expected_nonce"`
}

// NewBoss3Handler creates a new Boss3Handler.
func NewBoss3Handler() *Boss3Handler {
	return &Boss3Handler{
		nonceStore: nonce.NewStore(),
		signer:     token.NewSigner("boss-battle-secret-key-for-demo-only!", "http://localhost:4444"),
		sessions:   make(map[string]*nonceSession),
	}
}

// GenerateNonce generates a nonce and registers it in the store.
func (h *Boss3Handler) GenerateNonce(w http.ResponseWriter, _ *http.Request) {
	n, err := h.nonceStore.Generate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BossResult{
			Success: false,
			Message: "Failed to generate nonce",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"nonce": n,
	})
}

// Authorize simulates the /authorize endpoint with optional nonce.
func (h *Boss3Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	var req Boss3AuthorizeRequest
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

	session := &nonceSession{
		Nonce:   req.Nonce,
		Subject: "demo-user",
	}

	h.mu.Lock()
	h.sessions[code] = session
	h.mu.Unlock()

	if req.Nonce == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: true,
			Code:    code,
			Message: "Authorization code issued WITHOUT nonce. The resulting ID Token will have no replay protection!",
			Explanation: "Without a nonce in the authorization request, the ID Token cannot be bound to this specific session. " +
				"An attacker who captures the ID Token can replay it in a different session.",
			RFCLink: "https://openid.net/specs/openid-connect-core-1_0.html#NonceNotes",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success: true,
		Code:    code,
		Message: "Authorization code issued with nonce for replay protection.",
	})
}

// Token simulates the /token endpoint, returning an ID Token.
func (h *Boss3Handler) Token(w http.ResponseWriter, r *http.Request) {
	var req Boss3TokenRequest
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

	idToken, err := h.signer.Issue(session.Subject, "demo-client", session.Nonce)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BossResult{
			Success: false,
			Message: "Failed to issue ID Token",
		})
		return
	}

	// If nonce was provided, validate and consume it
	if session.Nonce != "" && req.ExpectedNonce != "" {
		if err := h.nonceStore.Consume(req.ExpectedNonce); err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success":  true,
				"id_token": idToken,
				"message":  "ID Token issued but nonce validation failed: " + err.Error(),
				"warning":  "The nonce could not be consumed. This may indicate a replay.",
			})
			return
		}

		// Verify nonce in token matches expected
		_, verifyErr := h.signer.VerifyWithNonce(idToken, req.ExpectedNonce)
		if verifyErr != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success":  true,
				"id_token": idToken,
				"message":  "ID Token issued but nonce mismatch: " + verifyErr.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"defeated": true,
			"id_token": idToken,
			"message":  "Boss defeated! ID Token issued with nonce protection. The nonce was validated and consumed.",
			"explanation": "The nonce lifecycle is complete:\n" +
				"1. Generated: unique random nonce created\n" +
				"2. Stored: bound to user's session\n" +
				"3. Sent: included in authorization request\n" +
				"4. Embedded: IdP includes nonce in ID Token claims\n" +
				"5. Validated: RP verifies nonce matches session\n" +
				"6. Consumed: nonce marked as used (one-time)",
			"rfc_link": "https://openid.net/specs/openid-connect-core-1_0.html#NonceNotes",
		})
		return
	}

	if session.Nonce == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"id_token": idToken,
			"message":  "ID Token issued WITHOUT nonce claim. This token can be replayed!",
			"explanation": "The ID Token has no nonce claim, so there's no way to detect " +
				"if this exact token is being replayed from a different session.",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"id_token": idToken,
		"message":  "ID Token issued with nonce, but it was not validated at token exchange. Remember to verify the nonce!",
	})
}

// Replay simulates replaying a captured ID Token.
func (h *Boss3Handler) Replay(w http.ResponseWriter, r *http.Request) {
	var req Boss3ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	claims, err := h.signer.Verify(req.IDToken)
	if err != nil {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Replay failed: " + err.Error(),
		})
		return
	}

	// No nonce in token -> replay succeeds
	if claims.Nonce == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: true,
			Message: "Replay attack succeeded! The ID Token has no nonce, so there's no way to detect this is a replayed token.",
			Explanation: "Since the ID Token contains no nonce claim, the RP has no way to distinguish " +
				"between a legitimate token and a replayed one. The attacker can reuse this token " +
				"to impersonate the user in any session.",
			RFCLink: "https://openid.net/specs/openid-connect-core-1_0.html#NonceNotes",
		})
		return
	}

	// Nonce in token but no expected nonce -> replay may succeed
	if req.ExpectedNonce == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: true,
			Message: "Replay attack succeeded! The token has a nonce but the RP is not checking it.",
			Explanation: "Even though the IdP included a nonce in the ID Token, the RP is not validating it. " +
				"The nonce only protects against replay if the RP actually checks it against the session.",
		})
		return
	}

	// Nonce check: if token nonce != expected nonce, replay detected
	if claims.Nonce != req.ExpectedNonce {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Replay attack blocked! The nonce in the token does not match the expected nonce for this session.",
			Explanation: "The RP correctly detected the replay by comparing the nonce in the ID Token " +
				"against the nonce stored in the current session. Since they don't match, " +
				"this token was not issued for this session.",
		})
		return
	}

	// Nonce matches but check if consumed
	if h.nonceStore.IsConsumed(claims.Nonce) {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Replay attack blocked! The nonce has already been consumed.",
			Explanation: "The nonce was already used in a previous token exchange. " +
				"One-time-use semantics prevent replaying the same ID Token.",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success: true,
		Message: "Replay possible: nonce matches and has not been consumed yet.",
	})
}

// Verify checks the boss defeat condition.
func (h *Boss3Handler) Verify(w http.ResponseWriter, r *http.Request) {
	var req Boss3VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Nonce == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Missing nonce. ID Token replay protection requires a unique nonce per session.",
		})
		return
	}

	if req.IDToken == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Missing ID Token.",
		})
		return
	}

	claims, err := h.signer.VerifyWithNonce(req.IDToken, req.ExpectedNonce)
	if err != nil {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Verification failed: " + err.Error(),
		})
		return
	}

	if claims.Nonce == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "ID Token has no nonce claim. Replay protection is not active.",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success:  true,
		Defeated: true,
		Message:  "Boss 3 Defeated! You correctly implemented nonce-based ID Token replay protection.",
		Explanation: "The nonce lifecycle ensures ID Tokens are bound to specific sessions:\n" +
			"1. Generate: create unique random nonce\n" +
			"2. Store: bind to user's session\n" +
			"3. Send: include in authorization request\n" +
			"4. Embed: IdP includes nonce in ID Token\n" +
			"5. Validate: RP verifies nonce matches\n" +
			"6. Consume: mark as used (one-time)\n\n" +
			"This prevents token replay as described in OpenID Connect Core Section 11.5.",
		RFCLink: "https://openid.net/specs/openid-connect-core-1_0.html#NonceNotes",
	})
}
