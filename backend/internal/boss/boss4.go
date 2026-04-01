package boss

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/jwks"
)

// Boss4Handler handles the JWKS Rotation Failure boss stage.
type Boss4Handler struct {
	keyStore *jwks.KeyStore
	cache    *jwks.Cache
}

// Boss4RotateRequest configures rotation behavior.
type Boss4RotateRequest struct {
	RevokeOldKey bool `json:"revoke_old_key"`
}

// Boss4SignRequest signs a test payload.
type Boss4SignRequest struct {
	Payload   string `json:"payload"`
	UseOldKey bool   `json:"use_old_key"`
}

// Boss4VerifyRequest verifies a signed token.
type Boss4VerifyRequest struct {
	Token string `json:"token"`
}

// Boss4CacheConfigRequest configures the cache strategy.
type Boss4CacheConfigRequest struct {
	SmartMode bool `json:"smart_mode"`
}

// NewBoss4Handler creates a new Boss4Handler.
func NewBoss4Handler() *Boss4Handler {
	ks, err := jwks.NewKeyStore()
	if err != nil {
		panic("failed to create key store: " + err.Error())
	}

	cache := jwks.NewCache(time.Hour, false) // Default: dumb cache
	cache.Fetch(ks.GetJWKS())

	return &Boss4Handler{
		keyStore: ks,
		cache:    cache,
	}
}

// JWKS returns the current JWKS endpoint response.
func (h *Boss4Handler) JWKS(w http.ResponseWriter, _ *http.Request) {
	jwkSet := h.keyStore.GetJWKS()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jwks":        jwkSet,
		"cache_state": h.cache.Info(),
		"current_kid": h.keyStore.CurrentKid(),
	})
}

// Rotate triggers a key rotation.
func (h *Boss4Handler) Rotate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss4RotateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	oldKid := h.keyStore.CurrentKid()

	newKey, err := h.keyStore.Rotate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BossResult{
			Success: false,
			Message: "Key rotation failed: " + err.Error(),
		})
		return
	}

	if req.RevokeOldKey {
		if err := h.keyStore.RevokeOldest(); err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"new_kid": newKey.Kid,
				"old_kid": oldKid,
				"revoked": false,
				"message": "Key rotated but old key could not be revoked: " + err.Error(),
				"warning": "Premature revocation can cause outages for tokens signed with the old key.",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"new_kid": newKey.Kid,
			"old_kid": oldKid,
			"revoked": true,
			"message": "Key rotated AND old key revoked immediately. Tokens signed with the old key will fail verification!",
			"explanation": "Revoking the old key immediately causes a service disruption. " +
				"Tokens signed before rotation but not yet verified will fail. " +
				"Best practice: keep old keys active during an overlap period.",
			"rfc_link": "https://datatracker.ietf.org/doc/html/rfc7517",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"new_kid": newKey.Kid,
		"old_kid": oldKid,
		"revoked": false,
		"message": "Key rotated. Both old and new keys are active (overlap period).",
	})
}

// Sign creates a signed token.
func (h *Boss4Handler) Sign(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss4SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	payload := req.Payload
	if payload == "" {
		payload = `{"sub":"demo-user","iat":` + time.Now().Format("2006") + `}`
	}

	payloadBytes := []byte(payload)

	if req.UseOldKey {
		// Get the JWKS and use the first (oldest) key
		jwkSet := h.keyStore.GetJWKS()
		if len(jwkSet.Keys) < 2 {
			writeJSON(w, http.StatusBadRequest, BossResult{
				Success: false,
				Message: "No old key available. Rotate first to create multiple keys.",
			})
			return
		}
		oldKid := jwkSet.Keys[0].Kid
		signer := func(data []byte) (string, error) {
			return h.keyStore.SignWithKid(data, oldKid)
		}
		tok, err := jwks.MarshalToken(payloadBytes, oldKid, signer)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, BossResult{
				Success: false,
				Message: "Signing failed: " + err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"token":   tok,
			"kid":     oldKid,
			"message": "Token signed with OLD key.",
		})
		return
	}

	currentKid := h.keyStore.CurrentKid()
	signer := func(data []byte) (string, error) {
		return h.keyStore.SignWithKid(data, currentKid)
	}
	tok, err := jwks.MarshalToken(payloadBytes, currentKid, signer)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BossResult{
			Success: false,
			Message: "Signing failed: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   tok,
		"kid":     currentKid,
		"message": "Token signed with current key.",
	})
}

