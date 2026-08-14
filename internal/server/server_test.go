package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/agent"
	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

func newTestServer(t *testing.T) *Server {
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{"127.0.0.0/8"}
	// Audit in a temp dir so tests never touch the real $HOME/.ccsm/audit.jsonl.
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	// Hermetic agent: point at a socket that cannot exist so Exec fails with a
	// dial error → 502 (instead of accidentally reaching a real ccsm-agent on
	// this machine, which would answer 403 for the test secret).
	cfg.AgentSocket = "/nonexistent/ccsm-agent-test.sock"
	return New(cfg, "", "")
}

// serve calls the full middleware chain (security headers, logging).
func serve(srv *Server, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(w, req)
	return w
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer(t)
	w := serve(srv, httptest.NewRequest("GET", "/api/health", nil))

	if w.Code != 200 {
		t.Errorf("health: %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("status: %v", body["status"])
	}
	if body["version"] != Version {
		t.Errorf("version: %v", body["version"])
	}
}

// TestStaticNoCache: static assets must be revalidated by the browser (no-cache)
// so a deployed app.js/css never stays stale after a release.
func TestStaticNoCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/app.js", []byte("window.x=1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/index.html", []byte("<html></html>"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	srv := New(cfg, dir, "")
	w := serve(srv, httptest.NewRequest("GET", "/static/app.js", nil))
	if w.Code != 200 {
		t.Fatalf("static: %d", w.Code)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control: %q, want no-cache", cc)
	}

	// The SPA root must revalidate too: a stale index.html is what keeps old
	// chat bubbles and the model dropdown alive in the browser.
	wRoot := serve(srv, httptest.NewRequest("GET", "/", nil))
	if wRoot.Code != 200 {
		t.Fatalf("root: %d", wRoot.Code)
	}
	if cc := wRoot.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("root Cache-Control: %q, want no-cache", cc)
	}
}

func TestAuthStatusUnauthenticated(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("status code: %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["authenticated"] != false {
		t.Error("should not be authenticated")
	}
}

func TestAuthStatusLANBypass(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("status code: %d", w.Code)
	}
}

