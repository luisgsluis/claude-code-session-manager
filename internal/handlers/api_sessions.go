package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/sessionname"
)

// profileNamePattern mirrors the agent's whitelist. A second check here keeps
// the server honest (clean 400) even if the agent were to be bypassed.
var profileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// sessionNamePattern mirrors the agent's whitelist for tmux session names.
var sessionNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// claudeTitlePattern mirrors the agent's whitelist for Claude session titles.
var claudeTitlePattern = regexp.MustCompile(`^[\p{L}\p{N}\p{P} ]{1,80}$`)

// projectNamePattern mirrors the agent's whitelist for project names (relative
// paths under home as returned by /api/projects).
var projectNamePattern = regexp.MustCompile(`^[\p{L}\p{N}._-]+(/[\p{L}\p{N}._-]+)*$`)

// SessionHandler handles /api/sessions endpoints.
type SessionHandler struct {
	Agent      Agent
	AttachAddr string // "admin@host" for display
	Audit      auditFunc
}

// ListSessions returns active tmux sessions.
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Agent.Exec("tmux-ls", nil)
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var sessions []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &sessions); err != nil {
		writeError(w, http.StatusInternalServerError, "parse error")
		return
	}

	if sessions == nil {
		sessions = []map[string]interface{}{}
	}

	// Enrich with attach command
	for i := range sessions {
		name, _ := sessions[i]["name"].(string)
		if name != "" && h.AttachAddr != "" {
			sessions[i]["attach_cmd"] = "ssh " + h.AttachAddr + " -t tmux a -t " + name
		}
	}

	writeJSON(w, http.StatusOK, sessions)
}

// LiveStream streams the live pane content of a session via Server-Sent
// Events. It polls session-pane once a second and emits a data event whenever
// the content changes. The per-iteration write deadline overrides the server's
// 35s WriteTimeout so a long-lived stream survives.
func (h *SessionHandler) LiveStream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx/olivetin)
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// ?color=1 keeps the pane's ANSI colour codes instead of stripping them,
	// for clients that render them (the multi-session terminal grid).
	paneArgs := map[string]string{"name": name}
	if r.URL.Query().Get("color") == "1" {
		paneArgs["color"] = "1"
	}

	last := ""
	heartbeats := 0
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}

		resp, err := h.Agent.Exec("session-pane", paneArgs)
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err)
			flusher.Flush()
			return
		}
		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		content, _ := data["content"].(string)

		// Emit only on change; heartbeat every 30s so proxies keep the stream open.
		if content == last {
			heartbeats++
			if heartbeats%30 == 0 && setWriteDeadline(rc) {
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			}
			continue
		}
		last = content
		heartbeats = 0
		if !setWriteDeadline(rc) {
			return
		}
		// SSE data lines cannot contain raw newlines; encode as \\n.
		fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(content, "\n", "\\n"))
		flusher.Flush()
	}
}

// Chat returns the full conversation of a live session as JSON (one-shot,
// used to load the clean chat view).
func (h *SessionHandler) Chat(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}
	resp, err := h.Agent.Exec("session-chat", map[string]string{"name": name})
	if err != nil {
		writeAgentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp.Data)
}

// ChatStream streams the full conversation of a live session via Server-Sent
// Events, re-emitting whenever the transcript or session status changes. Same
// polling and flushing mechanics as LiveStream.
func (h *SessionHandler) ChatStream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx/olivetin)
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastFP := ""
	heartbeats := 0
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}

		resp, err := h.Agent.Exec("session-chat", map[string]string{"name": name})
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err)
			flusher.Flush()
			return
		}
		// Only re-emit when the transcript or status actually changed.
		var payload map[string]any
		if err := json.Unmarshal(resp.Data, &payload); err != nil {
			continue
		}
		// waiting/choice must be in the fingerprint: a dialog opening or
		// resolving doesn't always touch the transcript (the question text
		// itself is never written to it — only the eventual answer is), so
		// ready/status/mode/updated/size can all stay identical across a
		// dialog's whole lifecycle. Without this the client can be left
		// showing a choice/approval panel the pane already resolved.
		fp := fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v",
			payload["ready"], payload["status"], payload["mode"],
			payload["updated"], payload["size"],
			payload["waiting"], payload["choice"],
			payload["working"], payload["status_text"])
		if fp == lastFP {
			heartbeats++
			if heartbeats%30 == 0 && setWriteDeadline(rc) {
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			}
			continue
		}
		lastFP = fp
		heartbeats = 0
		if !setWriteDeadline(rc) {
			return
		}
		// SSE data lines cannot contain raw newlines; encode as \\n.
		fmt.Fprintf(w, "data: %s\n\n", bytes.ReplaceAll(resp.Data, []byte("\n"), []byte("\\n")))
		flusher.Flush()
	}
}

