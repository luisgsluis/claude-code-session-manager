package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
	"github.com/luisgsluis/claude-code-session-manager/internal/host"
	"github.com/luisgsluis/claude-code-session-manager/internal/voice"
)

// theKey is a recognisable secret: any test that finds it in a response or a
// log has found a leak.
const theKey = "sk-THE-SECRET-KEY-DO-NOT-LEAK"

// newVoiceServer builds a server with voice configured and a writable prompts
// directory, and returns the config path so tests can inspect what was
// persisted.
func newVoiceServer(t *testing.T, mutate func(*config.Config)) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := config.Defaults()
	cfg.SessionSecret = "test-secret"
	cfg.LANSubnets = []string{"127.0.0.0/8"}
	cfg.AuditPath = filepath.Join(dir, "audit.jsonl")
	cfg.AgentSocket = "/nonexistent/ccsm-agent-test.sock"
	cfg.Voice = config.VoiceConfig{
		Enabled:     true,
		Language:    "es",
		PromptsPath: filepath.Join(dir, "prompts"),
		STT: config.VoiceSTTConfig{
			Mode: "whisper", Provider: "groq", Vocabulary: "sonarr, tmux",
		},
		Rewrite: config.VoiceRewriteConfig{
			Enabled: true, Provider: "groq", MaxQuestions: 3, DefaultRole: "auto",
		},
		Providers: map[string]config.VoiceProvider{
			"groq": {
				BaseURL: "https://api.groq.example/v1", APIKey: theKey,
				STTModel: "whisper-x", ChatModel: "chat-x",
			},
			"chatonly": {
				BaseURL: "https://other.example/v1", APIKey: theKey,
				ChatModel: "chat-y",
			},
		},
	}
	if mutate != nil {
		mutate(cfg)
	}
	if err := os.WriteFile(cfgPath, []byte("port: 8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return New(cfg, "", cfgPath), cfgPath
}

func doJSON(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1234" // LAN bypass, so no login is needed
	return serve(srv, req)
}

// TestVoiceConfigNeverExposesCredentials is the one that must never regress.
// The settings panel needs to list providers to build a dropdown; it must
// learn nothing beyond their names and capabilities.
func TestVoiceConfigNeverExposesCredentials(t *testing.T) {
	srv, _ := newVoiceServer(t, func(c *config.Config) {
		p := c.Voice.Providers["chatonly"]
		p.APIKey = ""
		p.APIKeyHelper = "/home/admin/.local/bin/claude-apikey"
		c.Voice.Providers["chatonly"] = p
	})

	w := doJSON(t, srv, "GET", "/api/config", nil)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	for _, forbidden := range []string{
		theKey,
		"api_key",
		"claude-apikey",  // the helper path is a filesystem hint, also not exposed
		"api_key_helper", //
		"from_profile",   //
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("GET /api/config leaked %q:\n%s", forbidden, body)
		}
	}

	var cfg struct {
		Voice struct {
			Enabled   bool `json:"enabled"`
			Providers []struct {
				Name string `json:"name"`
				STT  bool   `json:"stt"`
				Chat bool   `json:"chat"`
			} `json:"providers"`
			MaxSendLen int `json:"max_send_len"`
		} `json:"voice"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Voice.Providers) != 2 {
		t.Fatalf("expected 2 providers listed, got %d", len(cfg.Voice.Providers))
	}
	// Sorted, so the UI dropdown has a stable order rather than Go's random
	// map iteration.
	if cfg.Voice.Providers[0].Name != "chatonly" || cfg.Voice.Providers[1].Name != "groq" {
		t.Errorf("providers are not sorted: %+v", cfg.Voice.Providers)
	}
	if cfg.Voice.Providers[1].STT != true || cfg.Voice.Providers[0].STT != false {
		t.Error("capabilities are wrong: only groq has an stt_model")
	}
	// The counter in the review panel is driven by this, and it has to be the
	// real cap or the UI blocks valid prompts (or lets invalid ones through).
	if cfg.Voice.MaxSendLen != host.MaxSendLen {
		t.Errorf("max_send_len = %d, want %d", cfg.Voice.MaxSendLen, host.MaxSendLen)
	}
}

func TestPatchVoiceHotReload(t *testing.T) {
	srv, cfgPath := newVoiceServer(t, nil)

	w := doJSON(t, srv, "PATCH", "/api/config", map[string]any{
		"voice": map[string]any{
			"stt":     map[string]any{"mode": "webspeech", "vocabulary": "macvlan"},
			"rewrite": map[string]any{"provider": "chatonly", "model": "chat-y", "max_questions": 1},
		},
	})
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}

	srv.cfgMu.RLock()
	got := srv.cfg.Voice
	srv.cfgMu.RUnlock()
	if got.STT.Mode != "webspeech" {
		t.Errorf("mode = %q", got.STT.Mode)
	}
	if got.Rewrite.Provider != "chatonly" || got.Rewrite.Model != "chat-y" {
		t.Errorf("rewrite = %+v", got.Rewrite)
	}
	if got.Rewrite.MaxQuestions != 1 {
		t.Errorf("max_questions = %d", got.Rewrite.MaxQuestions)
	}
	// The provider's credentials must survive a patch that never mentioned
	// them: a hot switch must not blank the key it is switching away from.
	if srv.cfg.Voice.Providers["groq"].APIKey != theKey {
		t.Error("patching the voice config damaged a provider's credentials")
	}

	// And the key must not be written into a config file that a patch touches.
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "webspeech") {
		t.Error("the patch was not persisted")
	}
	if !strings.Contains(string(data), theKey) {
		t.Error("write-back dropped the provider credentials from config.yaml")
	}
}

func TestPatchVoiceRejections(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
		want string
	}{
		{
			"unknown stt provider",
			map[string]any{"stt": map[string]any{"provider": "ghost"}},
			"unknown voice provider ghost",
		},
		{
			"provider that cannot transcribe",
			map[string]any{"stt": map[string]any{"provider": "chatonly"}},
			"has no stt_model",
		},
		{
			"invalid mode",
			map[string]any{"stt": map[string]any{"mode": "telepathy"}},
			"voice.stt.mode must be one of",
		},
		{
			"max_questions out of range",
			map[string]any{"rewrite": map[string]any{"max_questions": 99}},
			"max_questions must be 0-10",
		},
		{
			"unknown role",
			map[string]any{"rewrite": map[string]any{"default_role": "wizard"}},
			"unknown role wizard",
		},
		{
			"invalid model name",
			map[string]any{"rewrite": map[string]any{"model": "bad model; rm -rf /"}},
			"invalid voice.rewrite.model",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv, _ := newVoiceServer(t, nil)
			before := srv.cfg.Voice

			w := doJSON(t, srv, "PATCH", "/api/config", map[string]any{"voice": c.body})
			if w.Code != 400 {
				t.Fatalf("status %d, want 400: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), c.want) {
				t.Errorf("error %q does not mention %q", w.Body.String(), c.want)
			}
			// A rejected patch must change nothing at all.
			if srv.cfg.Voice.STT != before.STT || srv.cfg.Voice.Rewrite != before.Rewrite {
				t.Error("a rejected patch mutated the config")
			}
		})
	}
}

// TestPatchVoiceAcceptsARoleFromTheEditedPrompt: roles live in the meta-prompt,
// so adding one there must make it selectable without touching Go.
func TestPatchVoiceAcceptsARoleFromTheEditedPrompt(t *testing.T) {
	srv, _ := newVoiceServer(t, nil)

	custom := `---
roles:
  - {id: auto, es: Automático, en: Auto}
  - {id: seguridad, es: Seguridad, en: Security}
---
# Base

Rules. Ask at most MAX_QUESTIONS questions.

# Role: auto

Classify.

# Role: seguridad

Threat modelling.
`
	if w := doJSON(t, srv, "PUT", "/api/voice/prompt", map[string]string{"content": custom}); w.Code != 200 {
		t.Fatalf("saving the prompt: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, srv, "PATCH", "/api/config", map[string]any{
		"voice": map[string]any{"rewrite": map[string]any{"default_role": "seguridad"}},
	}); w.Code != 200 {
		t.Fatalf("a role added to the meta-prompt should be selectable: %d %s", w.Code, w.Body.String())
	}
	// And one that was removed must stop being accepted.
	if w := doJSON(t, srv, "PATCH", "/api/config", map[string]any{
		"voice": map[string]any{"rewrite": map[string]any{"default_role": "devops"}},
	}); w.Code != 400 {
		t.Errorf("a role no longer in the prompt must be rejected, got %d", w.Code)
	}
}

func TestVoicePromptEndpoints(t *testing.T) {
	srv, _ := newVoiceServer(t, nil)

	// GET serves the embedded original when nothing has been saved.
	w := doJSON(t, srv, "GET", "/api/voice/prompt", nil)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var got struct {
		Content  string       `json:"content"`
		Original string       `json:"original"`
		Custom   bool         `json:"custom"`
		Roles    []voice.Role `json:"roles"`
		Versions []int        `json:"versions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Custom {
		t.Error("a fresh install must not report a custom prompt")
	}
	if got.Content != got.Original {
		t.Error("content should equal the original before any edit")
	}
	if len(got.Roles) < 5 {
		t.Errorf("expected the shipped roles, got %d", len(got.Roles))
	}

	// PUT saves and echoes the new state.
	edited := strings.Replace(got.Original, "# Base", "# Base\n\nEXTRA LINE.", 1)
	w = doJSON(t, srv, "PUT", "/api/voice/prompt", map[string]string{"content": edited})
	if w.Code != 200 {
		t.Fatalf("PUT: %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Custom || !strings.Contains(got.Content, "EXTRA LINE") {
		t.Error("PUT did not activate the edited prompt")
	}
	if got.Original == got.Content {
		t.Error("the original must still be served alongside the edit")
	}

	// Reset goes back to the original.
	w = doJSON(t, srv, "POST", "/api/voice/prompt/reset", nil)
	if w.Code != 200 {
		t.Fatalf("reset: %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Custom || strings.Contains(got.Content, "EXTRA LINE") {
		t.Error("reset did not restore the original")
	}
}

// TestVoicePromptRejectionIsActionable: the editor shows this string to the
// user, so "bad request" would leave them guessing what they broke.
func TestVoicePromptRejectionIsActionable(t *testing.T) {
	srv, _ := newVoiceServer(t, nil)

	broken := "---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\nno block\n"
	w := doJSON(t, srv, "PUT", "/api/voice/prompt", map[string]string{"content": broken})
	if w.Code != 400 {
		t.Fatalf("status %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "# Role: devops") {
		t.Errorf("the error should name the missing section: %s", w.Body.String())
	}

	// And the previous prompt is untouched.
	w = doJSON(t, srv, "GET", "/api/voice/prompt", nil)
	if strings.Contains(w.Body.String(), "no block") {
		t.Error("a rejected save reached disk")
	}
}

func TestVoicePromptVersionEndpoint(t *testing.T) {
	srv, _ := newVoiceServer(t, nil)
	orig := voice.Original()

	for _, marker := range []string{"MARK-ONE", "MARK-TWO"} {
		edited := strings.Replace(orig, "# Base", "# Base\n\n"+marker+".", 1)
		if w := doJSON(t, srv, "PUT", "/api/voice/prompt", map[string]string{"content": edited}); w.Code != 200 {
			t.Fatalf("PUT %s: %d", marker, w.Code)
		}
	}

	w := doJSON(t, srv, "GET", "/api/voice/prompt?version=1", nil)
	if w.Code != 200 {
		t.Fatalf("version 1: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "MARK-ONE") {
		t.Error("version 1 should hold the first edit")
	}
	if w := doJSON(t, srv, "GET", "/api/voice/prompt?version=99", nil); w.Code != 404 {
		t.Errorf("a missing version should 404, got %d", w.Code)
	}
	if w := doJSON(t, srv, "GET", "/api/voice/prompt?version=abc", nil); w.Code != 400 {
		t.Errorf("a non-numeric version should 400, got %d", w.Code)
	}
}

// TestTranscribeRejectsBadContentTypes: the whitelist is the only thing
// standing between the endpoint and arbitrary uploads.
func TestTranscribeRejectsBadContentTypes(t *testing.T) {
	srv, _ := newVoiceServer(t, nil)

	for _, ct := range []string{"application/json", "image/png", "text/plain", ""} {
		req := httptest.NewRequest("POST", "/api/voice/transcribe", bytes.NewReader([]byte("data")))
		req.Header.Set("Content-Type", ct)
		req.RemoteAddr = "127.0.0.1:1234"
		if w := serve(srv, req); w.Code != 400 {
			t.Errorf("content type %q: status %d, want 400", ct, w.Code)
		}
	}

	// The browser sends parameters along with the type; those must not defeat
	// the whitelist.
	req := httptest.NewRequest("POST", "/api/voice/transcribe", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "audio/webm;codecs=opus")
	req.RemoteAddr = "127.0.0.1:1234"
	w := serve(srv, req)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "empty recording") {
		t.Errorf("an accepted type with an empty body should report the empty body, got %d %s", w.Code, w.Body.String())
	}
}

// TestTranscribeRejectsOversizedBody proves the 8 MiB reader is wired, so a
// huge upload cannot be buffered whole on a 3.7 GiB Raspberry Pi.
func TestTranscribeRejectsOversizedBody(t *testing.T) {
	srv, _ := newVoiceServer(t, nil)

	big := bytes.Repeat([]byte("a"), (8<<20)+1024)
	req := httptest.NewRequest("POST", "/api/voice/transcribe", bytes.NewReader(big))
	req.Header.Set("Content-Type", "audio/webm")
	req.RemoteAddr = "127.0.0.1:1234"
	w := serve(srv, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status %d, want 413", w.Code)
	}
}

// TestVoiceEndpointsRequireAuth: dictation reaches a paid API and the
// meta-prompt steers what the model does, so neither may be open.
func TestVoiceEndpointsRequireAuth(t *testing.T) {
	srv, _ := newVoiceServer(t, func(c *config.Config) {
		c.LANSubnets = nil // no bypass
		c.Users = []config.User{{Username: "luis", PasswordHash: "x"}}
	})

	for _, ep := range []struct{ method, path string }{
		{"POST", "/api/voice/transcribe"},
		{"POST", "/api/voice/rewrite"},
		{"GET", "/api/voice/prompt"},
		{"PUT", "/api/voice/prompt"},
		{"POST", "/api/voice/prompt/reset"},
	} {
		req := httptest.NewRequest(ep.method, ep.path, bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "8.8.8.8:1234"
		if w := serve(srv, req); w.Code != 401 {
			t.Errorf("%s %s: status %d, want 401", ep.method, ep.path, w.Code)
		}
	}
}

// TestVoiceDisabledRefuses: turning the feature off must actually stop the
// outbound calls, not just hide the button.
func TestVoiceDisabledRefuses(t *testing.T) {
	srv, _ := newVoiceServer(t, func(c *config.Config) { c.Voice.Enabled = false })

	w := doJSON(t, srv, "POST", "/api/voice/rewrite", map[string]string{"text": "algo"})
	if w.Code != 403 {
		t.Errorf("status %d, want 403", w.Code)
	}
}

// TestMicrophonePermissionAllowsSelf is the regression for the header that
// silently disabled the whole feature: with microphone=() the browser denies
// getUserMedia with no error and no prompt.
func TestMicrophonePermissionAllowsSelf(t *testing.T) {
	srv := newTestServer(t)
	w := serve(srv, httptest.NewRequest("GET", "/api/health", nil))

	pp := w.Header().Get("Permissions-Policy")
	if !strings.Contains(pp, "microphone=(self)") {
		t.Errorf("Permissions-Policy must allow the mic for this origin: %q", pp)
	}
	if !strings.Contains(pp, "camera=()") {
		t.Errorf("the camera must stay denied: %q", pp)
	}
}
