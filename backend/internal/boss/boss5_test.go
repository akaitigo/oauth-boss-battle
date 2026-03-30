package boss_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/oauth-boss-battle/backend/internal/boss"
)

func TestBoss5_Login(t *testing.T) {
	h := boss.NewBoss5Handler()

	body := mustJSON(t, boss.Boss5LoginRequest{UserID: "test-user"})
	req := httptest.NewRequest(http.MethodPost, "/api/boss/5/login", body)
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result["sid_prefix"] == nil || result["sid_prefix"] == "" {
		t.Error("expected sid_prefix in response")
	}

	sessions, ok := result["sessions"].([]interface{})
	if !ok || len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %v", result["sessions"])
	}
}

func TestBoss5_Sessions(t *testing.T) {
	h := boss.NewBoss5Handler()

	// Login
	loginBody := mustJSON(t, boss.Boss5LoginRequest{UserID: "test-user"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/login", loginBody)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	sidPrefix := loginResult["sid_prefix"].(string)

	// Get sessions
	sessBody := mustJSON(t, boss.Boss5SessionsRequest{SIDPrefix: sidPrefix})
	sessReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/sessions", sessBody)
	sessRec := httptest.NewRecorder()
	h.Sessions(sessRec, sessReq)

	var sessResult map[string]interface{}
	if err := json.NewDecoder(sessRec.Body).Decode(&sessResult); err != nil {
		t.Fatalf("decode: %v", err)
	}

	activeCount := int(sessResult["active_count"].(float64))
	if activeCount != 3 {
		t.Errorf("expected 3 active sessions, got %d", activeCount)
	}
}

func TestBoss5_FrontChannelLogout_SessionsRemain(t *testing.T) {
	h := boss.NewBoss5Handler()

	// Login
	loginBody := mustJSON(t, boss.Boss5LoginRequest{})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/login", loginBody)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	sidPrefix := loginResult["sid_prefix"].(string)

	// Front-channel logout
	logoutBody := mustJSON(t, boss.Boss5LogoutRequest{SIDPrefix: sidPrefix})
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/logout-frontchannel", logoutBody)
	logoutRec := httptest.NewRecorder()
	h.LogoutFrontChannel(logoutRec, logoutReq)

	var logoutResult map[string]interface{}
	if err := json.NewDecoder(logoutRec.Body).Decode(&logoutResult); err != nil {
		t.Fatalf("decode: %v", err)
	}

	activeCount := int(logoutResult["active_count"].(float64))
	if activeCount == 0 {
		t.Error("expected some sessions to remain active after front-channel logout")
	}
	if activeCount == 3 {
		t.Error("expected at least one session to be logged out")
	}
}

func TestBoss5_BackChannelLogout_AllSessionsInvalidated(t *testing.T) {
	h := boss.NewBoss5Handler()

	// Login
	loginBody := mustJSON(t, boss.Boss5LoginRequest{})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/login", loginBody)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	sidPrefix := loginResult["sid_prefix"].(string)

	// Back-channel logout
	logoutBody := mustJSON(t, boss.Boss5LogoutRequest{SIDPrefix: sidPrefix})
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/logout-backchannel", logoutBody)
	logoutRec := httptest.NewRecorder()
	h.LogoutBackChannel(logoutRec, logoutReq)

	var logoutResult map[string]interface{}
	if err := json.NewDecoder(logoutRec.Body).Decode(&logoutResult); err != nil {
		t.Fatalf("decode: %v", err)
	}

	activeCount := int(logoutResult["active_count"].(float64))
	if activeCount != 0 {
		t.Errorf("expected 0 active sessions, got %d", activeCount)
	}

	defeated, _ := logoutResult["defeated"].(bool)
	if !defeated {
		t.Error("expected boss to be defeated")
	}
}

func TestBoss5_Verify_Success(t *testing.T) {
	h := boss.NewBoss5Handler()

	// Login
	loginBody := mustJSON(t, boss.Boss5LoginRequest{})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/login", loginBody)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	sidPrefix := loginResult["sid_prefix"].(string)

	// Back-channel logout
	logoutBody := mustJSON(t, boss.Boss5LogoutRequest{SIDPrefix: sidPrefix})
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/logout-backchannel", logoutBody)
	logoutRec := httptest.NewRecorder()
	h.LogoutBackChannel(logoutRec, logoutReq)

	// Verify
	verifyBody := mustJSON(t, boss.Boss5VerifyRequest{
		SIDPrefix:       sidPrefix,
		UsedBackChannel: true,
	})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/verify", verifyBody)
	verifyRec := httptest.NewRecorder()
	h.Verify(verifyRec, verifyReq)

	var verifyResult boss.BossResult
	mustDecode(t, verifyRec, &verifyResult)

	if !verifyResult.Defeated {
		t.Errorf("expected boss to be defeated, got: %s", verifyResult.Message)
	}
}

func TestBoss5_Verify_FrontChannelOnly_Fails(t *testing.T) {
	h := boss.NewBoss5Handler()

	// Login
	loginBody := mustJSON(t, boss.Boss5LoginRequest{})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/login", loginBody)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	sidPrefix := loginResult["sid_prefix"].(string)

	// Front-channel logout only
	logoutBody := mustJSON(t, boss.Boss5LogoutRequest{SIDPrefix: sidPrefix})
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/logout-frontchannel", logoutBody)
	logoutRec := httptest.NewRecorder()
	h.LogoutFrontChannel(logoutRec, logoutReq)

	// Verify
	verifyBody := mustJSON(t, boss.Boss5VerifyRequest{
		SIDPrefix:       sidPrefix,
		UsedBackChannel: false,
	})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/boss/5/verify", verifyBody)
	verifyRec := httptest.NewRecorder()
	h.Verify(verifyRec, verifyReq)

	var verifyResult boss.BossResult
	mustDecode(t, verifyRec, &verifyResult)

	if verifyResult.Defeated {
		t.Error("boss should not be defeated with front-channel only")
	}
}

func TestBoss5_Verify_MissingSID(t *testing.T) {
	h := boss.NewBoss5Handler()

	body := mustJSON(t, boss.Boss5VerifyRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/boss/5/verify", body)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	var result boss.BossResult
	mustDecode(t, rec, &result)

	if result.Defeated {
		t.Error("boss should not be defeated with missing SID")
	}
}
