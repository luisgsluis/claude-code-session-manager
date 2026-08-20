package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/luisgsluis/claude-code-session-manager/internal/voice"
)

// maxAudioBody caps an uploaded recording. Opus at dictation bitrates runs
// around 3 KB/s, so 8 MiB is roughly 45 minutes of speech — far past any
// sensible dictation and still small enough that a malicious upload cannot
// exhaust memory on a Raspberry Pi.
const maxAudioBody = 8 << 20

// audioTypes is the closed whitelist of recording formats accepted from the
// browser. Chrome and Firefox produce webm/opus; Safari on iOS and macOS
// produces mp4/aac. The rest are here for hand-rolled clients and curl.
var audioTypes = map[string]string{
	"audio/webm":  "webm",
	"audio/ogg":   "ogg",
	"audio/mp4":   "m4a",
	"audio/aac":   "aac",
	"audio/mpeg":  "mp3",
	"audio/wav":   "wav",
	"audio/x-wav": "wav",
}

// VoiceHandler serves dictation, prompt rewriting and meta-prompt editing.
type VoiceHandler struct {
	Service *voice.Service
	Store   voice.PromptStore
	Audit   auditFunc
}

// writeVoiceError maps a voice.Error to its HTTP status, mirroring how
// writeAgentError preserves the executor's status instead of flattening
// everything to 500.
func writeVoiceError(w http.ResponseWriter, err error) {
	if ve, ok := err.(*voice.Error); ok {
		writeError(w, ve.Status, ve.Msg)
		return
	}
	writeError(w, http.StatusInternalServerError, "voice error")
}

// Transcribe accepts a raw audio body and returns its transcription.
//
// The audio is NOT wrapped in JSON: base64 in a JSON body would inflate it by
// a third and collide with the 1 MiB decodeJSON cap, and accepting multipart
// would mean introducing inbound multipart parsing to a project that has none.
// A raw body with the MIME type in Content-Type is the smallest thing that
// works.
func (h *VoiceHandler) Transcribe(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	// Strip parameters: browsers send "audio/webm;codecs=opus".
	base := strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
	ext, ok := audioTypes[strings.ToLower(base)]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported audio content type: "+base)
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxAudioBody)
	data, err := io.ReadAll(body)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "recording too large")
		return
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, "empty recording")
		return
	}

	text, err := h.Service.Transcribe(data, base, "audio."+ext)
	if err != nil {
		writeVoiceError(w, err)
		return
	}
	audit(h.Audit, "voice_transcribe", UserFrom(r), "bytes="+itoa(len(data)))
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}

type rewriteRequest struct {
	Text    string         `json:"text"`
	Role    string         `json:"role"`
	Answers []voice.Answer `json:"answers"`
}

// Rewrite turns dictated or typed text into a structured prompt.
func (h *VoiceHandler) Rewrite(w http.ResponseWriter, r *http.Request) {
	var req rewriteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := h.Service.Rewrite(req.Text, req.Role, req.Answers)
	if err != nil {
		writeVoiceError(w, err)
		return
	}
	audit(h.Audit, "voice_rewrite", UserFrom(r), "role="+res.Role)
	writeJSON(w, http.StatusOK, res)
}

// promptResponse is what the meta-prompt editor needs to render: the active
// version's text, the full version list (each flagged original/active for the
// dropdown's check mark), and the role list the dropdown is built from.
type promptResponse struct {
	Content  string              `json:"content"`
	Roles    []voice.Role        `json:"roles"`
	Versions []voice.VersionInfo `json:"versions"`
}

// GetPrompt returns the active meta-prompt, or one specific version's content
// when asked for by id (0 = the embedded original) — used to populate the
// editor when the dropdown selects a version to view.
func (h *VoiceHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	if v := r.URL.Query().Get("version"); v != "" {
		id, err := atoiNonNegative(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version")
			return
		}
		content, err := h.Store.VersionContent(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "version not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"content": content, "version": id})
		return
	}

	content, _ := h.Store.Load()
	resp := promptResponse{
		Content:  content,
		Versions: h.Store.List(),
	}
	// Roles come from whatever is actually active, so the dropdown can never
	// offer a role the model was not given instructions for.
	if p, err := h.Store.Active(); err == nil {
		resp.Roles = p.Roles
	}
	writeJSON(w, http.StatusOK, resp)
}

type putPromptRequest struct {
	Content string `json:"content"`
	Version int    `json:"version"` // which version to overwrite; ignored when New is true
	Name    string `json:"name"`    // name for the new version; only used when New is true
	New     bool   `json:"new"`
}

// PutPrompt validates and saves a meta-prompt, either over an existing
// version (New false, Version > 0) or as a brand-new named one (New true).
// Saving never changes which version is active — see PromptStore.SaveNew.
func (h *VoiceHandler) PutPrompt(w http.ResponseWriter, r *http.Request) {
	var req putPromptRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "empty prompt")
		return
	}

	var id int
	var err error
	if req.New {
		id, err = h.Store.SaveNew(req.Content, req.Name)
	} else if req.Version > 0 {
		id, err = req.Version, h.Store.SaveOver(req.Version, req.Content)
	} else {
		err = fmt.Errorf("the original prompt cannot be modified; save it as a new version instead")
	}
	if err != nil {
		// Validation failures are the user's editing mistakes and must be
		// readable — "role X is declared but has no section" is actionable in
		// a way that "bad request" is not.
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	audit(h.Audit, "voice_prompt_save", UserFrom(r), fmt.Sprintf("version=%d new=%v", id, req.New))
	writeJSON(w, http.StatusOK, map[string]any{"content": req.Content, "version": id, "versions": h.Store.List()})
}

type activatePromptRequest struct {
	Version int `json:"version"` // 0 = the embedded original
}

// ActivatePrompt makes one saved version (or the embedded original, id 0) the
// one dictation actually uses. It never touches content, so it is always
// reversible by activating a different version — there is no separate
// "restore" operation because of it.
func (h *VoiceHandler) ActivatePrompt(w http.ResponseWriter, r *http.Request) {
	var req activatePromptRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Version < 0 {
		writeError(w, http.StatusBadRequest, "invalid version")
		return
	}
	if err := h.Store.SetActive(req.Version); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit(h.Audit, "voice_prompt_activate", UserFrom(r), fmt.Sprintf("version=%d", req.Version))
	h.GetPrompt(w, r)
}

// AgentProfileReader adapts the executor to voice.ProfileReader, so a provider
// can borrow credentials from a Claude Code profile.
//
// This is why "from_profile" needs no new agent command: profile-content
// already exists and already runs on the host, where the profiles and the key
// helper live.
type AgentProfileReader struct{ Agent Agent }

func (a AgentProfileReader) ProfileContent(name string) (string, error) {
	resp, err := a.Agent.Exec("profile-content", map[string]string{"name": name})
	if err != nil {
		return "", err
	}
	var out map[string]string
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return "", err
	}
	return out["content"], nil
}

// itoa avoids pulling strconv in for one call site in an audit string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// atoiNonNegative parses a version id: 0 (the embedded original) is valid,
// unlike the ids elsewhere in this file that must be positive.
func atoiNonNegative(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, errInvalid
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalid
		}
		n = n*10 + int(c-'0')
		if n > 1_000_000 {
			return 0, errInvalid
		}
	}
	return n, nil
}

var errInvalid = &voice.Error{Status: http.StatusBadRequest, Msg: "invalid number"}
