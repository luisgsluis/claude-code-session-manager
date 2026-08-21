package voice

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

// capture records what the fake provider received, so tests can assert on the
// request CCSM actually built rather than only on what it returned.
type capture struct {
	path    string
	auth    string
	fields  map[string]string
	file    []byte
	fileCT  string
	fname   string
	chatReq map[string]any
}

// fakeProvider stands in for Groq/DeepSeek. reply is served verbatim; status
// overrides the response code when non-zero.
func fakeProvider(t *testing.T, cap *capture, status int, reply string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.path = r.URL.Path
		cap.auth = r.Header.Get("Authorization")

		if strings.HasSuffix(r.URL.Path, "/audio/transcriptions") {
			_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Errorf("bad content type: %v", err)
			}
			mr := multipart.NewReader(r.Body, params["boundary"])
			cap.fields = map[string]string{}
			for {
				part, err := mr.NextPart()
				if err != nil {
					break
				}
				data, _ := io.ReadAll(part)
				if part.FormName() == "file" {
					cap.file = data
					cap.fileCT = part.Header.Get("Content-Type")
					cap.fname = part.FileName()
					continue
				}
				cap.fields[part.FormName()] = string(data)
			}
		} else {
			_ = json.NewDecoder(r.Body).Decode(&cap.chatReq)
		}

		if status != 0 {
			w.WriteHeader(status)
		}
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newService wires a Service against a fake provider with a known-good prompt
// saved and applied, so tests can assert on validPrompt's own text.
func newService(t *testing.T, url string, mutate func(*config.VoiceConfig)) *Service {
	t.Helper()
	dir := t.TempDir()
	store := PromptStore{Dir: dir}
	id, err := store.SaveNew(validPrompt, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetActive(id); err != nil {
		t.Fatal(err)
	}
	cfg := config.VoiceConfig{
		Enabled:  true,
		Language: "es",
		STT: config.VoiceSTTConfig{
			Mode: "whisper", Provider: "p", Vocabulary: "sonarr, tmux, macvlan",
		},
		Rewrite: config.VoiceRewriteConfig{
			Enabled: true, Provider: "p", DefaultRole: "auto",
		},
		Providers: map[string]config.VoiceProvider{
			"p": {BaseURL: url, APIKey: "sk-test", STTModels: []string{"whisper-x"}, ChatModels: []string{"chat-x"}},
		},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	return NewService(func() config.VoiceConfig { return cfg }, nil, store)
}

func TestTranscribeBuildsTheRequest(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, `{"text":"  arregla sonarr  "}`)
	s := newService(t, srv.URL, nil)

	got, err := s.Transcribe([]byte("RIFFfake"), "audio/mp4", "audio.m4a")
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if got != "arregla sonarr" {
		t.Errorf("text = %q (not trimmed?)", got)
	}
	if cap.path != "/audio/transcriptions" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.auth != "Bearer sk-test" {
		t.Errorf("auth = %q", cap.auth)
	}
	if string(cap.file) != "RIFFfake" {
		t.Errorf("audio body = %q", cap.file)
	}
	// iOS records mp4, everyone else webm. Forwarding the real type and a
	// matching filename is what lets the provider pick its decoder.
	if cap.fileCT != "audio/mp4" {
		t.Errorf("file part content type = %q", cap.fileCT)
	}
	if cap.fname != "audio.m4a" {
		t.Errorf("file name = %q", cap.fname)
	}
	if cap.fields["model"] != "whisper-x" {
		t.Errorf("model = %q", cap.fields["model"])
	}
	if cap.fields["language"] != "es" {
		t.Errorf("language = %q", cap.fields["language"])
	}
	// The vocabulary hint is the whole reason dictated jargon survives.
	if !strings.Contains(cap.fields["prompt"], "sonarr") {
		t.Errorf("vocabulary hint not sent: %q", cap.fields["prompt"])
	}
}

func TestTranscribeOmitsLanguageWhenAutodetecting(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, `{"text":"hi"}`)
	s := newService(t, srv.URL, func(c *config.VoiceConfig) { c.Language = "" })

	if _, err := s.Transcribe([]byte("x"), "audio/webm", "a.webm"); err != nil {
		t.Fatal(err)
	}
	if _, ok := cap.fields["language"]; ok {
		t.Error("an empty language must be omitted so the provider autodetects")
	}
}

