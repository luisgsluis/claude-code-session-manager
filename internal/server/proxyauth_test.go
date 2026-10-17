package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

// newProxyServer is a server in proxy auth mode whose only trusted proxy is
// 172.24.0.1. LANSubnets still lists loopback, to prove the LAN bypass no
// longer opens anything in this mode.
func newProxyServer(t *testing.T) *Server {
	cfg := config.Defaults()
	cfg.Auth = config.AuthProxy
	cfg.TrustedProxies = []string{"172.24.0.1/32"}
	cfg.LANSubnets = []string{"127.0.0.0/8", "192.168.1.0/24"}
	cfg.Users = []config.User{{Username: "luis", PasswordHash: "x"}}
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	cfg.AgentSocket = ""
	return New(cfg, "", "")
}

func jsonBody(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("bad json %q: %v", s, err)
	}
	return m
}

func TestProxyAuthRefusesDirectClients(t *testing.T) {
	srv := newProxyServer(t)
	for _, addr := range []string{"8.8.8.8:1", "127.0.0.1:1", "192.168.1.50:1"} {
		req := httptest.NewRequest("GET", "/api/config", nil)
		req.RemoteAddr = addr
		// A forged identity must not help a client that is not the proxy.
		req.Header.Set("Remote-User", "luis")
		req.Header.Set("X-Forwarded-For", "172.24.0.1")
		if w := serve(srv, req); w.Code != 403 {
			t.Errorf("%s: got %d, want 403", addr, w.Code)
		}
	}
}

func TestProxyAuthTrustsProxyAndTakesUser(t *testing.T) {
	srv := newProxyServer(t)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	req.RemoteAddr = "172.24.0.1:5555"
	req.Header.Set("Remote-User", "luisgsluis")
	w := serve(srv, req)
	b := jsonBody(t, w.Body.String())
	if b["authenticated"] != true || b["username"] != "luisgsluis" || b["proxy_auth"] != true {
		t.Errorf("status = %v", b)
	}

	req = httptest.NewRequest("GET", "/api/config", nil)
	req.RemoteAddr = "172.24.0.1:5555"
	w = serve(srv, req)
	if w.Code != 200 {
		t.Fatalf("config via proxy: %d %s", w.Code, w.Body.String())
	}
	if jsonBody(t, w.Body.String())["auth"] != "proxy" {
		t.Errorf("config should report auth=proxy: %s", w.Body.String())
	}
}

func TestProxyAuthUserFallbackAndSanitize(t *testing.T) {
	srv := newProxyServer(t)
	for hdr, want := range map[string]string{"": "[proxy]", "bad user": "[proxy]", strings.Repeat("a", 65): "[proxy]", "ana": "ana"} {
		req := httptest.NewRequest("GET", "/api/auth/status", nil)
		req.RemoteAddr = "172.24.0.1:1"
		req.Header.Set("Remote-User", hdr)
		if got := jsonBody(t, serve(srv, req).Body.String())["username"]; got != want {
			t.Errorf("header %q: username %v, want %q", hdr, got, want)
		}
	}
}

func TestProxyAuthLoginLogoutAndUsers(t *testing.T) {
	srv := newProxyServer(t)

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
	req.RemoteAddr = "172.24.0.1:1"
	w := serve(srv, req)
	if b := jsonBody(t, w.Body.String()); w.Code != 200 || b["ok"] != true {
		t.Errorf("login via proxy: %d %v", w.Code, b)
	}
	if strings.Contains(w.Header().Get("Set-Cookie"), "ccsm_session") {
		t.Error("proxy mode must not issue a session cookie")
	}

	req = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
	req.RemoteAddr = "192.168.1.50:1"
	if w := serve(srv, req); w.Code != 403 {
		t.Errorf("login direct: got %d, want 403", w.Code)
	}

	req = httptest.NewRequest("POST", "/api/auth/totp", strings.NewReader(`{"code":"123456"}`))
	req.RemoteAddr = "172.24.0.1:1"
	if w := serve(srv, req); w.Code != 404 {
		t.Errorf("totp: got %d, want 404", w.Code)
	}

	for _, rt := range [][2]string{{"GET", "/api/config/users"}, {"POST", "/api/config/users"}, {"POST", "/api/config/users/luis/totp"}} {
		req = httptest.NewRequest(rt[0], rt[1], strings.NewReader(`{}`))
		req.RemoteAddr = "172.24.0.1:1"
		if w := serve(srv, req); w.Code != 404 {
			t.Errorf("%s %s: got %d, want 404", rt[0], rt[1], w.Code)
		}
	}
}

func TestBuiltinModeUnchangedByProxyFields(t *testing.T) {
	// The default mode ignores the proxy header entirely: a header from an
	// outside client is not an identity.
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/config", nil)
	req.RemoteAddr = "8.8.8.8:1"
	req.Header.Set("Remote-User", "luis")
	if w := serve(srv, req); w.Code != 401 {
		t.Errorf("got %d, want 401", w.Code)
	}
}
