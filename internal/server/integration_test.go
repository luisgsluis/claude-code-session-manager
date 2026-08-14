package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

// mockAgent is a programmable fake ccsm-agent over a Unix socket. It records
// every /exec request and answers from a per-command data map.
type mockAgent struct {
	mu      sync.Mutex
	reqs    []agentRequest
	data    map[string]json.RawMessage
	err     map[string]string // command -> error message (returned as ok:false)
	sock    string
	status  int // forced HTTP status for /exec (0 = normal)
	cleanup func()
}

type agentRequest struct {
	Cmd  string            `json:"cmd"`
	Args map[string]string `json:"args"`
}

// startMockAgent creates the socket and serves it; the caller must stop it.
func startMockAgent(t *testing.T, data map[string]string, errs map[string]string) *mockAgent {
	t.Helper()
	sock := t.TempDir() + "/agent.sock"
	m := &mockAgent{
		data: map[string]json.RawMessage{},
		err:  map[string]string{},
		sock: sock,
	}
	for k, v := range data {
		m.data[k] = json.RawMessage(v)
	}
	for k, v := range errs {
		m.err[k] = v
	}

	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/exec", m.handle)
	go http.Serve(l, mux)
	m.cleanup = func() { l.Close() }
	t.Cleanup(m.cleanup)
	return m
}

func (m *mockAgent) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Cmd    string            `json:"cmd"`
		Args   map[string]string `json:"args"`
		Secret string            `json:"secret"`
	}
	json.Unmarshal(body, &req)

	m.mu.Lock()
	m.reqs = append(m.reqs, agentRequest{Cmd: req.Cmd, Args: req.Args})
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if m.status != 0 {
		w.WriteHeader(m.status)
		w.Write([]byte(`{"ok":false,"error":"forced status"}`))
		return
	}
	if emsg, ok := m.err[req.Cmd]; ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ok":false,"error":%q}`, emsg)
		return
	}
	if data, ok := m.data[req.Cmd]; ok {
		fmt.Fprintf(w, `{"ok":true,"data":%s}`, data)
		return
	}
	// Unknown command: mimic the real agent.
	fmt.Fprintf(w, `{"ok":false,"error":"unknown command: %s"}`, req.Cmd)
}

func (m *mockAgent) calls() []agentRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]agentRequest(nil), m.reqs...)
}

func (m *mockAgent) lastCmd() (string, map[string]string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.reqs) == 0 {
		return "", nil, false
	}
	c := m.reqs[len(m.reqs)-1]
	return c.Cmd, c.Args, true
}

// newIntegrationServer builds a real Server wired to the mock agent. LAN bypass
// is off so every request must carry a valid cookie.
func newIntegrationServer(t *testing.T, m *mockAgent, staticPath string) *Server {
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{} // auth required for all
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	cfg.AgentSocket = m.sock
	cfg.AgentSecret = "secret"
	cfg.HostAttachAddr = "admin@rb.lan"
	return New(cfg, staticPath, "")
}

// login gets a session cookie via the real login flow.
func login(t *testing.T, srv *Server, username, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	w := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "ccsm_session" {
			return c.Value
		}
	}
	t.Fatal("no session cookie returned")
	return ""
}

func authedRequest(srv *Server, token, method, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:12345"
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "ccsm_session", Value: token})
	}
	w := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(w, req)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode: %v (body: %s)", err, w.Body.String())
	}
	return m
}

const testUUID = "a1b2c3d4-1111-2222-3333-444455556666"

// --- Full lifecycle E2E ---

