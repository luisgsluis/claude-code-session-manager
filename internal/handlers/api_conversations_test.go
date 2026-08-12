package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListConversationsAgentError(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()
	h.ListConversations(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestListConversationsWithQuery(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations?q=bug&origin=pi&page=1&per_page=10", nil)
	w := httptest.NewRecorder()
	h.ListConversations(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestListConversationsDefaults(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()
	h.ListConversations(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestGetConversationNoID(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations/", nil)
	w := httptest.NewRecorder()
	h.GetConversation(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetConversationInvalidID(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	for _, id := range []string{"notauuid", "../../etc/passwd", "00000000-0000-0000-0000-00000000000Z"} {
		req := httptest.NewRequest("GET", "/api/conversations/"+id, nil)
		req.SetPathValue("id", id)
		w := httptest.NewRecorder()
		h.GetConversation(w, req)
		if w.Code != 400 {
			t.Errorf("id %q: expected 400, got %d", id, w.Code)
		}
	}
}

func TestGetConversationAgentError(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations/00000000-0000-0000-0000-000000000001?lines=10", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.GetConversation(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestListConversationsSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`[{"id":"00000000-0000-0000-0000-000000000001","date":"hoy","origin":"pi","preview":"hola"}]`))
	defer cleanup()

	h := &ConversationHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()
	h.ListConversations(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetConversationSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"id":"uuid","messages":[{"role":"user","content":"hello"}]}`))
	defer cleanup()

	h := &ConversationHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("GET", "/api/conversations/00000000-0000-0000-0000-000000000001", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.GetConversation(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetConversationDefaultLines(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations/00000000-0000-0000-0000-000000000001", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.GetConversation(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestExportConversationSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"filename":"abc.txt","content":"line one\nline two"}`))
	defer cleanup()

	h := &ConversationHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("GET", "/api/conversations/00000000-0000-0000-0000-000000000001/export?format=txt", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.ExportConversation(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment; filename=") {
		t.Errorf("expected attachment disposition, got %q", got)
	}
	if !strings.Contains(w.Body.String(), "line one") {
		t.Errorf("expected exported content, got %q", w.Body.String())
	}
}

func TestExportConversationInvalidID(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	for _, id := range []string{"", "nope", "../x"} {
		req := httptest.NewRequest("GET", "/api/conversations/x/export", nil)
		req.SetPathValue("id", id)
		w := httptest.NewRecorder()
		h.ExportConversation(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for id %q, got %d", id, w.Code)
		}
	}
}

func TestExportConversationAgentError(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations/00000000-0000-0000-0000-000000000001/export", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.ExportConversation(w, req)
	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestGetConversationMetaSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"tags":["infra"],"notes":"","pinned":true,"archived":false}`))
	defer cleanup()

	h := &ConversationHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("GET", "/api/conversations/00000000-0000-0000-0000-000000000001/meta", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.GetConversationMeta(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"pinned":true`) {
		t.Errorf("expected meta in body, got %q", w.Body.String())
	}
}

func TestGetConversationMetaInvalidID(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	for _, id := range []string{"", "nope", "../x"} {
		req := httptest.NewRequest("GET", "/api/conversations/x/meta", nil)
		req.SetPathValue("id", id)
		w := httptest.NewRecorder()
		h.GetConversationMeta(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for id %q, got %d", id, w.Code)
		}
	}
}

func TestGetConversationMetaAgentError(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/conversations/00000000-0000-0000-0000-000000000001/meta", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.GetConversationMeta(w, req)
	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestSetConversationMetaSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"tags":["a","b"],"notes":"n","pinned":false,"archived":true}`))
	defer cleanup()

	h := &ConversationHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("PUT", "/api/conversations/00000000-0000-0000-0000-000000000001/meta",
		strings.NewReader(`{"tags":["a","b"],"notes":"n","archived":true}`))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.SetConversationMeta(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"archived":true`) {
		t.Errorf("expected meta in body, got %q", w.Body.String())
	}
}

func TestSetConversationMetaInvalidID(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	for _, id := range []string{"", "nope", "../x"} {
		req := httptest.NewRequest("PUT", "/api/conversations/x/meta", strings.NewReader(`{"pinned":true}`))
		req.SetPathValue("id", id)
		w := httptest.NewRecorder()
		h.SetConversationMeta(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for id %q, got %d", id, w.Code)
		}
	}
}

func TestSetConversationMetaBadBody(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("PUT", "/api/conversations/00000000-0000-0000-0000-000000000001/meta", strings.NewReader("{not json"))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.SetConversationMeta(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSetConversationMetaAgentError(t *testing.T) {
	h := &ConversationHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("PUT", "/api/conversations/00000000-0000-0000-0000-000000000001/meta",
		strings.NewReader(`{"pinned":true}`))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	h.SetConversationMeta(w, req)
	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}
