package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/auth"
	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

// bcrypt hash of "test123", same fixture the rest of the auth tests use.
const testHash = "$2a$10$H/SC9MUlyPBtcbU1Y8/EMu1vClnnTOUf8gK3jQ7WDv.8.5pwwTQ4W"

// newAuthServer builds a server with one user and no LAN bypass, so requests
// really go through the login flow. secret enrolls a second factor when set.
func newAuthServer(t *testing.T, secret string) *Server {
	t.Helper()
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{}
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	cfg.AgentSocket = "/nonexistent/ccsm-agent-test.sock"
	cfg.Users = []config.User{{Username: "luis", PasswordHash: testHash, TOTPSecret: secret}}
	return New(cfg, "", t.TempDir()+"/config.yaml")
}

func postJSON(srv *Server, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return serve(srv, req)
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad JSON %q: %v", w.Body.String(), err)
	}
	return body
}

func TestLoginWithTOTPRequiresSecondStep(t *testing.T) {
	secret, _ := auth.GenerateTOTPSecret()
	srv := newAuthServer(t, secret)

	w := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["ok"] != false || body["totp_required"] != true {
		t.Fatalf("expected totp_required, got %v", body)
	}
	pending := w.Result().Cookies()
	if len(pending) == 0 {
		t.Fatal("no pending cookie issued")
	}

	// The pending cookie must not open anything.
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	for _, c := range pending {
		req.AddCookie(c)
	}
	if w := serve(srv, req); w.Code != 401 {
		t.Errorf("pending session reached a protected endpoint: %d", w.Code)
	}

	// And /api/auth/status must say so, so a reload resumes at the code step.
	req = httptest.NewRequest("GET", "/api/auth/status", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	for _, c := range pending {
		req.AddCookie(c)
	}
	st := decode(t, serve(srv, req))
	if st["authenticated"] != false || st["totp_required"] != true {
		t.Errorf("status: %v", st)
	}

	// Correct code → full session.
	code, _ := auth.TOTPCode(secret, time.Now())
	w = postJSON(srv, "/api/auth/totp", `{"code":"`+code+`"}`, pending)
	if w.Code != 200 {
		t.Fatalf("totp: %d %s", w.Code, w.Body.String())
	}
	full := w.Result().Cookies()
	if len(full) == 0 {
		t.Fatal("no session cookie after TOTP")
	}

	req = httptest.NewRequest("GET", "/api/auth/status", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	for _, c := range full {
		req.AddCookie(c)
	}
	st = decode(t, serve(srv, req))
	if st["authenticated"] != true || st["username"] != "luis" {
		t.Errorf("status after TOTP: %v", st)
	}
}

func TestLoginWithoutTOTPUnchanged(t *testing.T) {
	srv := newAuthServer(t, "")
	body := decode(t, postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil))
	if body["ok"] != true {
		t.Errorf("a user without 2FA should log in directly: %v", body)
	}
}

func TestTOTPWrongCodeRejected(t *testing.T) {
	secret, _ := auth.GenerateTOTPSecret()
	srv := newAuthServer(t, secret)

	pending := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil).Result().Cookies()
	if w := postJSON(srv, "/api/auth/totp", `{"code":"000000"}`, pending); w.Code != 401 {
		t.Errorf("expected 401 for a wrong code, got %d", w.Code)
	}
}

func TestTOTPWithoutPendingSessionRejected(t *testing.T) {
	secret, _ := auth.GenerateTOTPSecret()
	srv := newAuthServer(t, secret)
	code, _ := auth.TOTPCode(secret, time.Now())

	if w := postJSON(srv, "/api/auth/totp", `{"code":"`+code+`"}`, nil); w.Code != 401 {
		t.Errorf("a valid code with no pending login was accepted: %d", w.Code)
	}
}

// A code stays valid for its whole 30 s window, so it must not be usable twice.
func TestTOTPCodeCannotBeReplayed(t *testing.T) {
	secret, _ := auth.GenerateTOTPSecret()
	srv := newAuthServer(t, secret)
	code, _ := auth.TOTPCode(secret, time.Now())

	pending := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil).Result().Cookies()
	if w := postJSON(srv, "/api/auth/totp", `{"code":"`+code+`"}`, pending); w.Code != 200 {
		t.Fatalf("first use: %d %s", w.Code, w.Body.String())
	}

	pending2 := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil).Result().Cookies()
	if w := postJSON(srv, "/api/auth/totp", `{"code":"`+code+`"}`, pending2); w.Code == 200 {
		t.Error("the same code was accepted a second time")
	}
}

