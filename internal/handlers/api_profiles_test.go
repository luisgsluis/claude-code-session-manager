package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListProfilesAgentError(t *testing.T) {
	h := &ProfileHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/profiles", nil)
	w := httptest.NewRecorder()
	h.ListProfiles(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestApplyProfileNoName(t *testing.T) {
	h := &ProfileHandler{Agent: newMockAgent()}
	body := `{}`
	req := httptest.NewRequest("POST", "/api/profiles/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ApplyProfile(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApplyProfileInvalidJSON(t *testing.T) {
	h := &ProfileHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("POST", "/api/profiles/apply", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ApplyProfile(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApplyProfileInvalidName(t *testing.T) {
	h := &ProfileHandler{Agent: newMockAgent()}
	for _, name := range []string{"bad/name", "../evil", "-leading-dash"} {
		body := `{"profile":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/profiles/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ApplyProfile(w, req)
		if w.Code != 400 {
			t.Errorf("name %q: expected 400, got %d", name, w.Code)
		}
	}
}

func TestListProfilesParseError(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`"not-an-array"`))
	defer cleanup()

	h := &ProfileHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("GET", "/api/profiles", nil)
	w := httptest.NewRecorder()
	h.ListProfiles(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListProfilesSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`[{"name":"estandar","label":"estandar"}]`))
	defer cleanup()

	h := &ProfileHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("GET", "/api/profiles", nil)
	w := httptest.NewRecorder()
	h.ListProfiles(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestListProfilesNullData(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`null`))
	defer cleanup()

	h := &ProfileHandler{Agent: requireAgent(t, sockPath)}
	req := httptest.NewRequest("GET", "/api/profiles", nil)
	w := httptest.NewRecorder()
	h.ListProfiles(w, req)

	// null data → should be handled (nil check)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestApplyProfileSuccess(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`"profile deepseek applied"`))
	defer cleanup()

	h := &ProfileHandler{Agent: requireAgent(t, sockPath)}
	body := `{"profile":"deepseek"}`
	req := httptest.NewRequest("POST", "/api/profiles/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ApplyProfile(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApplyProfileRelaunchedSessions(t *testing.T) {
	sockPath, cleanup := mockAgentServer(t, mockAgentOK(`{"relaunched":["5","7"]}`))
	defer cleanup()

	h := &ProfileHandler{Agent: requireAgent(t, sockPath)}
	body := `{"profile":"estandar"}`
	req := httptest.NewRequest("POST", "/api/profiles/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ApplyProfile(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var data struct {
		Relaunched []string `json:"relaunched"`
		OK         string   `json:"ok"`
	}
	json.Unmarshal(w.Body.Bytes(), &data)
	if len(data.Relaunched) != 2 || data.Relaunched[0] != "5" || data.Relaunched[1] != "7" {
		t.Errorf("relaunched: %v", data.Relaunched)
	}
	if !strings.Contains(data.OK, "5") || !strings.Contains(data.OK, "7") {
		t.Errorf("ok message should mention relaunched sessions: %q", data.OK)
	}
}

func TestApplyProfileAgentError(t *testing.T) {
	h := &ProfileHandler{Agent: newMockAgent()}
	body := `{"profile":"deepseek"}`
	req := httptest.NewRequest("POST", "/api/profiles/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ApplyProfile(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}
