package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListProjectsAgentError(t *testing.T) {
	h := &SessionHandler{Agent: newMockAgent()}
	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	h.ListProjects(w, req)

	if w.Code != 502 {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListProjectsOK(t *testing.T) {
	sock, cleanup := mockAgentServer(t, mockAgentOK(`[{"name":"principal","path":"/home/admin"},{"name":"projects/ccsm","path":"/home/admin/projects/ccsm"}]`))
	defer cleanup()

	h := &SessionHandler{Agent: requireAgent(t, sock)}
	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()
	h.ListProjects(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var projects []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 || projects[0]["name"] != "principal" || projects[1]["name"] != "projects/ccsm" {
		t.Errorf("unexpected projects: %v", projects)
	}
}