// TestTranscribeModelOverridesProviderDefault: cfg.STT.Model, when set, wins
// over the provider's first catalog entry — mirrors
// TestRewriteModelOverridesProviderDefault for the transcription side.
func TestTranscribeModelOverridesProviderDefault(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, `{"text":"x"}`)
	s := newService(t, srv.URL, func(c *config.VoiceConfig) {
		p := c.Providers["p"]
		p.STTModels = []string{"whisper-x", "whisper-y"}
		c.Providers["p"] = p
		c.STT.Model = "whisper-y"
	})

	if _, err := s.Transcribe([]byte("x"), "audio/webm", "a.webm"); err != nil {
		t.Fatal(err)
	}
	if cap.fields["model"] != "whisper-y" {
		t.Errorf("model = %q, want the configured override", cap.fields["model"])
	}
}

func TestTranscribeRefusals(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, `{"text":"x"}`)

	cases := []struct {
		name   string
		mutate func(*config.VoiceConfig)
		status int
	}{
		{"voice disabled", func(c *config.VoiceConfig) { c.Enabled = false }, http.StatusForbidden},
		{"no provider", func(c *config.VoiceConfig) { c.STT.Provider = "" }, http.StatusBadRequest},
		{"unknown provider", func(c *config.VoiceConfig) { c.STT.Provider = "ghost" }, http.StatusBadRequest},
		{"provider cannot transcribe", func(c *config.VoiceConfig) {
			p := c.Providers["p"]
			p.STTModels = nil
			c.Providers["p"] = p
		}, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newService(t, srv.URL, c.mutate)
			_, err := s.Transcribe([]byte("x"), "audio/webm", "a.webm")
			ve, ok := err.(*Error)
			if !ok {
				t.Fatalf("expected a *voice.Error, got %v", err)
			}
			if ve.Status != c.status {
				t.Errorf("status = %d, want %d", ve.Status, c.status)
			}
		})
	}
}

// TestProviderErrorsBecome502AndSayNothing: the provider's own error body can
// echo the request back, and the request carries the Authorization header, so
// it must never be forwarded.
func TestProviderErrorsBecome502AndSayNothing(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, http.StatusUnauthorized,
		`{"error":"invalid api key sk-test, sent as Bearer sk-test"}`)
	s := newService(t, srv.URL, nil)

	_, err := s.Transcribe([]byte("x"), "audio/webm", "a.webm")
	ve, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected a *voice.Error, got %v", err)
	}
	if ve.Status != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", ve.Status)
	}
	if strings.Contains(ve.Msg, "sk-test") {
		t.Fatalf("the API key leaked through the error: %q", ve.Msg)
	}
}

func TestProviderUnreachable(t *testing.T) {
	// Port 1 on loopback: nothing listens, so the dial fails fast.
	s := newService(t, "http://127.0.0.1:1", nil)
	_, err := s.Transcribe([]byte("x"), "audio/webm", "a.webm")
	ve, ok := err.(*Error)
	if !ok || ve.Status != http.StatusBadGateway {
		t.Fatalf("expected a 502, got %v", err)
	}
}

// chatReply wraps content the way an OpenAI-compatible completion does.
func chatReply(content string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"content": content}}},
	})
	return string(b)
}

func TestRewriteForcedRole(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, chatReply(`{"role":"devops","question":null,"prompt":"Reinicia sonarr en la Pi."}`))
	s := newService(t, srv.URL, nil)

	res, err := s.Rewrite("pues eh reinicia el sonarr ese", "devops", nil)
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if res.Prompt != "Reinicia sonarr en la Pi." {
		t.Errorf("prompt = %q", res.Prompt)
	}
	if res.Role != "devops" {
		t.Errorf("role = %q", res.Role)
	}

	msgs := cap.chatReq["messages"].([]any)
	system := msgs[0].(map[string]any)["content"].(string)
	if !strings.Contains(system, "Operate running systems") {
		t.Error("the devops block did not reach the model")
	}
	if strings.Contains(system, "Write documentation") {
		t.Error("a forced role leaked another role's block into the system prompt")
	}
	if !strings.Contains(system, "sonarr") {
		t.Error("the vocabulary was not appended to the system prompt")
	}
	if cap.chatReq["model"] != "chat-x" {
		t.Errorf("model = %v", cap.chatReq["model"])
	}
}

