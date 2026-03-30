package boss_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/boss"
)

func TestBoss3_Authorize_WithoutNonce(t *testing.T) {
	h := boss.NewBoss3Handler()

	body := mustJSON(t, boss.Boss3AuthorizeRequest{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3000/callback",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/3/authorize", body)
	rec := httptest.NewRecorder()
	h.Authorize(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Code == "" {
		t.Error("expected authorization code")
	}
	if result.RFCLink == "" {
		t.Error("expected RFC link for vulnerable flow")
	}
}

func TestBoss3_Authorize_WithNonce(t *testing.T) {
	h := boss.NewBoss3Handler()

	body := mustJSON(t, boss.Boss3AuthorizeRequest{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3000/callback",
		Nonce:       "test-nonce",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/3/authorize", body)
	rec := httptest.NewRecorder()
	h.Authorize(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestBoss3_FullFlow_WithoutNonce_ReplaySucceeds(t *testing.T) {
	h := boss.NewBoss3Handler()

	// Authorize without nonce
	authBody := mustJSON(t, boss.Boss3AuthorizeRequest{
		ClientID: "test-client",
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Get token
	tokenBody := mustJSON(t, boss.Boss3TokenRequest{
		Code: authResult.Code,
	})
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/token", tokenBody)
	tokenRec := httptest.NewRecorder()
	h.Token(tokenRec, tokenReq)

	var tokenResult map[string]interface{}
	if err := json.NewDecoder(tokenRec.Body).Decode(&tokenResult); err != nil {
		t.Fatalf("decode token response: %v", err)
	}

	idToken, ok := tokenResult["id_token"].(string)
	if !ok || idToken == "" {
		t.Fatal("expected id_token in response")
	}

	// Replay the token
	replayBody := mustJSON(t, boss.Boss3ReplayRequest{
		IDToken: idToken,
	})
	replayReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/replay", replayBody)
	replayRec := httptest.NewRecorder()
	h.Replay(replayRec, replayReq)

	var replayResult boss.BossResult
	mustDecode(t, replayRec, &replayResult)

	if !replayResult.Success {
		t.Error("expected replay attack to succeed without nonce")
	}
}

func TestBoss3_FullFlow_WithNonce_BossDefeated(t *testing.T) {
	h := boss.NewBoss3Handler()

	// Generate nonce
	genReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/generate-nonce", nil)
	genRec := httptest.NewRecorder()
	h.GenerateNonce(genRec, genReq)

	var nonceResp map[string]string
	if err := json.NewDecoder(genRec.Body).Decode(&nonceResp); err != nil {
		t.Fatalf("decode nonce response: %v", err)
	}
	generatedNonce := nonceResp["nonce"]

	// Authorize with nonce
	authBody := mustJSON(t, boss.Boss3AuthorizeRequest{
		ClientID: "test-client",
		Nonce:    generatedNonce,
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Exchange with nonce validation
	tokenBody := mustJSON(t, boss.Boss3TokenRequest{
		Code:          authResult.Code,
		ExpectedNonce: generatedNonce,
	})
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/token", tokenBody)
	tokenRec := httptest.NewRecorder()
	h.Token(tokenRec, tokenReq)

	var tokenResult map[string]interface{}
	if err := json.NewDecoder(tokenRec.Body).Decode(&tokenResult); err != nil {
		t.Fatalf("decode token response: %v", err)
	}

	defeated, _ := tokenResult["defeated"].(bool)
	if !defeated {
		t.Errorf("expected boss to be defeated, got: %v", tokenResult["message"])
	}
}

func TestBoss3_Replay_Blocked_NonceMismatch(t *testing.T) {
	h := boss.NewBoss3Handler()

	// Generate nonce and authorize
	genReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/generate-nonce", nil)
	genRec := httptest.NewRecorder()
	h.GenerateNonce(genRec, genReq)

	var nonceResp map[string]string
	if err := json.NewDecoder(genRec.Body).Decode(&nonceResp); err != nil {
		t.Fatalf("decode nonce: %v", err)
	}

	authBody := mustJSON(t, boss.Boss3AuthorizeRequest{
		ClientID: "test-client",
		Nonce:    nonceResp["nonce"],
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	tokenBody := mustJSON(t, boss.Boss3TokenRequest{
		Code: authResult.Code,
	})
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/token", tokenBody)
	tokenRec := httptest.NewRecorder()
	h.Token(tokenRec, tokenReq)

	var tokenResult map[string]interface{}
	if err := json.NewDecoder(tokenRec.Body).Decode(&tokenResult); err != nil {
		t.Fatalf("decode token: %v", err)
	}

	idToken := tokenResult["id_token"].(string)

	// Try replay with different nonce
	replayBody := mustJSON(t, boss.Boss3ReplayRequest{
		IDToken:       idToken,
		ExpectedNonce: "different-nonce",
	})
	replayReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/replay", replayBody)
	replayRec := httptest.NewRecorder()
	h.Replay(replayRec, replayReq)

	var replayResult boss.BossResult
	mustDecode(t, replayRec, &replayResult)

	if replayResult.Success {
		t.Error("expected replay to be blocked with nonce mismatch")
	}
}

func TestBoss3_Verify_Success(t *testing.T) {
	h := boss.NewBoss3Handler()

	// Generate nonce and authorize to get a valid token
	genReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/generate-nonce", nil)
	genRec := httptest.NewRecorder()
	h.GenerateNonce(genRec, genReq)

	var nonceResp map[string]string
	if err := json.NewDecoder(genRec.Body).Decode(&nonceResp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	authBody := mustJSON(t, boss.Boss3AuthorizeRequest{
		ClientID: "test-client",
		Nonce:    nonceResp["nonce"],
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	tokenBody := mustJSON(t, boss.Boss3TokenRequest{Code: authResult.Code})
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/token", tokenBody)
	tokenRec := httptest.NewRecorder()
	h.Token(tokenRec, tokenReq)

	var tokenResult map[string]interface{}
	if err := json.NewDecoder(tokenRec.Body).Decode(&tokenResult); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Verify
	verifyBody := mustJSON(t, boss.Boss3VerifyRequest{
		IDToken:       tokenResult["id_token"].(string),
		Nonce:         nonceResp["nonce"],
		ExpectedNonce: nonceResp["nonce"],
	})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/boss/3/verify", verifyBody)
	verifyRec := httptest.NewRecorder()
	h.Verify(verifyRec, verifyReq)

	var verifyResult boss.BossResult
	mustDecode(t, verifyRec, &verifyResult)

	if !verifyResult.Defeated {
		t.Errorf("expected boss to be defeated, message: %s", verifyResult.Message)
	}
}

func TestBoss3_Verify_MissingNonce(t *testing.T) {
	h := boss.NewBoss3Handler()

	body := mustJSON(t, boss.Boss3VerifyRequest{
		IDToken: "some.token.value",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/3/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if result.Defeated {
		t.Error("boss should not be defeated with missing nonce")
	}
}

func TestBoss3_Token_InvalidCode(t *testing.T) {
	h := boss.NewBoss3Handler()

	body := mustJSON(t, boss.Boss3TokenRequest{
		Code: "nonexistent",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/3/token", body)
	rec := httptest.NewRecorder()
	h.Token(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