// Verify verifies a signed token using the cache.
func (h *Boss4Handler) Verify(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss4VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Token == "" {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Token is required",
		})
		return
	}

	kid, payload, signingInput, sig, err := jwks.ParseToken(req.Token)
	if err != nil {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Invalid token format: " + err.Error(),
		})
		return
	}

	// Try cache first
	_, found := h.cache.Lookup(kid)

	if !found {
		if h.cache.IsSmart() {
			// Smart cache: kid not found -> re-fetch JWKS
			h.cache.Fetch(h.keyStore.GetJWKS())
			_, found = h.cache.Lookup(kid)

			if !found {
				writeJSON(w, http.StatusOK, BossResult{
					Success: false,
					Message: "Key not found even after JWKS refresh. The key may have been revoked.",
					Explanation: "Smart cache refreshed the JWKS when an unknown kid was encountered, " +
						"but the key still wasn't found. This means the key has been revoked from the server.",
				})
				return
			}

			// Verify against actual key store using the full signing input (header.payload)
			if verifyErr := h.keyStore.VerifyWithKid([]byte(signingInput), kid, sig); verifyErr != nil {
				writeJSON(w, http.StatusOK, BossResult{
					Success: false,
					Message: "Signature verification failed: " + verifyErr.Error(),
				})
				return
			}

			writeJSON(w, http.StatusOK, BossResult{
				Success:  true,
				Defeated: true,
				Message:  "Boss defeated! Token verified after smart cache refresh. The kid-based re-fetch pattern works!",
				Explanation: "When the cache didn't contain the key matching the token's kid, " +
					"the smart cache immediately re-fetched the JWKS from the server. " +
					"This handles key rotation gracefully without waiting for cache TTL expiry.\n\n" +
					"Best practice: TTL-based cache + kid-mismatch re-fetch + overlap period.",
				RFCLink: "https://datatracker.ietf.org/doc/html/rfc7517",
			})
			return
		}

		// Dumb cache: kid not found -> fail
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Verification failed: key not found in cached JWKS. The cache is stale and does not refresh on kid mismatch.",
			Explanation: "The token was signed with a key (kid=" + kid + ") that is not in the cached JWKS. " +
				"Without kid-based cache refresh, the client cannot verify tokens signed with new keys " +
				"until the cache TTL expires. This causes a service disruption during key rotation.\n\n" +
				"Fix: Enable smart cache mode to re-fetch JWKS when an unknown kid is encountered.",
			RFCLink: "https://datatracker.ietf.org/doc/html/rfc7517",
		})
		return
	}

	// Key found in cache — verify signature using the full signing input (header.payload)
	if verifyErr := h.keyStore.VerifyWithKid([]byte(signingInput), kid, sig); verifyErr != nil {
		writeJSON(w, http.StatusOK, BossResult{
			Success: false,
			Message: "Signature verification failed: " + verifyErr.Error(),
			Explanation: "The key was found in the JWKS but the signature does not match. " +
				"This could indicate token tampering.",
		})
		return
	}

	_ = payload
	writeJSON(w, http.StatusOK, BossResult{
		Success: true,
		Message: "Token verified successfully using cached key (kid=" + kid + ").",
	})
}

// ConfigureCache sets the cache strategy.
func (h *Boss4Handler) ConfigureCache(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req Boss4CacheConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BossResult{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	h.cache.SetSmart(req.SmartMode)

	mode := "stale (TTL-only)"
	if req.SmartMode {
		mode = "smart (TTL + kid-based refresh)"
	}

	// Refresh cache with current JWKS
	h.cache.Fetch(h.keyStore.GetJWKS())

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"cache_mode":  mode,
		"cache_state": h.cache.Info(),
		"message":     "Cache strategy updated to: " + mode,
	})
}