func TestRewriteAutoRoleShowsEveryBlock(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, chatReply(`{"role":"docs","question":null,"prompt":"x"}`))
	s := newService(t, srv.URL, nil)

	res, err := s.Rewrite("actualiza el claude.md", "auto", nil)
	if err != nil {
		t.Fatal(err)
	}
	// In auto mode the model reports which role it picked, and that is what
	// the panel shows.
	if res.Role != "docs" {
		t.Errorf("role = %q, want the classified one", res.Role)
	}
	msgs := cap.chatReq["messages"].([]any)
	system := msgs[0].(map[string]any)["content"].(string)
	for _, want := range []string{"Operate running systems", "Write documentation", "Pick a role"} {
		if !strings.Contains(system, want) {
			t.Errorf("auto mode did not include %q", want)
		}
	}
}

// TestRewriteContinuesAskingMidLoop: one answer in hand is not the end of the
// loop — the model may still ask another question, one at a time, and the UI
// relies on that to show the next round.
func TestRewriteContinuesAskingMidLoop(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0,
		chatReply(`{"role":"devops","question":{"text":"¿y esto?","options":["a","b"]},"prompt":"provisional"}`))
	s := newService(t, srv.URL, nil)

	res, err := s.Rewrite("haz algo", "devops", []Answer{{Question: "¿cuál?", Answer: "el de la Pi"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Question == nil || res.Question.Text != "¿y esto?" {
		t.Errorf("a mid-loop question must reach the caller, got %+v", res.Question)
	}
	if len(res.Question.Options) != 2 {
		t.Errorf("options must survive parsing, got %v", res.Question.Options)
	}

	msgs := cap.chatReq["messages"].([]any)
	system := msgs[0].(map[string]any)["content"].(string)
	if !strings.Contains(system, "Continuing clarification") {
		t.Error("the model was not told this is a continuation")
	}
	if strings.Contains(system, "last round") {
		t.Error("a mid-loop round must not be told it is the last one")
	}
	user := msgs[1].(map[string]any)["content"].(string)
	if !strings.Contains(user, "el de la Pi") {
		t.Errorf("the answers did not reach the model: %q", user)
	}
}

// TestRewriteCapsRounds pins the loop guard: once the caller has accumulated
// maxClarifyRounds answers, the model is told to stop, and anything it asks
// anyway is dropped. Without this a model that keeps finding things to ask
// could bounce the panel forever.
func TestRewriteCapsRounds(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0,
		chatReply(`{"role":"devops","question":{"text":"¿otra más?"},"prompt":"listo"}`))
	s := newService(t, srv.URL, nil)

	answers := make([]Answer, maxClarifyRounds)
	for i := range answers {
		answers[i] = Answer{Question: "q", Answer: "a"}
	}

	res, err := s.Rewrite("haz algo", "devops", answers)
	if err != nil {
		t.Fatal(err)
	}
	if res.Question != nil {
		t.Errorf("the round cap must drop any question the model still asks, got %+v", res.Question)
	}
	msgs := cap.chatReq["messages"].([]any)
	system := msgs[0].(map[string]any)["content"].(string)
	if !strings.Contains(system, "last round") {
		t.Error("the model was not told this is the last round")
	}
}

func TestRewriteUnknownRoleFromModelFallsBack(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, chatReply(`{"role":"invented","question":null,"prompt":"x"}`))
	s := newService(t, srv.URL, nil)

	res, err := s.Rewrite("algo", "devops", nil)
	if err != nil {
		t.Fatalf("an invented role must not fail the request: %v", err)
	}
	if res.Role != "devops" {
		t.Errorf("role = %q, want the requested one", res.Role)
	}
}

func TestRewriteRefusals(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, chatReply(`{"role":"devops","prompt":"x"}`))

	cases := []struct {
		name   string
		text   string
		role   string
		mutate func(*config.VoiceConfig)
		status int
	}{
		{"voice disabled", "x", "", func(c *config.VoiceConfig) { c.Enabled = false }, http.StatusForbidden},
		{"rewrite disabled", "x", "", func(c *config.VoiceConfig) { c.Rewrite.Enabled = false }, http.StatusForbidden},
		{"empty text", "   ", "", nil, http.StatusBadRequest},
		{"unknown role", "x", "ghost", nil, http.StatusBadRequest},
		{"no provider", "x", "", func(c *config.VoiceConfig) { c.Rewrite.Provider = "" }, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newService(t, srv.URL, c.mutate)
			_, err := s.Rewrite(c.text, c.role, nil)
			ve, ok := err.(*Error)
			if !ok {
				t.Fatalf("expected a *voice.Error, got %v", err)
			}
			if ve.Status != c.status {
				t.Errorf("status = %d, want %d", ve.Status, c.status)
			}
		})
	}
}

