package boss_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/boss"
)

func TestBoss4_JWKS(t *testing.T) {
	h := boss.NewBoss4Handler()

	req := httptest.NewRequest(http.MethodPost, "/api/boss/4/jwks", nil)
	rec := httptest.NewRecorder()
	h.JWKS(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result["current_kid"] == nil || result["current_kid"] == "" {
		t.Error("expected current_kid in response")
	}
}

func TestBoss4_SignAndVerify(t *testing.T) {
	h := boss.NewBoss4Handler()

	// Sign a token
	signBody := mustJSON(t, boss.Boss4SignRequest{
		Payload: `{"sub":"test-user"}`,
	})
	signReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/sign", signBody)
	signRec := httptest.NewRecorder()
	h.Sign(signRec, signReq)

	var signResult map[string]interface{}
	if err := json.NewDecoder(signRec.Body).Decode(&signResult); err != nil {
		t.Fatalf("decode: %v", err)
	}

	tok, ok := signResult["token"].(string)
	if !ok || tok == "" {
		t.Fatal("expected token in sign response")
	}

	// Verify the token
	verifyBody := mustJSON(t, boss.Boss4VerifyRequest{Token: tok})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/verify", verifyBody)
	verifyRec := httptest.NewRecorder()
	h.Verify(verifyRec, verifyReq)

	var verifyResult boss.BossResult
	mustDecode(t, verifyRec, &verifyResult)

	if !verifyResult.Success {
		t.Errorf("expected verification success, got message: %s", verifyResult.Message)
	}
}

func TestBoss4_Rotation_StaleCache_Fails(t *testing.T) {
	h := boss.NewBoss4Handler()

	// Rotate key (old key remains active in key store but cache is stale)
	rotateBody := mustJSON(t, boss.Boss4RotateRequest{RevokeOldKey: false})
	rotateReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/rotate", rotateBody)
	rotateRec := httptest.NewRecorder()
	h.Rotate(rotateRec, rotateReq)

	// Sign with NEW key
	signBody := mustJSON(t, boss.Boss4SignRequest{Payload: `{"sub":"test-user"}`})
	signReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/sign", signBody)
	signRec := httptest.NewRecorder()
	h.Sign(signRec, signReq)

	var signResult map[string]interface{}
	if err := json.NewDecoder(signRec.Body).Decode(&signResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	tok := signResult["token"].(string)

	// Verify with STALE cache (dumb mode) — should fail because cache doesn't have new key
	verifyBody := mustJSON(t, boss.Boss4VerifyRequest{Token: tok})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/verify", verifyBody)
	verifyRec := httptest.NewRecorder()
	h.Verify(verifyRec, verifyReq)

	var verifyResult boss.BossResult
	mustDecode(t, verifyRec, &verifyResult)

	if verifyResult.Success {
		t.Error("expected verification to fail with stale cache")
	}
}

func TestBoss4_SmartCache_BossDefeated(t *testing.T) {
	h := boss.NewBoss4Handler()

	// Enable smart cache
	configBody := mustJSON(t, boss.Boss4CacheConfigRequest{SmartMode: true})
	configReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/configure-cache", configBody)
	configRec := httptest.NewRecorder()
	h.ConfigureCache(configRec, configReq)

	// Rotate key
	rotateBody := mustJSON(t, boss.Boss4RotateRequest{RevokeOldKey: false})
	rotateReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/rotate", rotateBody)
	rotateRec := httptest.NewRecorder()
	h.Rotate(rotateRec, rotateReq)

	// Sign with new key
	signBody := mustJSON(t, boss.Boss4SignRequest{Payload: `{"sub":"test-user"}`})
	signReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/sign", signBody)
	signRec := httptest.NewRecorder()
	h.Sign(signRec, signReq)

	var signResult map[string]interface{}
	if err := json.NewDecoder(signRec.Body).Decode(&signResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	tok := signResult["token"].(string)

	// Verify with smart cache — should succeed after re-fetch
	verifyBody := mustJSON(t, boss.Boss4VerifyRequest{Token: tok})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/verify", verifyBody)
	verifyRec := httptest.NewRecorder()
	h.Verify(verifyRec, verifyReq)

	var verifyResult boss.BossResult
	mustDecode(t, verifyRec, &verifyResult)

	if !verifyResult.Defeated {
		t.Errorf("expected boss to be defeated with smart cache, got message: %s", verifyResult.Message)
	}
}

func TestBoss4_Rotation_WithRevoke_OldTokenFails(t *testing.T) {
	h := boss.NewBoss4Handler()

	// Enable smart cache for proper testing
	configBody := mustJSON(t, boss.Boss4CacheConfigRequest{SmartMode: true})
	configReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/configure-cache", configBody)
	configRec := httptest.NewRecorder()
	h.ConfigureCache(configRec, configReq)

	// Sign with current key
	signBody := mustJSON(t, boss.Boss4SignRequest{Payload: `{"sub":"test-user"}`})
	signReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/sign", signBody)
	signRec := httptest.NewRecorder()
	h.Sign(signRec, signReq)

	var signResult map[string]interface{}
	if err := json.NewDecoder(signRec.Body).Decode(&signResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	tok := signResult["token"].(string)

	// Rotate AND revoke
	rotateBody := mustJSON(t, boss.Boss4RotateRequest{RevokeOldKey: true})
	rotateReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/rotate", rotateBody)
	rotateRec := httptest.NewRecorder()
	h.Rotate(rotateRec, rotateReq)

	// Verify old token — should fail because old key was revoked
	verifyBody := mustJSON(t, boss.Boss4VerifyRequest{Token: tok})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/verify", verifyBody)
	verifyRec := httptest.NewRecorder()
	h.Verify(verifyRec, verifyReq)

	var verifyResult boss.BossResult
	mustDecode(t, verifyRec, &verifyResult)

	if verifyResult.Success {
		t.Error("expected verification to fail after old key revocation")
	}
}

func TestBoss4_Sign_OldKey(t *testing.T) {
	h := boss.NewBoss4Handler()

	// Rotate to have two keys
	rotateBody := mustJSON(t, boss.Boss4RotateRequest{RevokeOldKey: false})
	rotateReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/rotate", rotateBody)
	rotateRec := httptest.NewRecorder()
	h.Rotate(rotateRec, rotateReq)

	// Sign with old key
	signBody := mustJSON(t, boss.Boss4SignRequest{
		Payload:   `{"sub":"old-key-test"}`,
		UseOldKey: true,
	})
	signReq := httptest.NewRequest(http.MethodPost, "/api/boss/4/sign", signBody)
	signRec := httptest.NewRecorder()
	h.Sign(signRec, signReq)

	var signResult map[string]interface{}
	if err := json.NewDecoder(signRec.Body).Decode(&signResult); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if signResult["token"] == nil || signResult["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestBoss4_Verify_EmptyToken(t *testing.T) {
	h := boss.NewBoss4Handler()

	body := mustJSON(t, boss.Boss4VerifyRequest{Token: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/boss/4/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
