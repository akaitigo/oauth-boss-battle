package boss

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/pkce"
)

// Boss1Handler handles the PKCE Missing Attack boss stage.
// It simulates an authorization server that can detect missing PKCE parameters.
type Boss1Handler struct {
	mu    sync.RWMutex
	codes map[string]*authSession // authorization_code -> session
}

type authSession struct {
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

// AuthorizeRequest is the request body for the authorize endpoint.
type AuthorizeRequest struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

// TokenRequest is the request body for the token endpoint.
type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier"`
}

// VerifyRequest is the request body for the verify (boss defeat check) endpoint.
type VerifyRequest struct {
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	CodeVerifier        string `json:"code_verifier"`
}

// BossResult is the common response structure for boss stages.
type BossResult struct {
	Success     bool   `json:"success"`
	Defeated    bool   `json:"defeated"`
	Message     string `json:"message"`
	Explanation string `json:"explanation,omitempty"`
	Code        string `json:"code,omitempty"`
	RFCLink     string `json:"rfc_link,omitempty"`
}

// NewBoss1Handler creates a new Boss1Handler.
func NewBoss1Handler() *Boss1Handler {
	return &Boss1Handler{
		codes: make(map[string]*authSession),
	}
}

// Authorize simulates the /authorize endpoint.
// Without PKCE parameters, it returns an authorization code but flags the vulnerability.
func (h *Boss1Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizeRequest
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

	session := &authSession{
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}

	h.mu.Lock()
	h.codes[code] = session
	h.mu.Unlock()

	if req.CodeChallenge == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: true,
			Code:    code,
			Message: "Authorization code issued WITHOUT PKCE. An attacker who intercepts this code can exchange it for tokens!",
			Explanation: "The authorization request did not include code_challenge. " +
				"Without PKCE, an attacker who intercepts the authorization code " +
				"(e.g., via a malicious app on the same device) can exchange it for tokens.",
			RFCLink: "https://datatracker.ietf.org/doc/html/rfc7636",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success: true,
		Code:    code,
		Message: "Authorization code issued with PKCE protection.",
	})
}

// Token simulates the /token endpoint.
// Without a valid code_verifier, the attack succeeds (bad outcome for the user).
func (h *Boss1Handler) Token(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	h.mu.RLock()
	session, exists := h.codes[req.Code]
	h.mu.RUnlock()

	if !exists {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid or expired authorization code",
		})
		return
	}

	// Clean up used code
	h.mu.Lock()
	delete(h.codes, req.Code)
	h.mu.Unlock()

	// No PKCE was used during authorization — attack succeeds
	if session.CodeChallenge == "" {
		if req.CodeVerifier == "" {
			writeJSON(w, http.StatusOK, BossResult{
				Success: true,
				Message: "Token issued! Attack succeeded — no PKCE protection. The intercepted authorization code was exchanged without proof of possession.",
				Explanation: "Without PKCE, any party with the authorization code can exchange it for tokens. " +
					"This is the core vulnerability that PKCE prevents.",
				RFCLink: "https://datatracker.ietf.org/doc/html/rfc7636",
			})
			return
		}
		writeJSON(w, http.StatusOK, BossResult{
			Success: true,
			Message: "Token issued. You sent a code_verifier, but the server had no code_challenge to verify against. PKCE must be enforced at authorization time.",
		})
		return
	}

	// PKCE was used — verify
	if err := pkce.Verify(session.CodeChallengeMethod, session.CodeChallenge, req.CodeVerifier); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success:     false,
			Message:     "Token exchange failed: " + err.Error(),
			Explanation: "The code_verifier does not match the code_challenge sent during authorization. The authorization code is now invalid.",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success:  true,
		Defeated: true,
		Message:  "Boss defeated! Token issued with valid PKCE verification. The authorization code was protected from interception attacks.",
		Explanation: "PKCE (Proof Key for Code Exchange) ensures that only the client that started " +
			"the authorization flow can complete it. Even if an attacker intercepts the authorization code, " +
			"they cannot exchange it without the original code_verifier.",
		RFCLink: "https://datatracker.ietf.org/doc/html/rfc7636",
	})
}

// Verify is a simplified check endpoint for the boss defeat condition.
func (h *Boss1Handler) Verify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.CodeChallenge == "" || req.CodeChallengeMethod == "" || req.CodeVerifier == "" {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Missing PKCE parameters. You need code_challenge, code_challenge_method, and code_verifier.",
			Explanation: "All three PKCE parameters are required: " +
				"code_challenge (sent at authorization), " +
				"code_challenge_method (S256 recommended), " +
				"and code_verifier (sent at token exchange).",
		})
		return
	}

	if err := pkce.Verify(req.CodeChallengeMethod, req.CodeChallenge, req.CodeVerifier); err != nil {
		writeJSON(w, http.StatusOK, BossResult{
			Success:     false,
			Message:     "PKCE verification failed: " + err.Error(),
			Explanation: "The code_verifier does not produce the expected code_challenge. Make sure you are using S256 method.",
		})
		return
	}

	writeJSON(w, http.StatusOK, BossResult{
		Success:  true,
		Defeated: true,
		Message:  "Boss 1 Defeated! You have correctly implemented PKCE.",
		Explanation: "By using PKCE with S256, you ensure that only the original client can exchange " +
			"the authorization code for tokens. This prevents authorization code interception attacks " +
			"described in RFC 7636.",
		RFCLink: "https://datatracker.ietf.org/doc/html/rfc7636",
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func generateCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
