package boss_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/boss"
	"github.com/akaitigo/oauth-boss-battle/backend/internal/pkce"
)

func TestBoss1_Authorize_WithoutPKCE(t *testing.T) {
	h := boss.NewBoss1Handler()

	body := mustJSON(t, boss.AuthorizeRequest{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3000/callback",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/1/authorize", body)
	rec := httptest.NewRecorder()

	h.Authorize(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if !result.Success {
		t.Error("expected success=true for authorize without PKCE")
	}
	if result.Code == "" {
		t.Error("expected authorization code to be returned")
	}
	if result.RFCLink == "" {
		t.Error("expected RFC link in response")
	}
}

func TestBoss1_Authorize_WithPKCE(t *testing.T) {
	h := boss.NewBoss1Handler()

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := pkce.S256Challenge(verifier)

	body := mustJSON(t, boss.AuthorizeRequest{
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:3000/callback",
		CodeChallenge:       challenge,
		CodeChallengeMethod: pkce.MethodS256,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/1/authorize", body)
	rec := httptest.NewRecorder()

	h.Authorize(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Code == "" {
		t.Error("expected authorization code")
	}
}

func TestBoss1_FullFlow_WithoutPKCE_AttackSucceeds(t *testing.T) {
	h := boss.NewBoss1Handler()

	// Step 1: Authorize without PKCE
	authBody := mustJSON(t, boss.AuthorizeRequest{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3000/callback",
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/1/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Step 2: Exchange code without verifier (attacker scenario)
	tokenBody := mustJSON(t, boss.TokenRequest{
		GrantType:   "authorization_code",
		Code:        authResult.Code,
		RedirectURI: "http://localhost:3000/callback",
	})
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/boss/1/token", tokenBody)
	tokenRec := httptest.NewRecorder()
	h.Token(tokenRec, tokenReq)

	var tokenResult boss.BossResult
	mustDecode(t, tokenRec, &tokenResult)

	if !tokenResult.Success {
		t.Error("expected attack to succeed (token issued without PKCE)")
	}
	if tokenResult.Defeated {
		t.Error("boss should NOT be defeated when attack succeeds")
	}
}

func TestBoss1_FullFlow_WithPKCE_BossDefeated(t *testing.T) {
	h := boss.NewBoss1Handler()

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := pkce.S256Challenge(verifier)

	// Step 1: Authorize with PKCE
	authBody := mustJSON(t, boss.AuthorizeRequest{
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:3000/callback",
		CodeChallenge:       challenge,
		CodeChallengeMethod: pkce.MethodS256,
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/1/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Step 2: Exchange with valid verifier
	tokenBody := mustJSON(t, boss.TokenRequest{
		GrantType:    "authorization_code",
		Code:         authResult.Code,
		RedirectURI:  "http://localhost:3000/callback",
		CodeVerifier: verifier,
	})
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/boss/1/token", tokenBody)
	tokenRec := httptest.NewRecorder()
	h.Token(tokenRec, tokenReq)

	var tokenResult boss.BossResult
	mustDecode(t, tokenRec, &tokenResult)

	if !tokenResult.Success {
		t.Error("expected success=true")
	}
	if !tokenResult.Defeated {
		t.Error("expected boss to be defeated with correct PKCE")
	}
}

func TestBoss1_FullFlow_WithPKCE_WrongVerifier(t *testing.T) {
	h := boss.NewBoss1Handler()

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := pkce.S256Challenge(verifier)

	// Authorize with PKCE
	authBody := mustJSON(t, boss.AuthorizeRequest{
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:3000/callback",
		CodeChallenge:       challenge,
		CodeChallengeMethod: pkce.MethodS256,
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/1/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Exchange with WRONG verifier (attacker does not know the verifier)
	wrongVerifier := "wrong-verifier-that-is-at-least-43-characters-long-xxxxx"
	tokenBody := mustJSON(t, boss.TokenRequest{
		GrantType:    "authorization_code",
		Code:         authResult.Code,
		RedirectURI:  "http://localhost:3000/callback",
		CodeVerifier: wrongVerifier,
	})
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/boss/1/token", tokenBody)
	tokenRec := httptest.NewRecorder()
	h.Token(tokenRec, tokenReq)

	if tokenRec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", tokenRec.Code, http.StatusBadRequest)
	}
}

func TestBoss1_Verify_Success(t *testing.T) {
	h := boss.NewBoss1Handler()

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := pkce.S256Challenge(verifier)

	body := mustJSON(t, boss.VerifyRequest{
		CodeChallenge:       challenge,
		CodeChallengeMethod: pkce.MethodS256,
		CodeVerifier:        verifier,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/1/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if !result.Defeated {
		t.Error("expected boss to be defeated")
	}
}

func TestBoss1_Verify_MissingParams(t *testing.T) {
	h := boss.NewBoss1Handler()

	body := mustJSON(t, boss.VerifyRequest{})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/1/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if result.Defeated {
		t.Error("boss should not be defeated with missing params")
	}
}

func TestBoss1_Token_InvalidCode(t *testing.T) {
	h := boss.NewBoss1Handler()

	body := mustJSON(t, boss.TokenRequest{
		GrantType:   "authorization_code",
		Code:        "nonexistent-code",
		RedirectURI: "http://localhost:3000/callback",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/1/token", body)
	rec := httptest.NewRecorder()
	h.Token(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func mustJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		t.Fatalf("failed to encode JSON: %v", err)
	}
	return buf
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}