func TestLoginRateLimit(t *testing.T) {
	srv := newAuthServer(t, "")

	for i := 0; i < auth.DefaultLoginMaxFails; i++ {
		if w := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"wrong"}`, nil); w.Code != 401 {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}
	// Blocked now — even the right password must not get through.
	w := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after %d failures, got %d", auth.DefaultLoginMaxFails, w.Code)
	}
}

// The TOTP step shares the counter: otherwise it would be an open window for
// brute-forcing six digits.
func TestTOTPStepIsRateLimited(t *testing.T) {
	secret, _ := auth.GenerateTOTPSecret()
	srv := newAuthServer(t, secret)
	pending := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil).Result().Cookies()

	for i := 0; i < auth.DefaultLoginMaxFails; i++ {
		postJSON(srv, "/api/auth/totp", `{"code":"000000"}`, pending)
	}
	code, _ := auth.TOTPCode(secret, time.Now())
	if w := postJSON(srv, "/api/auth/totp", `{"code":"`+code+`"}`, pending); w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

// The UI probes the LAN bypass on every page load by POSTing empty
// credentials. Counting those would let a reloading browser lock itself out.
func TestLANProbeDoesNotCountAsFailure(t *testing.T) {
	srv := newAuthServer(t, "")
	for i := 0; i < auth.DefaultLoginMaxFails*3; i++ {
		postJSON(srv, "/api/auth/login", `{"username":"","password":""}`, nil)
	}
	if w := postJSON(srv, "/api/auth/login", `{"username":"luis","password":"test123"}`, nil); w.Code != 200 {
		t.Errorf("the LAN probe locked the client out: %d", w.Code)
	}
}

func TestTOTPEnrollmentFlow(t *testing.T) {
	srv := newAuthServer(t, "")
	token, _ := srv.sessions.CreateSession("luis", false)
	cookie := &http.Cookie{Name: "ccsm_session", Value: token}

	// Start: a secret and its otpauth URI, not yet persisted.
	w := postJSON(srv, "/api/config/users/luis/totp", "", []*http.Cookie{cookie})
	if w.Code != 200 {
		t.Fatalf("enroll start: %d %s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	secret, _ := body["secret"].(string)
	if secret == "" || !strings.HasPrefix(body["uri"].(string), "otpauth://totp/CCSM:luis") {
		t.Fatalf("unexpected enrollment payload: %v", body)
	}
	srv.cfgMu.RLock()
	persisted := srv.cfg.Users[0].TOTPSecret
	srv.cfgMu.RUnlock()
	if persisted != "" {
		t.Error("the secret was persisted before being confirmed")
	}

	// A wrong code must not enable anything.
	req := httptest.NewRequest("PUT", "/api/config/users/luis/totp", strings.NewReader(`{"code":"000000"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(cookie)
	if w := serve(srv, req); w.Code != 400 {
		t.Errorf("a wrong confirmation code was accepted: %d", w.Code)
	}

	// The right one enables it.
	code, _ := auth.TOTPCode(secret, time.Now())
	req = httptest.NewRequest("PUT", "/api/config/users/luis/totp", strings.NewReader(`{"code":"`+code+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(cookie)
	if w := serve(srv, req); w.Code != 200 {
		t.Fatalf("enroll confirm: %d %s", w.Code, w.Body.String())
	}
	srv.cfgMu.RLock()
	persisted = srv.cfg.Users[0].TOTPSecret
	srv.cfgMu.RUnlock()
	if persisted != secret {
		t.Fatal("the confirmed secret was not persisted")
	}

	// The user list reports the state without ever exposing the secret.
	req = httptest.NewRequest("GET", "/api/config/users", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(cookie)
	w = serve(srv, req)
	if strings.Contains(w.Body.String(), secret) {
		t.Error("the TOTP secret leaked through /api/config/users")
	}
	var users []userInfo
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 1 || !users[0].TOTP {
		t.Errorf("user list does not report 2FA: %v", users)
	}

	// Disable.
	req = httptest.NewRequest("DELETE", "/api/config/users/luis/totp", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(cookie)
	if w := serve(srv, req); w.Code != 200 {
		t.Fatalf("disable: %d %s", w.Code, w.Body.String())
	}
	srv.cfgMu.RLock()
	persisted = srv.cfg.Users[0].TOTPSecret
	srv.cfgMu.RUnlock()
	if persisted != "" {
		t.Error("2FA was not disabled")
	}
}

func TestTOTPEnrollUnknownUser(t *testing.T) {
	srv := newAuthServer(t, "")
	token, _ := srv.sessions.CreateSession("luis", false)
	cookie := &http.Cookie{Name: "ccsm_session", Value: token}

	if w := postJSON(srv, "/api/config/users/ghost/totp", "", []*http.Cookie{cookie}); w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestTOTPConfirmWithoutStart(t *testing.T) {
	srv := newAuthServer(t, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	req := httptest.NewRequest("PUT", "/api/config/users/luis/totp", strings.NewReader(`{"code":"000000"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
	if w := serve(srv, req); w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