// Send types a message (or one whitelisted special key) into a live session.
func (h *SessionHandler) Send(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}
	var req struct {
		Text   string `json:"text"`
		Keys   string `json:"keys"`
		Mode   string `json:"mode"`
		Choice *int   `json:"choice"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Text == "" && req.Keys == "" && req.Mode == "" && req.Choice == nil {
		writeError(w, http.StatusBadRequest, "missing text, keys, mode or choice")
		return
	}

	// mode goes through session-mode (Shift+Tab wheel / /plan in the host), not
	// as text: /mode does not exist in Claude Code and the message would be lost.
	// choice goes through session-choice, which navigates the picker to that
	// option itself instead of trusting the client's last-known cursor position.
	op, args := "session-send", map[string]string{"name": name, "text": req.Text, "keys": req.Keys}
	switch {
	case req.Mode != "":
		op, args = "session-mode", map[string]string{"name": name, "mode": req.Mode}
	case req.Choice != nil:
		op, args = "session-choice", map[string]string{"name": name, "index": strconv.Itoa(*req.Choice)}
	}
	_, err := h.Agent.Exec(op, args)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	detail := "session=" + name
	if req.Text != "" {
		detail += ", len=" + strconv.Itoa(len(req.Text))
	}
	if req.Keys != "" {
		detail += ", keys=" + req.Keys
	}
	if req.Mode != "" {
		detail += ", mode=" + req.Mode
	}
	if req.Choice != nil {
		detail += ", choice=" + strconv.Itoa(*req.Choice)
	}
	audit(h.Audit, "session_send", UserFrom(r), detail)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "sent to " + name})
}

// ReconnectRC forces a fresh Remote Control re-registration in a live session.
// /remote-control toggles, so the host decides how many presses: one to enable
// when RC is down, two (off+on) when it's up — the off+on pair makes claude.ai
// register a fresh bridge id, which is what a resumed session that never
// appeared in the mobile app needs.
func (h *SessionHandler) ReconnectRC(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}

	resp, err := h.Agent.Exec("session-rc", map[string]string{"name": name})
	if err != nil {
		writeAgentError(w, err)
		return
	}
	var data map[string]interface{}
	json.Unmarshal(resp.Data, &data)
	presses, _ := data["presses"].(float64)
	status, _ := data["status"].(string)
	recovered, _ := data["recovered"].(bool)
	newsession, _ := data["session"].(string)
	if newsession == "" {
		newsession = name
	}

	audit(h.Audit, "session_rc", UserFrom(r), "session="+name+", presses="+strconv.FormatFloat(presses, 'f', -1, 64)+", status="+status)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"session":   newsession,
		"presses":   presses,
		"status":    status,
		"recovered": recovered,
	})
}

// setWriteDeadline arms a short per-write deadline so a dead client unblocks
// the stream. Some writers (test recorders) don't support deadlines; that is
// not fatal, so ErrNotSupported is ignored.
func setWriteDeadline(rc *http.ResponseController) bool {
	if err := rc.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return false
	}
	return true
}

// KillSession kills a tmux session by name.
func (h *SessionHandler) KillSession(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}

	_, err := h.Agent.Exec("tmux-kill", map[string]string{"name": name})
	if err != nil {
		writeAgentError(w, err)
		return
	}

	audit(h.Audit, "session_kill", UserFrom(r), "session="+name)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "killed " + name})
}

// NewSession creates a new Claude session.
func (h *SessionHandler) NewSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Profile    string `json:"profile"`
		Name       string `json:"name"`
		ClaudeName string `json:"claude_name"`
		Project    string `json:"project"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	args := map[string]string{}
	if req.Profile != "" {
		if !profileNamePattern.MatchString(req.Profile) {
			writeError(w, http.StatusBadRequest, "invalid profile name")
			return
		}
		args["profile"] = req.Profile
	}
	if req.Name != "" {
		name, ok := sessionname.Normalize(req.Name)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid session name")
			return
		}
		args["name"] = name
	}
	if req.ClaudeName != "" {
		if !claudeTitlePattern.MatchString(req.ClaudeName) {
			writeError(w, http.StatusBadRequest, "invalid claude name")
			return
		}
		args["claude_name"] = req.ClaudeName
	}
	if req.Project != "" {
		if !projectNamePattern.MatchString(req.Project) || strings.Contains(req.Project, "..") {
			writeError(w, http.StatusBadRequest, "invalid project")
			return
		}
		args["project"] = req.Project
	}

	resp, err := h.Agent.Exec("claude-nueva", args)
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var data map[string]interface{}
	json.Unmarshal(resp.Data, &data)

	sessionName := ""
	if name, ok := data["session"].(string); ok {
		sessionName = name
		data["attach_cmd"] = "ssh " + h.AttachAddr + " -t tmux a -t " + name
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":           true,
		"session_name": sessionName,
		"status":       data["status"],
		"attach_cmd":   data["attach_cmd"],
	})
	audit(h.Audit, "session_new", UserFrom(r), "session="+sessionName+", profile="+req.Profile+", project="+req.Project)
}