func TestProtectedEndpointNoAuth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	w := serve(srv, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestProtectedEndpointLANBypass(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	w := serve(srv, req)

	// LAN bypass allows, but agent is not available → 502 Bad Gateway
	if w.Code != 502 {
		t.Errorf("expected 502 (agent unavailable), got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginWithBadCredentials(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	w := serve(srv, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLoginLANBypass(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:45678"
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["ok"] != true {
		t.Error("login should succeed on LAN")
	}
	if body["lan_bypass"] != true {
		t.Error("should indicate LAN bypass")
	}
	if w.Header().Get("Set-Cookie") == "" {
		t.Error("should set session cookie")
	}
}

func TestLogout(t *testing.T) {
	srv := newTestServer(t)
	w := serve(srv, httptest.NewRequest("POST", "/api/auth/logout", nil))

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Set-Cookie") == "" {
		t.Error("should clear cookie")
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv := newTestServer(t)
	w := serve(srv, httptest.NewRequest("GET", "/api/health", nil))

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing CSP")
	}
}

func TestSPARoot(t *testing.T) {
	srv := newTestServer(t)
	w := serve(srv, httptest.NewRequest("GET", "/", nil))

	if w.Code != 200 {
		t.Errorf("GET /: %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("content type: %s", ct)
	}
}

func TestAPI404(t *testing.T) {
	srv := newTestServer(t)
	w := serve(srv, httptest.NewRequest("GET", "/api/nonexistent", nil))

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestLoginInvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	w := serve(srv, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthStatusAuthenticated(t *testing.T) {
	srv := newTestServer(t)
	token, _ := srv.sessions.CreateSession("luis", false)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("status: %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["authenticated"] != true {
		t.Error("should be authenticated with valid cookie")
	}
	if body["username"] != "luis" {
		t.Errorf("username: %v", body["username"])
	}
}

func TestAuthenticatedAccessToProtected(t *testing.T) {
	srv := newTestServer(t)
	token, _ := srv.sessions.CreateSession("luis", false)

	req := httptest.NewRequest("GET", "/api/profiles", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
	w := serve(srv, req)

	// Allowed (authenticated), but agent unavailable → 502
	if w.Code != 502 {
		t.Errorf("expected 502 (agent unavailable), got %d", w.Code)
	}
}

func TestConfigEndpoint(t *testing.T) {
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{"192.168.1.0/24"}
	cfg.AgentSocket = "" // direct mode
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	cfg.Users = []config.User{{Username: "luis", PasswordHash: "x"}}
	srv := New(cfg, "", "")

	t.Run("requires auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/config", nil)
		req.RemoteAddr = "8.8.8.8:12345"
		w := serve(srv, req)
		if w.Code != 401 {
			t.Errorf("got %d, want 401", w.Code)
		}
	})

	t.Run("returns non-secret info", func(t *testing.T) {
		token, _ := srv.sessions.CreateSession("luis", false)
		req := httptest.NewRequest("GET", "/api/config", nil)
		req.RemoteAddr = "8.8.8.8:12345"
		req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
		w := serve(srv, req)
		if w.Code != 200 {
			t.Fatalf("got %d: %s", w.Code, w.Body.String())
		}
		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)
		if body["mode"] != "direct" {
			t.Errorf("mode: %v", body["mode"])
		}
		users := body["users"].([]interface{})
		if len(users) != 1 {
			t.Fatalf("users: %v", users)
		}
		u := users[0].(map[string]interface{})
		if u["username"] != "luis" || u["totp"] != false {
			t.Errorf("users: %v", users)
		}
		if _, hasSecret := body["session_secret"]; hasSecret {
			t.Error("session_secret must never be exposed")
		}
		if _, hasHash := body["password_hash"]; hasHash {
			t.Error("password_hash must never be exposed")
		}
		paths := body["paths"].(map[string]interface{})
		if _, ok := paths["settings"]; !ok {
			t.Error("paths missing settings")
		}
	})
}

func TestLoginSuccessWithValidUser(t *testing.T) {
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{}
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	// Use a real bcrypt hash for "test123"
	hash := "$2a$10$H/SC9MUlyPBtcbU1Y8/EMu1vClnnTOUf8gK3jQ7WDv.8.5pwwTQ4W"
	cfg.Users = []config.User{{Username: "luis", PasswordHash: hash}}
	srv := New(cfg, "", "")

	body := `{"username":"luis","password":"test123"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Set-Cookie") == "" {
		t.Error("should set cookie on success")
	}
}

func TestSPARootWithStaticPath(t *testing.T) {
	cfg := config.Defaults()
	cfg.SessionSecret = "test"
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/index.html", []byte("<html>test</html>"), 0644)
	srv := New(cfg, tmpDir, "")

	req := httptest.NewRequest("GET", "/", nil)
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLogoutNoToken(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLogoutWithToken(t *testing.T) {
	srv := newTestServer(t)
	token, _ := srv.sessions.CreateSession("luis", false)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
	w := serve(srv, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// After logout, token should be invalid
	if srv.sessions.GetSession(token) != nil {
		t.Error("session should be deleted after logout")
	}
}

func TestAuthStatusExpiredSession(t *testing.T) {
	srv := newTestServer(t)
	srv.sessions.CreateSession("test", false)
	// Find a valid token by creating one
	token, _ := srv.sessions.CreateSession("user", false)
	// Delete it to simulate expiration
	srv.sessions.DeleteSession(token)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
	w := serve(srv, req)

	// Expired/deleted token should show not authenticated
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["authenticated"] != false {
		t.Error("deleted token should not authenticate")
	}
}

func TestAuthMiddlewareExpiredSession(t *testing.T) {
	srv := newTestServer(t)
	token, _ := srv.sessions.CreateSession("user", false)
	srv.sessions.DeleteSession(token)

	req := httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
	w := serve(srv, req)

	// Expired cookie → 401
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
	// Should clear cookie
	if w.Header().Get("Set-Cookie") == "" {
		t.Error("should clear expired cookie")
	}
}

func TestAuthMiddlewareEmptyToken(t *testing.T) {
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{} // no LAN bypass
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	srv := New(cfg, "", "")

	req := httptest.NewRequest("GET", "/api/sessions", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	// No cookie
	w := serve(srv, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRenameSessionNoAuth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/sessions/x/rename", strings.NewReader(`{"new_name":"y"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	w := serve(srv, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRenameSessionLAN(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/sessions/x/rename", strings.NewReader(`{"new_name":"y"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:45678"
	w := serve(srv, req)
	// LAN bypass allows, agent unavailable → 502
	if w.Code != 502 {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetClaudeNameNoAuth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/sessions/x/claude-name", strings.NewReader(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	w := serve(srv, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSetClaudeNameLAN(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/sessions/x/claude-name", strings.NewReader(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:45678"
	w := serve(srv, req)
	// LAN bypass allows, agent unavailable → 502
	if w.Code != 502 {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExpiredSessionAccess(t *testing.T) {
	srv := newTestServer(t)
	// Create expired session manually
	srv.sessions.CreateSession("old", false)
	// We can't easily expire it without manipulating time, but we can test with invalid token
	req := httptest.NewRequest("GET", "/api/health", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: "invalid-token"})
	w := serve(srv, req)

	// Health is not protected, so 200
	if w.Code != 200 {
		t.Errorf("health should work without auth: %d", w.Code)
	}
}

func TestAuditEndpoint(t *testing.T) {
	srv := newTestServer(t)

	// LAN-bypass login logs an entry for [lan].
	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("{}"))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.RemoteAddr = "127.0.0.1:45678"
	serve(srv, loginReq)

	// Failed login from off-LAN logs login_failed (no users configured → 401).
	failReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"nobody","password":"wrong"}`))
	failReq.Header.Set("Content-Type", "application/json")
	failReq.RemoteAddr = "8.8.8.8:12345"
	serve(srv, failReq)

	// Unauthenticated read is rejected.
	unauth := httptest.NewRequest("GET", "/api/audit", nil)
	unauth.RemoteAddr = "8.8.8.8:12345"
	if w := serve(srv, unauth); w.Code != 401 {
		t.Fatalf("unauthenticated audit: got %d, want 401", w.Code)
	}

	// LAN (authenticated) read returns both entries, most recent first.
	req := httptest.NewRequest("GET", "/api/audit", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	w := serve(srv, req)
	if w.Code != 200 {
		t.Fatalf("audit: got %d, want 200: %s", w.Code, w.Body.String())
	}
	var body struct {
		Entries []struct {
			Action string `json:"action"`
			User   string `json:"user"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(body.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %s", len(body.Entries), w.Body.String())
	}
	if body.Entries[0].Action != "login_failed" {
		t.Errorf("entries[0].action = %q, want login_failed", body.Entries[0].Action)
	}
	if body.Entries[1].Action != "login" || body.Entries[1].User != "[lan]" {
		t.Errorf("entries[1] = %+v, want login/[lan]", body.Entries[1])
	}
}

func TestExportConversationRoute(t *testing.T) {
	srv := newTestServer(t)
	id := "00000000-0000-0000-0000-000000000001"

	// Unauthenticated (WAN) → 401, proving the route exists behind auth.
	wan := httptest.NewRequest("GET", "/api/conversations/"+id+"/export?format=txt", nil)
	wan.RemoteAddr = "8.8.8.8:12345"
	if w := serve(srv, wan); w.Code != 401 {
		t.Fatalf("export from WAN: got %d, want 401", w.Code)
	}

	// LAN bypass → reaches the host (hermetic agent unreachable → 502).
	lan := httptest.NewRequest("GET", "/api/conversations/"+id+"/export?format=txt", nil)
	lan.RemoteAddr = "127.0.0.1:45678"
	if w := serve(srv, lan); w.Code != 502 {
		t.Fatalf("export from LAN: got %d, want 502 (agent unreachable)", w.Code)
	}
}

func TestConversationMetaRoutes(t *testing.T) {
	srv := newTestServer(t)
	id := "00000000-0000-0000-0000-000000000001"

	for _, tc := range []struct {
		method, path string
		body         string
	}{
		{"GET", "/api/conversations/" + id + "/meta", ""},
		{"PUT", "/api/conversations/" + id + "/meta", `{"tags":["a"],"notes":"n","pinned":true}`},
	} {
		// Unauthenticated (WAN) → 401.
		wan := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		wan.RemoteAddr = "8.8.8.8:12345"
		if w := serve(srv, wan); w.Code != 401 {
			t.Errorf("%s %s from WAN: got %d, want 401", tc.method, tc.path, w.Code)
		}

		// LAN bypass → reaches the host (hermetic agent unreachable → 502).
		lan := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		lan.RemoteAddr = "127.0.0.1:45678"
		if w := serve(srv, lan); w.Code != 502 {
			t.Errorf("%s %s from LAN: got %d, want 502 (agent unreachable)", tc.method, tc.path, w.Code)
		}
	}
}

func TestMetricsRoute(t *testing.T) {
	srv := newTestServer(t)

	// Unauthenticated (WAN) → 401.
	wan := httptest.NewRequest("GET", "/api/metrics", nil)
	wan.RemoteAddr = "8.8.8.8:12345"
	if w := serve(srv, wan); w.Code != 401 {
		t.Fatalf("metrics from WAN: got %d, want 401", w.Code)
	}

	// LAN bypass → reaches the host (hermetic agent unreachable → 502).
	lan := httptest.NewRequest("GET", "/api/metrics", nil)
	lan.RemoteAddr = "127.0.0.1:45678"
	if w := serve(srv, lan); w.Code != 502 {
		t.Fatalf("metrics from LAN: got %d, want 502 (agent unreachable)", w.Code)
	}
}

// stubAgent records the args it receives and returns a canned metrics payload.
type stubAgent struct {
	lastArgs map[string]string
}

func (s *stubAgent) Exec(_ string, args map[string]string) (*agent.Response, error) {
	s.lastArgs = args
	return &agent.Response{
		OK:   true,
		Data: json.RawMessage(`{"sessions_per_day":[],"top_profiles":[],"token_usage_per_day":[],"token_usage_by_model":[]}`),
	}, nil
}

func TestMetricsRouteModelFilter(t *testing.T) {
	srv := newTestServer(t)
	stub := &stubAgent{}
	srv.exec = stub

	req := httptest.NewRequest("GET", "/api/metrics?model=claude-sonnet-4-5", nil)
	req.RemoteAddr = "127.0.0.1:45678" // LAN bypass
	if w := serve(srv, req); w.Code != 200 {
		t.Fatalf("metrics filtered: got %d, want 200", w.Code)
	}
	if stub.lastArgs["model"] != "claude-sonnet-4-5" {
		t.Errorf("model arg = %q, want claude-sonnet-4-5", stub.lastArgs["model"])
	}
	if stub.lastArgs["audit_path"] == "" {
		t.Error("audit_path arg not forwarded")
	}
}

func TestEventsRouteUnauth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/events", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	if w := serve(srv, req); w.Code != 401 {
		t.Fatalf("events from WAN: got %d, want 401", w.Code)
	}
}

func TestEventsSSE(t *testing.T) {
	srv := newTestServer(t)
	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	reader := bufio.NewReader(resp.Body)
	// First frame is the "open" event.
	first, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read open frame: %v", err)
	}
	if !strings.Contains(first, "event: open") {
		t.Errorf("first frame = %q, want event: open", first)
	}

	// Broadcast a session event and read lines until the subscriber data lands.
	srv.events.broadcast("session_new", "luis", "session=x, profile=estandar")
	got := ""
	for !strings.Contains(got, `"action":"session_new"`) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read data frame: %v (last=%q)", err, got)
		}
		got = line
	}
	if !strings.HasPrefix(got, "data: ") {
		t.Errorf("data frame = %q, want data: {...}", got)
	}
}

func TestEventHubDropFull(t *testing.T) {
	hub := newEventHub()
	ch := hub.subscribe()
	defer hub.unsubscribe(ch)
	// Fill the buffer; the 17th broadcast must not block.
	for i := 0; i < 20; i++ {
		hub.broadcast("session_new", "luis", "x")
	}
	if len(ch) != cap(ch) {
		t.Errorf("subscriber buffer = %d/%d, want full (dropped)", len(ch), cap(ch))
	}
}

func TestChatRoutesRegistered(t *testing.T) {
	srv := newTestServer(t)
	// One-shot chat and send hit the (unreachable) agent → 502 on LAN.
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/api/sessions/x/chat", ""},
		{"POST", "/api/sessions/x/send", `{"text":"hola"}`},
	} {
		var r *http.Request
		if c.body != "" {
			r = httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		} else {
			r = httptest.NewRequest(c.method, c.path, nil)
		}
		r.RemoteAddr = "127.0.0.1:45678" // LAN bypass
		w := serve(srv, r)
		if w.Code != 502 {
			t.Errorf("%s %s: expected 502 (agent unavailable), got %d: %s",
				c.method, c.path, w.Code, w.Body.String())
		}
	}
	// The SSE stream opens 200 and reports the agent failure as an error event.
	req := httptest.NewRequest("GET", "/api/sessions/x/chat/stream", nil)
	req.RemoteAddr = "127.0.0.1:45678"
	w := serve(srv, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "event: error") {
		t.Errorf("stream: expected 200 + error event, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChatRoutesRequireAuth(t *testing.T) {
	srv := newTestServer(t)
	cases := []struct{ method, path string }{
		{"GET", "/api/sessions/x/chat"},
		{"GET", "/api/sessions/x/chat/stream"},
		{"POST", "/api/sessions/x/send"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		req.RemoteAddr = "8.8.8.8:12345" // not LAN, no cookie
		w := serve(srv, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401 unauthenticated, got %d", c.method, c.path, w.Code)
		}
	}
}
