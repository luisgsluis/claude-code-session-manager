package handlers

import (
	"encoding/json"
	"net/http"
)

// ListProjects returns the discoverable launch targets for a new session: the
// "principal" entry (home) plus every directory under home with a CLAUDE.md.
func (h *SessionHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Agent.Exec("projects-ls", nil)
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var projects []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &projects); err != nil {
		writeError(w, http.StatusInternalServerError, "parse error")
		return
	}
	if projects == nil {
		projects = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, projects)
}