// ResumeSession resumes a conversation.
func (h *SessionHandler) ResumeSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "missing conversation id")
		return
	}
	if !uuidPattern.MatchString(req.ID) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}

	resp, err := h.Agent.Exec("claude-resume", map[string]string{"id": req.ID})
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var data map[string]interface{}
	json.Unmarshal(resp.Data, &data)

	sessionName := ""
	if name, ok := data["session"].(string); ok {
		sessionName = name
		data["attach_cmd"] = "ssh " + h.AttachAddr + " -t tmux a -t " + name
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":           true,
		"session_name": sessionName,
		"status":       data["status"],
		"attach_cmd":   data["attach_cmd"],
	})
	audit(h.Audit, "session_resume", UserFrom(r), "conversation="+req.ID+", session="+sessionName)
}

// RenameSession renames a tmux session.
func (h *SessionHandler) RenameSession(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}
	var req struct {
		NewName string `json:"new_name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.NewName == "" {
		writeError(w, http.StatusBadRequest, "missing new_name")
		return
	}
	newName, ok := sessionname.Normalize(req.NewName)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid new name")
		return
	}

	_, err := h.Agent.Exec("tmux-rename", map[string]string{"name": name, "new_name": newName})
	if err != nil {
		writeAgentError(w, err)
		return
	}
	audit(h.Audit, "session_rename", UserFrom(r), "session="+name+" → "+newName)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "renamed " + name + " to " + newName})
}

// SetClaudeName sets the Claude session name (sends /rename).
func (h *SessionHandler) SetClaudeName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing session name")
		return
	}
	if !sessionNamePattern.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid session name")
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "missing title")
		return
	}
	if !claudeTitlePattern.MatchString(req.Title) {
		writeError(w, http.StatusBadRequest, "invalid title")
		return
	}

	_, err := h.Agent.Exec("claude-rename", map[string]string{"session": name, "title": req.Title})
	if err != nil {
		writeAgentError(w, err)
		return
	}
	audit(h.Audit, "session_claude_name", UserFrom(r), "session="+name+", title="+req.Title)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "renamed claude session " + name})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// maxJSONBody caps request bodies decoded as JSON, so a client can't force
// unbounded memory allocation in json.Decode by sending an oversized body.
const maxJSONBody = 1 << 20 // 1 MiB

// decodeJSON reads and decodes a JSON body under maxJSONBody, writing a 400
// and returning false on any read/size/parse error.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}
