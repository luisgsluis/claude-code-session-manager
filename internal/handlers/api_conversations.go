package handlers

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// uuidPattern mirrors the agent's whitelist for conversation ids.
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ConversationHandler handles /api/conversations endpoints.
type ConversationHandler struct {
	Agent Agent
}

// ListConversations returns past Claude conversations from .jsonl files.
func (h *ConversationHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	args := map[string]string{
		"q":        query.Get("q"),
		"origin":   query.Get("origin"),
		"from":     query.Get("from"),
		"to":       query.Get("to"),
		"alive":    query.Get("alive"),
		"archived": query.Get("archived"),
		"page":     query.Get("page"),
		"per_page": query.Get("per_page"),
	}

	// Defaults
	if args["per_page"] == "" {
		args["per_page"] = "20"
	}
	if args["page"] == "" {
		args["page"] = "1"
	}

	resp, err := h.Agent.Exec("conversations-ls", args)
	if err != nil {
		writeAgentError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp.Data)
}

// GetConversation returns a preview of a single conversation.
func (h *ConversationHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing conversation id")
		return
	}
	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}

	linesStr := r.URL.Query().Get("lines")
	lines := 50
	if linesStr != "" {
		n, err := strconv.Atoi(linesStr)
		if err != nil || n < 1 || n > 200 {
			writeError(w, http.StatusBadRequest, "invalid lines (must be 1-200)")
			return
		}
		lines = n
	}

	resp, err := h.Agent.Exec("conversation-get", map[string]string{
		"id":    id,
		"lines": strconv.Itoa(lines),
	})
	if err != nil {
		writeAgentError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp.Data)
}

// ExportConversation downloads a conversation as jsonl or txt.
func (h *ConversationHandler) ExportConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing conversation id")
		return
	}
	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	format := r.URL.Query().Get("format")
	if format != "jsonl" && format != "txt" {
		format = "jsonl"
	}

	resp, err := h.Agent.Exec("conversation-export", map[string]string{
		"id":     id,
		"format": format,
	})
	if err != nil {
		writeAgentError(w, err)
		return
	}

	var data struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		writeError(w, http.StatusInternalServerError, "parse error")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": data.Filename}))
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, data.Content)
}

// GetConversationMeta returns the tags/notes/pin/archive sidecar of a conversation.
func (h *ConversationHandler) GetConversationMeta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing conversation id")
		return
	}
	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	resp, err := h.Agent.Exec("conversation-meta-get", map[string]string{"id": id})
	if err != nil {
		writeAgentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp.Data)
}

// SetConversationMeta saves the tags/notes/pin/archive sidecar of a conversation.
func (h *ConversationHandler) SetConversationMeta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing conversation id")
		return
	}
	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	var req struct {
		Tags     []string `json:"tags"`
		Notes    string   `json:"notes"`
		Pinned   bool     `json:"pinned"`
		Archived bool     `json:"archived"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	resp, err := h.Agent.Exec("conversation-meta-set", map[string]string{
		"id":       id,
		"tags":     strings.Join(req.Tags, ","),
		"notes":    req.Notes,
		"pinned":   strconv.FormatBool(req.Pinned),
		"archived": strconv.FormatBool(req.Archived),
	})
	if err != nil {
		writeAgentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp.Data)
}
