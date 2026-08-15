package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ProfileHandler handles /api/profiles endpoints.
type ProfileHandler struct {
	Agent Agent
	Audit auditFunc
}

// ListProfiles returns available profiles.
func (h *ProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Agent.Exec("profiles-ls", nil)
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var profiles []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &profiles); err != nil {
		writeError(w, http.StatusInternalServerError, "parse error")
		return
	}

	if profiles == nil {
		profiles = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, profiles)
}

// ApplyProfile writes a profile to settings.json on the host.
func (h *ProfileHandler) ApplyProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Profile string `json:"profile"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Profile == "" {
		writeError(w, http.StatusBadRequest, "missing profile name")
		return
	}
	if !profileNamePattern.MatchString(req.Profile) {
		writeError(w, http.StatusBadRequest, "invalid profile name")
		return
	}

	resp, err := h.Agent.Exec("claude-perfil", map[string]string{
		"profile": req.Profile,
	})
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var data struct {
		Relaunched []string `json:"relaunched"`
	}
	json.Unmarshal(resp.Data, &data)

	detail := "profile=" + req.Profile
	msg := "profile " + req.Profile + " applied"
	if len(data.Relaunched) > 0 {
		// The outgoing profile carried alternate credentials the new one
		// doesn't match, so live sessions were killed and resumed rather than
		// left to hot-reload stale auth (see claudeProfile).
		relaunchedList := strings.Join(data.Relaunched, ", ")
		detail += ", relaunched=" + relaunchedList
		msg += "; relaunched sessions to apply new credentials: " + relaunchedList
	}
	audit(h.Audit, "profile_apply", UserFrom(r), detail)
	writeJSON(w, http.StatusOK, map[string]any{"ok": msg, "relaunched": data.Relaunched})
}

// GetProfileContent returns the raw content of a profile file.
func (h *ProfileHandler) GetProfileContent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" || !profileNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid profile name")
		return
	}

	resp, err := h.Agent.Exec("profile-content", map[string]string{"name": name})
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var content map[string]string
	if err := json.Unmarshal(resp.Data, &content); err != nil {
		writeError(w, http.StatusInternalServerError, "parse error")
		return
	}

	writeJSON(w, http.StatusOK, content)
}