// TestRewriteModelOverridesProviderDefault: the UI can change the model
// without touching the provider's credentials.
func TestRewriteModelOverridesProviderDefault(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, chatReply(`{"role":"devops","prompt":"x"}`))
	s := newService(t, srv.URL, func(c *config.VoiceConfig) { c.Rewrite.Model = "chosen-model" })

	if _, err := s.Rewrite("algo", "devops", nil); err != nil {
		t.Fatal(err)
	}
	if cap.chatReq["model"] != "chosen-model" {
		t.Errorf("model = %v, want the configured override", cap.chatReq["model"])
	}
}

// TestParseRewrite covers what models actually do to "reply with JSON only".
func TestParseRewrite(t *testing.T) {
	ok := []struct {
		name string
		in   string
		want string
	}{
		{"plain", `{"role":"devops","prompt":"hola"}`, "hola"},
		{"markdown fence", "```json\n{\"role\":\"devops\",\"prompt\":\"hola\"}\n```", "hola"},
		{"bare fence", "```\n{\"role\":\"devops\",\"prompt\":\"hola\"}\n```", "hola"},
		{"prose around it", "Sure! Here you go:\n{\"role\":\"devops\",\"prompt\":\"hola\"}\nHope that helps.", "hola"},
		{"leading whitespace", "\n\n  {\"role\":\"devops\",\"prompt\":\"hola\"}", "hola"},
	}
	for _, c := range ok {
		t.Run(c.name, func(t *testing.T) {
			res, err := parseRewrite(c.in)
			if err != nil {
				t.Fatalf("parseRewrite: %v", err)
			}
			if res.Prompt != c.want {
				t.Errorf("prompt = %q, want %q", res.Prompt, c.want)
			}
		})
	}

	bad := []struct{ name, in string }{
		{"empty", "   "},
		{"not json at all", "I cannot help with that."},
		{"json without a prompt", `{"role":"devops","question":null}`},
		{"prompt is only whitespace", `{"role":"devops","prompt":"   "}`},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseRewrite(c.in); err == nil {
				t.Error("expected an error")
			}
		})
	}

	t.Run("a question with blank text is treated as no question", func(t *testing.T) {
		res, err := parseRewrite(`{"role":"devops","prompt":"x","question":{"text":"   "}}`)
		if err != nil {
			t.Fatal(err)
		}
		if res.Question != nil {
			t.Errorf("question = %+v, want nil", res.Question)
		}
	})

	t.Run("blank options are dropped, real ones kept", func(t *testing.T) {
		res, err := parseRewrite(`{"role":"devops","prompt":"x","question":{"text":"¿cuál?","options":["  ","real option",""]}}`)
		if err != nil {
			t.Fatal(err)
		}
		if res.Question == nil || len(res.Question.Options) != 1 || res.Question.Options[0] != "real option" {
			t.Errorf("question = %+v", res.Question)
		}
	})

	t.Run("no question key means nothing is unclear", func(t *testing.T) {
		res, err := parseRewrite(`{"role":"devops","prompt":"x"}`)
		if err != nil {
			t.Fatal(err)
		}
		if res.Question != nil {
			t.Errorf("question = %+v, want nil", res.Question)
		}
	})
}

// TestRewriteGarbageResponseIs502: an unparseable answer is an upstream
// problem, not a client one.
func TestRewriteGarbageResponseIs502(t *testing.T) {
	var cap capture
	srv := fakeProvider(t, &cap, 0, chatReply("I'm afraid I can't do that."))
	s := newService(t, srv.URL, nil)

	_, err := s.Rewrite("algo", "devops", nil)
	ve, ok := err.(*Error)
	if !ok || ve.Status != http.StatusBadGateway {
		t.Fatalf("expected a 502, got %v", err)
	}
}
