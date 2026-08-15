package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/agent"
)

func newMockAgent() *agent.Client {
	return agent.NewClient("/dev/null", "")
}

func TestListSessionsAgentError(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	h.ListSessions(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestKillSessionNoName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("DELETE", "/api/sessions/", nil)
	// Path value "" because no {name} in pattern when called directly
	w := httptest.NewRecorder()
	h.KillSession(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestKillSessionInvalidName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	for _, name := range []string{"a!b", "..", "sess;rm"} {
		req := httptest.NewRequest("DELETE", "/api/sessions/"+name, nil)
		req.SetPathValue("name", name)
		w := httptest.NewRecorder()
		h.KillSession(w, req)
		if w.Code != 400 {
			t.Errorf("name %q: expected 400, got %d", name, w.Code)
		}
	}
}

func TestKillSessionAgentError(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("DELETE", "/api/sessions/test", nil)
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()
	h.KillSession(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestNewSessionAgentError(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NewSession(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestNewSessionWithProfile(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	body := `{"profile":"deepseek"}`
	req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NewSession(w, req)

	// Agent unavailable → 502, but profile should be parsed correctly
	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestNewSessionWithProject(t *testing.T) {
	// Valid project names pass the handler; the agent is unavailable here so
	// 502 means the request was forwarded.
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	for _, body := range []string{
		`{"project":"claude-code-session-manager"}`,
		`{"project":"principal"}`,
		`{"project":"homelab/services/ccsm"}`,
	} {
		req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.NewSession(w, req)
		if w.Code != 502 {
			t.Errorf("body %s: expected 502, got %d: %s", body, w.Code, w.Body.String())
		}
	}
}

func TestNewSessionInvalidProject(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	for _, body := range []string{
		`{"project":"../etc"}`,
		`{"project":"a b"}`,
		`{"project":"a\\b"}`,
		`{"project":"/abs"}`,
	} {
		req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.NewSession(w, req)
		if w.Code != 400 {
			t.Errorf("body %s: expected 400, got %d: %s", body, w.Code, w.Body.String())
		}
	}
}

func TestNewSessionInvalidProfileName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	body := `{"profile":"bad/name"}`
	req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NewSession(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNewSessionInvalidJSON(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader("notjson"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NewSession(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for malformed JSON, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResumeSessionNoID(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/resume", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResumeSession(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestResumeSessionInvalidID(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	for _, id := range []string{"notauuid", "../../etc/passwd", "00000000-0000-0000-0000-00000000000Z"} {
		body := `{"id":"` + id + `"}`
		req := httptest.NewRequest("POST", "/api/sessions/resume", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ResumeSession(w, req)
		if w.Code != 400 {
			t.Errorf("id %q: expected 400, got %d", id, w.Code)
		}
	}
}

func TestResumeSessionAgentError(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent(), AttachAddr: "admin@host"}
	body := `{"id":"00000000-0000-0000-0000-000000000001"}`
	req := httptest.NewRequest("POST", "/api/sessions/resume", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResumeSession(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, 200, map[string]string{"ok": "yes"})

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("content type: %s", w.Header().Get("Content-Type"))
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["ok"] != "yes" {
		t.Errorf("body: %v", body)
	}
}

func TestListSessionsSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`[{"name":"0","created":"today","task":"hello"}]`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath), AttachAddr: "admin@host"}
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	h.ListSessions(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var sessions []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sessions)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0]["name"] != "0" {
		t.Errorf("name: %v", sessions[0]["name"])
	}
	if sessions[0]["attach_cmd"] != "ssh admin@host -t tmux a -t 0" {
		t.Errorf("attach_cmd: %v", sessions[0]["attach_cmd"])
	}
}

func TestListSessionsEmpty(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`[]`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath), AttachAddr: "admin@host"}
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	h.ListSessions(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var sessions []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sessions)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestNewSessionSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"session":"3","status":"ok"}`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath), AttachAddr: "admin@host"}
	req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NewSession(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["session_name"] != "3" {
		t.Errorf("session: %v", body["session_name"])
	}
}

func TestResumeSessionSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"session":"5","status":"ok"}`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath), AttachAddr: "admin@host"}
	body := `{"id":"00000000-0000-0000-0000-000000000001"}`
	req := httptest.NewRequest("POST", "/api/sessions/resume", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ResumeSession(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["session_name"] != "5" {
		t.Errorf("session: %v", resp["session_name"])
	}
}

func TestListSessionsNoAttachForEmptyName(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`[{"name":"","created":"today"},{"name":"2","created":"today"}]`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath), AttachAddr: "admin@host"}
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	h.ListSessions(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var sessions []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sessions)
	if _, ok := sessions[0]["attach_cmd"]; ok {
		t.Errorf("session with empty name must not get attach_cmd")
	}
	if sessions[1]["attach_cmd"] != "ssh admin@host -t tmux a -t 2" {
		t.Errorf("attach_cmd: %v", sessions[1]["attach_cmd"])
	}
}

func TestListSessionsParseError(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`"not-an-array"`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath), AttachAddr: "admin@host"}
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	h.ListSessions(w, req)

	// Should fail on JSON unmarshal of non-array data
	if w.Code != 500 {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestKillSessionSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`"killed test"`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("DELETE", "/api/sessions/test", nil)
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()
	h.KillSession(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, 500, "something broke")

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "something broke" {
		t.Errorf("error: %s", body["error"])
	}
}

func TestRenameSessionNoName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions//rename", strings.NewReader(`{"new_name":"y"}`))
	w := httptest.NewRecorder()
	h.RenameSession(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRenameSessionInvalidName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/a!b/rename", strings.NewReader(`{"new_name":"y"}`))
	req.SetPathValue("name", "a!b")
	w := httptest.NewRecorder()
	h.RenameSession(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRenameSessionMissingNewName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/x/rename", strings.NewReader(`{}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.RenameSession(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameSessionInvalidNewName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/x/rename", strings.NewReader(`{"new_name":"a!b"}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.RenameSession(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRenameSessionAgentError(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/test/rename", strings.NewReader(`{"new_name":"newname"}`))
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()
	h.RenameSession(w, req)
	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestRenameSessionSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"old_name":"x","new_name":"y"}`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("POST", "/api/sessions/x/rename", strings.NewReader(`{"new_name":"y"}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.RenameSession(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetClaudeNameNoName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions//claude-name", strings.NewReader(`{"title":"hello"}`))
	w := httptest.NewRecorder()
	h.SetClaudeName(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetClaudeNameInvalidName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/a!b/claude-name", strings.NewReader(`{"title":"hello"}`))
	req.SetPathValue("name", "a!b")
	w := httptest.NewRecorder()
	h.SetClaudeName(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetClaudeNameMissingTitle(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/x/claude-name", strings.NewReader(`{}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.SetClaudeName(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetClaudeNameInvalidTitle(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/x/claude-name", strings.NewReader(`{"title":"\n"}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.SetClaudeName(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetClaudeNameAgentError(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/test/claude-name", strings.NewReader(`{"title":"hello"}`))
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()
	h.SetClaudeName(w, req)
	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestSetClaudeNameSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"session":"x","name":"hello"}`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("POST", "/api/sessions/x/claude-name", strings.NewReader(`{"title":"hello"}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.SetClaudeName(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNewSessionWithName(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"session":"mysession","status":"ok"}`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sockPath), AttachAddr: "admin@host"}
	req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader(`{"name":"mysession"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NewSession(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNewSessionInvalidName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/sessions/new", strings.NewReader(`{"name":"bad/name"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.NewSession(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLiveStreamInvalidName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	for _, name := range []string{"", "bad name!", "../x", "a/b"} {
		req := httptest.NewRequest("GET", "/api/sessions/x/stream", nil)
		req.SetPathValue("name", name)
		w := httptest.NewRecorder()
		h.LiveStream(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for name %q, got %d: %s", name, w.Code, w.Body.String())
		}
	}
}

// TestLiveStreamEmitsPaneContent drives a full SSE round: first poll returns
// pane content (emitted, newlines SSE-encoded), second poll errors (session
// died) and the handler returns. The agent mock makes it deterministic.
func TestLiveStreamEmitsPaneContent(t *testing.T) {
	calls := 0
	sock, stop := mockAgentServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true,"data":{"session":"test","content":"hello\nworld"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":false,"error":"session not found"}`))
	})
	defer stop()

	h := &SessionHandler{Agent: requireAgent(t, sock)}
	req := httptest.NewRequest("GET", "/api/sessions/test/stream", nil)
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()
	h.LiveStream(w, req)

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("expected event-stream content type, got %q", ct)
	}
	if x := w.Header().Get("X-Accel-Buffering"); x != "no" {
		t.Errorf("expected X-Accel-Buffering: no, got %q", x)
	}
	body := w.Body.String()
	if !strings.Contains(body, "data: hello\\nworld") {
		t.Errorf("expected newline-encoded pane content, got: %q", body)
	}
	if !strings.Contains(body, "event: error") {
		t.Errorf("expected error event after session dies, got: %q", body)
	}
}

func TestSendValidation(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	for _, name := range []string{"", "bad name!", "../x", "a/b"} {
		req := httptest.NewRequest("POST", "/api/sessions/x/send", nil)
		req.SetPathValue("name", name)
		w := httptest.NewRecorder()
		h.Send(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for name %q, got %d", name, w.Code)
		}
	}
	// Missing text and keys.
	req := httptest.NewRequest("POST", "/api/sessions/x/send", strings.NewReader(`{}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.Send(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for empty body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendForwardsToAgentAndAudits(t *testing.T) {
	sock, stop := mockAgentServer(t, mockAgentOK(`{"session":"x","sent":"text"}`))
	defer stop()

	audited := ""
	h := &SessionHandler{Agent: requireAgent(t, sock), Audit: func(action, user, detail string) {
		audited = action + " " + detail
	}}

	req := httptest.NewRequest("POST", "/api/sessions/x/send", strings.NewReader(`{"text":"hola"}`))
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.Send(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.HasPrefix(audited, "session_send") || !strings.Contains(audited, "session=x") {
		t.Errorf("audit not recorded: %q", audited)
	}
}

func TestChatInvalidName(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	for _, name := range []string{"", "bad name!", "a/b"} {
		req := httptest.NewRequest("GET", "/api/sessions/x/chat", nil)
		req.SetPathValue("name", name)
		w := httptest.NewRecorder()
		h.Chat(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for name %q, got %d", name, w.Code)
		}
	}
}

func TestChatReturnsSessionChat(t *testing.T) {
	sock, stop := mockAgentServer(t, mockAgentOK(`{"session":"x","id":"20000000-0000-0000-0000-0000000000aa","ready":true,"messages":[{"index":0,"role":"user","content":"hola"}]}`))
	defer stop()

	h := &SessionHandler{Agent: requireAgent(t, sock)}
	req := httptest.NewRequest("GET", "/api/sessions/x/chat", nil)
	req.SetPathValue("name", "x")
	w := httptest.NewRecorder()
	h.Chat(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"role":"user"`) {
		t.Errorf("chat payload not passed through: %s", w.Body.String())
	}
}

// TestChatStreamEmitsOnlyOnChange drives one SSE round: first poll emits the
// payload, a second identical poll is suppressed (same fingerprint), then an
// agent error ends the stream.
func TestChatStreamEmitsOnlyOnChange(t *testing.T) {
	payload := `{"session":"test","id":"20000000-0000-0000-0000-0000000000aa","ready":true,"status":"rc_connected","mode":"","updated":"2026-08-11T10:00:00Z","size":123,"messages":[{"index":0,"role":"user","content":"hola"}]}`
	calls := 0
	sock, stop := mockAgentServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls <= 2 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true,"data":` + payload + `}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":false,"error":"session not found"}`))
	})
	defer stop()

	h := &SessionHandler{Agent: requireAgent(t, sock)}
	req := httptest.NewRequest("GET", "/api/sessions/test/chat/stream", nil)
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()
	h.ChatStream(w, req)

	body := w.Body.String()
	if n := strings.Count(body, "data: {"); n != 1 {
		t.Errorf("expected exactly 1 data event (deduped), got %d: %q", n, body)
	}
	if !strings.Contains(body, `data: {"session":"test"`) {
		t.Errorf("expected chat payload emitted, got: %q", body)
	}
	if !strings.Contains(body, "event: error") {
		t.Errorf("expected error event, got: %q", body)
	}
}

// TestChatStreamEmitsOnWaitingChange: a dialog opening or resolving doesn't
// necessarily touch ready/status/mode/updated/size — the question text is
// never written to the transcript, only the eventual answer is (and even
// that can land in the same poll window as the pane already clearing). If
// waiting/choice aren't in the fingerprint, a client can be left showing a
// choice/approval panel the pane already resolved. Two payloads differing
// only in "waiting" must both be emitted, not deduped as identical.
func TestChatStreamEmitsOnWaitingChange(t *testing.T) {
	base := `"session":"test","id":"20000000-0000-0000-0000-0000000000aa","ready":true,"status":"rc_connected","mode":"","updated":"2026-08-11T10:00:00Z","size":123,"messages":[]`
	waiting := `{` + base + `,"waiting":"choice","choice":{"question":"q","options":["A","B"],"selected":0}}`
	resolved := `{` + base + `,"waiting":""}`
	calls := 0
	sock, stop := mockAgentServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case calls == 1:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true,"data":` + waiting + `}`))
		case calls == 2:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true,"data":` + resolved + `}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":false,"error":"session not found"}`))
		}
	})
	defer stop()

	h := &SessionHandler{Agent: requireAgent(t, sock)}
	req := httptest.NewRequest("GET", "/api/sessions/test/chat/stream", nil)
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()
	h.ChatStream(w, req)

	body := w.Body.String()
	if n := strings.Count(body, "data: {"); n != 2 {
		t.Errorf("expected 2 data events (waiting change must not be deduped), got %d: %q", n, body)
	}
	if !strings.Contains(body, `"waiting":"choice"`) || !strings.Contains(body, `"waiting":""`) {
		t.Errorf("expected both waiting states in the stream, got: %q", body)
	}
}