func TestE2EProtectedEndpoints(t *testing.T) {
	m := startMockAgent(t, map[string]string{"tmux-ls": `[]`}, nil)
	srv := newIntegrationServer(t, m, "")

	// Without a cookie every protected endpoint must 401.
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/sessions"},
		{"DELETE", "/api/sessions/0"},
		{"POST", "/api/sessions/new"},
		{"POST", "/api/sessions/resume"},
		{"GET", "/api/profiles"},
		{"POST", "/api/profiles/apply"},
		{"GET", "/api/projects"},
		{"GET", "/api/conversations"},
		{"GET", "/api/conversations/" + testUUID},
	} {
		w := authedRequest(srv, "", tc.method, tc.path, `{}`)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestE2ESessions(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"tmux-ls": `[{"name":"9","created":"10/08 21:00","task":"old","status":"rc_pending"},{"name":"0","created":"10/08 09:00","task":"Proyecto CCSM","status":"rc_connected"}]`,
	}, nil)
	srv := newIntegrationServer(t, m, "")

	token, _ := srv.sessions.CreateSession("luis", false)

	w := authedRequest(srv, token, "GET", "/api/sessions", "")
	if w.Code != http.StatusOK {
		t.Fatalf("sessions: %d %s", w.Code, w.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
	// attach_cmd enrichment
	if ac, ok := list[0]["attach_cmd"].(string); !ok || !strings.HasPrefix(ac, "ssh admin@rb.lan") {
		t.Errorf("missing attach_cmd: %v", list[0])
	}
}

func TestE2EProjects(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"projects-ls": `[{"name":"principal","path":"/home/admin"},{"name":"projects/ccsm","path":"/home/admin/projects/ccsm"}]`,
	}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	w := authedRequest(srv, token, "GET", "/api/projects", "")
	if w.Code != http.StatusOK {
		t.Fatalf("projects: %d %s", w.Code, w.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 || list[0]["name"] != "principal" {
		t.Errorf("unexpected projects: %v", list)
	}
}

func TestE2ENewSession(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"claude-nueva": `{"session":"5","status":"ok"}`,
	}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	// With a profile.
	w := authedRequest(srv, token, "POST", "/api/sessions/new", `{"profile":"deepseek"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("new: %d %s", w.Code, w.Body.String())
	}
	body := decodeJSON(t, w)
	if body["ok"] != true || body["session_name"] != "5" {
		t.Errorf("unexpected body: %v", body)
	}
	if ac, ok := body["attach_cmd"].(string); !ok || !strings.Contains(ac, "-t 5") {
		t.Errorf("attach_cmd: %v", body["attach_cmd"])
	}
	cmd, args, _ := m.lastCmd()
	if cmd != "claude-nueva" || args["profile"] != "deepseek" {
		t.Errorf("agent got %s %v", cmd, args)
	}

	// Without a profile.
	authedRequest(srv, token, "POST", "/api/sessions/new", `{}`)
	cmd, args, _ = m.lastCmd()
	if cmd != "claude-nueva" {
		t.Errorf("cmd: %s", cmd)
	}
	if _, has := args["profile"]; has {
		t.Errorf("profile should be absent, got %v", args)
	}

	// With a project.
	w = authedRequest(srv, token, "POST", "/api/sessions/new", `{"project":"projects/ccsm"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("new with project: %d %s", w.Code, w.Body.String())
	}
	cmd, args, _ = m.lastCmd()
	if cmd != "claude-nueva" || args["project"] != "projects/ccsm" {
		t.Errorf("agent got %s %v", cmd, args)
	}

	// Invalid profile name must be rejected server-side.
	w = authedRequest(srv, token, "POST", "/api/sessions/new", `{"profile":"deepseek; rm -rf /"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid profile: expected 400, got %d", w.Code)
	}
	// Path-traversal project must be rejected server-side.
	w = authedRequest(srv, token, "POST", "/api/sessions/new", `{"project":"../etc"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid project: expected 400, got %d", w.Code)
	}
	// Malformed JSON body.
	w = authedRequest(srv, token, "POST", "/api/sessions/new", `not-json`)
	if w.Code != http.StatusOK {
		// Handler ignores decode errors; acceptable, but ensure no panic and valid JSON.
		decodeJSON(t, w)
	}
}

func TestE2EResume(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"claude-resume": `{"session":"7","status":"ok"}`,
	}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	w := authedRequest(srv, token, "POST", "/api/sessions/resume", fmt.Sprintf(`{"id":%q}`, testUUID))
	if w.Code != http.StatusOK {
		t.Fatalf("resume: %d %s", w.Code, w.Body.String())
	}
	if body := decodeJSON(t, w); body["session_name"] != "7" {
		t.Errorf("session_name: %v", body)
	}
	cmd, args, _ := m.lastCmd()
	if cmd != "claude-resume" || args["id"] != testUUID {
		t.Errorf("agent got %s %v", cmd, args)
	}

	// Missing id.
	w = authedRequest(srv, token, "POST", "/api/sessions/resume", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing id: expected 400, got %d", w.Code)
	}
	// Invalid id format must not reach the agent.
	w = authedRequest(srv, token, "POST", "/api/sessions/resume", `{"id":"notauuid"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad id: expected 400, got %d", w.Code)
	}
}

func TestE2EKill(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"tmux-kill": `"killed 3"`,
	}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	w := authedRequest(srv, token, "DELETE", "/api/sessions/3", "")
	if w.Code != http.StatusOK {
		t.Fatalf("kill: %d %s", w.Code, w.Body.String())
	}
	cmd, args, _ := m.lastCmd()
	if cmd != "tmux-kill" || args["name"] != "3" {
		t.Errorf("agent got %s %v", cmd, args)
	}

	// Invalid name is rejected before reaching the agent.
	w = authedRequest(srv, token, "DELETE", "/api/sessions/%3B%2Fetc%2Fpasswd", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad name: expected 400, got %d", w.Code)
	}
}

func TestE2EConversations(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"conversations-ls": `[{"id":"a1b2c3d4-1111-2222-3333-444455556666","date":"10/08 21:00","origin":"pi","preview":"hola","is_alive":true}]`,
	}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	// List with pagination + search forwarded to the agent.
	w := authedRequest(srv, token, "GET", "/api/conversations?q=hola&page=2&per_page=10&origin=pi", "")
	if w.Code != http.StatusOK {
		t.Fatalf("conv list: %d %s", w.Code, w.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 || list[0]["preview"] != "hola" {
		t.Errorf("unexpected list: %v", list)
	}
	cmd, args, _ := m.lastCmd()
	if cmd != "conversations-ls" || args["q"] != "hola" || args["page"] != "2" || args["per_page"] != "10" || args["origin"] != "pi" {
		t.Errorf("agent got %s %v", cmd, args)
	}
}

func TestE2EConversationGet(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"conversation-get": fmt.Sprintf(`{"id":%q,"date":"10/08","origin":"pi","messages":[{"index":0,"role":"user","content":"hola"}]}`, testUUID),
	}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	w := authedRequest(srv, token, "GET", "/api/conversations/"+testUUID+"?lines=5", "")
	if w.Code != http.StatusOK {
		t.Fatalf("conv get: %d %s", w.Code, w.Body.String())
	}
	body := decodeJSON(t, w)
	if body["id"] != testUUID {
		t.Errorf("id: %v", body["id"])
	}
	cmd, args, _ := m.lastCmd()
	if cmd != "conversation-get" || args["id"] != testUUID || args["lines"] != "5" {
		t.Errorf("agent got %s %v", cmd, args)
	}

	// Default lines.
	authedRequest(srv, token, "GET", "/api/conversations/"+testUUID, "")
	_, args, _ = m.lastCmd()
	if args["lines"] != "50" {
		t.Errorf("default lines: %v", args)
	}

	// Non-UUID id is rejected before reaching the agent.
	w = authedRequest(srv, token, "GET", "/api/conversations/notauuid", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad id: expected 400, got %d", w.Code)
	}
}

func TestE2EProfiles(t *testing.T) {
	m := startMockAgent(t, map[string]string{
		"profiles-ls":   `[{"name":"estandar","label":"estandar","is_active":true},{"name":"deepseek","label":"deepseek","is_active":false}]`,
		"claude-perfil": `"profile estandar applied"`,
	}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	w := authedRequest(srv, token, "GET", "/api/profiles", "")
	if w.Code != http.StatusOK {
		t.Fatalf("profiles: %d %s", w.Code, w.Body.String())
	}
	var profiles []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &profiles); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(profiles) != 2 || profiles[0]["is_active"] != true {
		t.Errorf("unexpected profiles: %v", profiles)
	}

	w = authedRequest(srv, token, "POST", "/api/profiles/apply", `{"profile":"estandar"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("apply: %d %s", w.Code, w.Body.String())
	}
	cmd, args, _ := m.lastCmd()
	if cmd != "claude-perfil" || args["profile"] != "estandar" {
		t.Errorf("agent got %s %v", cmd, args)
	}

	// Empty / invalid profile.
	if w = authedRequest(srv, token, "POST", "/api/profiles/apply", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("empty profile: expected 400, got %d", w.Code)
	}
	if w = authedRequest(srv, token, "POST", "/api/profiles/apply", `{"profile":"../../etc"}`); w.Code != http.StatusBadRequest {
		t.Errorf("invalid profile: expected 400, got %d", w.Code)
	}
	if w = authedRequest(srv, token, "POST", "/api/profiles/apply", `bad`); w.Code != http.StatusBadRequest {
		t.Errorf("bad json: expected 400, got %d", w.Code)
	}
}

func TestE2EAgentDown(t *testing.T) {
	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{}
	cfg.AuditPath = t.TempDir() + "/audit.jsonl"
	cfg.AgentSocket = "/nonexistent/socket.sock"
	srv := New(cfg, "", "")
	token, _ := srv.sessions.CreateSession("luis", false)

	w := authedRequest(srv, token, "GET", "/api/sessions", "")
	if w.Code != http.StatusBadGateway {
		t.Errorf("agent down: expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2ESecurityHeadersEverywhere(t *testing.T) {
	m := startMockAgent(t, map[string]string{"tmux-ls": `[]`}, nil)
	srv := newIntegrationServer(t, m, "")
	token, _ := srv.sessions.CreateSession("luis", false)

	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/health"},
		{"GET", "/api/sessions"},
		{"POST", "/api/auth/login"},
	} {
		var w *httptest.ResponseRecorder
		if tc.method == "POST" && tc.path == "/api/auth/login" {
			w = authedRequest(srv, "", "POST", "/api/auth/login", `{}`)
		} else {
			w = authedRequest(srv, token, tc.method, tc.path, "")
		}
		h := w.Header()
		if h.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s %s: missing nosniff", tc.method, tc.path)
		}
		if h.Get("X-Frame-Options") != "DENY" {
			t.Errorf("%s %s: missing XFO", tc.method, tc.path)
		}
		if h.Get("Content-Security-Policy") == "" {
			t.Errorf("%s %s: missing CSP", tc.method, tc.path)
		}
		if h.Get("Referrer-Policy") == "" {
			t.Errorf("%s %s: missing Referrer-Policy", tc.method, tc.path)
		}
		if h.Get("Permissions-Policy") == "" {
			t.Errorf("%s %s: missing Permissions-Policy", tc.method, tc.path)
		}
	}
}

// TestE2EStaticAssets checks the real static dir is served with the right types.
func TestE2EStaticAssets(t *testing.T) {
	m := startMockAgent(t, map[string]string{"tmux-ls": `[]`}, nil)
	srv := newIntegrationServer(t, m, "../../static")

	for _, tc := range []struct{ path, ct string }{
		{"/", "text/html"},
		{"/static/css/app.css", "text/css"},
		{"/static/js/app.js", "text/javascript"},
		{"/static/js/alpine.min.js", "text/javascript"},
	} {
		w := authedRequest(srv, "", "GET", tc.path, "")
		if w.Code != http.StatusOK {
			t.Errorf("%s: %d", tc.path, w.Code)
			continue
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, tc.ct) {
			t.Errorf("%s: content-type %q, want %q", tc.path, ct, tc.ct)
		}
	}

	// The SPA HTML must reference the self-contained assets, not a CDN.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	w := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(w, req)
	html := w.Body.String()
	for _, cdn := range []string{"cdn.jsdelivr", "unpkg.com", "cdn.tailwindcss"} {
		if strings.Contains(html, cdn) {
			t.Errorf("HTML still references CDN: %s", cdn)
		}
	}
}
