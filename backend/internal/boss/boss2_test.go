package boss_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/boss"
)

func TestBoss2_Authorize_WithoutState(t *testing.T) {
	h := boss.NewBoss2Handler()

	body := mustJSON(t, boss.Boss2AuthorizeRequest{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3000/callback",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/2/authorize", body)
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

func TestBoss2_Authorize_WithState(t *testing.T) {
	h := boss.NewBoss2Handler()

	body := mustJSON(t, boss.Boss2AuthorizeRequest{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3000/callback",
		State:       "random-state-value",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/2/authorize", body)
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
}

func TestBoss2_Callback_WithoutState_AttackSucceeds(t *testing.T) {
	h := boss.NewBoss2Handler()

	// Authorize without state
	authBody := mustJSON(t, boss.Boss2AuthorizeRequest{
		ClientID: "test-client",
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Callback without state
	callbackBody := mustJSON(t, boss.Boss2CallbackRequest{
		Code: authResult.Code,
	})
	callbackReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/callback", callbackBody)
	callbackRec := httptest.NewRecorder()
	h.Callback(callbackRec, callbackReq)

	var callbackResult boss.BossResult
	mustDecode(t, callbackRec, &callbackResult)

	if !callbackResult.Success {
		t.Error("expected attack to succeed")
	}
	if callbackResult.Defeated {
		t.Error("boss should NOT be defeated when attack succeeds")
	}
}

func TestBoss2_FullFlow_WithState_BossDefeated(t *testing.T) {
	h := boss.NewBoss2Handler()

	// Generate state
	genReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/generate-state", nil)
	genRec := httptest.NewRecorder()
	h.GenerateState(genRec, genReq)

	var stateResp map[string]string
	if err := json.NewDecoder(genRec.Body).Decode(&stateResp); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	generatedState := stateResp["state"]

	// Authorize with state
	authBody := mustJSON(t, boss.Boss2AuthorizeRequest{
		ClientID: "test-client",
		State:    generatedState,
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Callback with matching state
	callbackBody := mustJSON(t, boss.Boss2CallbackRequest{
		Code:          authResult.Code,
		ReturnedState: generatedState,
		OriginalState: generatedState,
	})
	callbackReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/callback", callbackBody)
	callbackRec := httptest.NewRecorder()
	h.Callback(callbackRec, callbackReq)

	var callbackResult boss.BossResult
	mustDecode(t, callbackRec, &callbackResult)

	if !callbackResult.Defeated {
		t.Errorf("expected boss to be defeated, got message: %s", callbackResult.Message)
	}
}

func TestBoss2_Callback_CrossSessionStateBypass_Blocked(t *testing.T) {
	h := boss.NewBoss2Handler()

	// Attacker generates their own state via the state store
	genReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/generate-state", nil)
	genRec := httptest.NewRecorder()
	h.GenerateState(genRec, genReq)

	var attackerStateResp map[string]string
	if err := json.NewDecoder(genRec.Body).Decode(&attackerStateResp); err != nil {
		t.Fatalf("decode attacker state response: %v", err)
	}
	attackerState := attackerStateResp["state"]

	// Victim generates their own state
	genReq2 := httptest.NewRequest(http.MethodPost, "/api/boss/2/generate-state", nil)
	genRec2 := httptest.NewRecorder()
	h.GenerateState(genRec2, genReq2)

	var victimStateResp map[string]string
	if err := json.NewDecoder(genRec2.Body).Decode(&victimStateResp); err != nil {
		t.Fatalf("decode victim state response: %v", err)
	}
	victimState := victimStateResp["state"]

	// Victim authorizes with their own state (bound to session)
	authBody := mustJSON(t, boss.Boss2AuthorizeRequest{
		ClientID: "victim-client",
		State:    victimState,
	})
	authReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/authorize", authBody)
	authRec := httptest.NewRecorder()
	h.Authorize(authRec, authReq)

	var authResult boss.BossResult
	mustDecode(t, authRec, &authResult)

	// Attacker tries to bypass by supplying their own state as both
	// original_state and returned_state on the victim's authorization code.
	// Before the fix, this would succeed because req.OriginalState was trusted.
	callbackBody := mustJSON(t, boss.Boss2CallbackRequest{
		Code:          authResult.Code,
		ReturnedState: attackerState,
		OriginalState: attackerState,
	})
	callbackReq := httptest.NewRequest(http.MethodPost, "/api/boss/2/callback", callbackBody)
	callbackRec := httptest.NewRecorder()
	h.Callback(callbackRec, callbackReq)

	var callbackResult boss.BossResult
	mustDecode(t, callbackRec, &callbackResult)

	if callbackResult.Defeated {
		t.Error("expected boss NOT to be defeated when attacker uses cross-session state bypass")
	}
}

func TestBoss2_Attack_WithState_Blocked(t *testing.T) {
	h := boss.NewBoss2Handler()

	body := mustJSON(t, boss.Boss2AttackRequest{
		AttackerCode: "attacker-code",
		VictimState:  "victim-state-value",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/2/attack", body)
	rec := httptest.NewRecorder()
	h.Attack(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if result.Success {
		t.Error("expected attack to fail when victim has state")
	}
}

func TestBoss2_Attack_WithoutState_Succeeds(t *testing.T) {
	h := boss.NewBoss2Handler()

	body := mustJSON(t, boss.Boss2AttackRequest{
		AttackerCode: "attacker-code",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/2/attack", body)
	rec := httptest.NewRecorder()
	h.Attack(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if !result.Success {
		t.Error("expected attack to succeed without state")
	}
}

func TestBoss2_Verify_Success(t *testing.T) {
	h := boss.NewBoss2Handler()

	body := mustJSON(t, boss.Boss2VerifyRequest{
		State:         "matching-state",
		ReturnedState: "matching-state",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/2/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if !result.Defeated {
		t.Error("expected boss to be defeated")
	}
}

func TestBoss2_Verify_MissingState(t *testing.T) {
	h := boss.NewBoss2Handler()

	body := mustJSON(t, boss.Boss2VerifyRequest{})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/2/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if result.Defeated {
		t.Error("boss should not be defeated with missing state")
	}
}

func TestBoss2_Verify_Mismatch(t *testing.T) {
	h := boss.NewBoss2Handler()

	body := mustJSON(t, boss.Boss2VerifyRequest{
		State:         "original-state",
		ReturnedState: "different-state",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/boss/2/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if result.Defeated {
		t.Error("boss should not be defeated with mismatched state")
	}
}
