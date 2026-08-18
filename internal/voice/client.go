package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

// Timeouts for the two outbound calls. Transcription of a minute of speech is
// a couple of seconds on a fast provider; rewriting is a normal chat
// completion. Both are well inside the server's own 120s write timeout.
const (
	transcribeTimeout = 60 * time.Second
	rewriteTimeout    = 90 * time.Second
)

// Service performs the two outbound operations. It is the only place in CCSM
// that talks to anything outside the machine.
type Service struct {
	Cfg      func() config.VoiceConfig
	Profiles ProfileReader
	Store    PromptStore

	client *http.Client
}

// NewService builds the service with an explicit HTTP client, following the
// shape of internal/agent's client: own transport, own timeouts, nothing
// inherited from http.DefaultClient.
func NewService(cfg func() config.VoiceConfig, profiles ProfileReader, store PromptStore) *Service {
	return &Service{
		Cfg:      cfg,
		Profiles: profiles,
		Store:    store,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
				MaxIdleConns:        4,
				IdleConnTimeout:     60 * time.Second,
			},
		},
	}
}

// Error is a failure with the HTTP status the handler should report. Provider
// failures become 502: from the client's point of view CCSM's upstream is
// down, which is the same class of failure as the agent being unreachable.
type Error struct {
	Status int
	Msg    string
}

func (e *Error) Error() string { return e.Msg }

func errStatus(status int, format string, a ...any) *Error {
	return &Error{Status: status, Msg: fmt.Sprintf(format, a...)}
}

// provider looks up a configured provider by name and resolves its credential.
func (s *Service) provider(name string) (config.VoiceProvider, resolved, error) {
	cfg := s.Cfg()
	p, ok := cfg.Providers[name]
	if !ok {
		return p, resolved{}, errStatus(http.StatusBadRequest, "unknown voice provider: %s", name)
	}
	r, err := resolveProvider(p, s.Profiles)
	if err != nil {
		return p, resolved{}, errStatus(http.StatusBadGateway, "provider %s: %v", name, err)
	}
	return p, r, nil
}

// Transcribe sends recorded audio to the configured speech-to-text provider.
//
// The audio arrives as raw bytes and leaves as multipart, which is the shape
// the OpenAI-compatible /audio/transcriptions endpoint wants. CCSM never has
// to PARSE multipart — only build it — so no inbound multipart handling is
// introduced anywhere in the project.
func (s *Service) Transcribe(audio []byte, contentType, filename string) (string, error) {
	cfg := s.Cfg()
	if !cfg.Enabled {
		return "", errStatus(http.StatusForbidden, "voice is disabled")
	}
	if cfg.STT.Provider == "" {
		return "", errStatus(http.StatusBadRequest, "no speech-to-text provider configured")
	}
	p, r, err := s.provider(cfg.STT.Provider)
	if err != nil {
		return "", err
	}
	if !p.CanSTT() {
		return "", errStatus(http.StatusBadRequest, "provider %s has no stt_model", cfg.STT.Provider)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	// The provider picks its decoder from the filename's extension, so the
	// part needs the real content type rather than multipart's default
	// application/octet-stream. iOS records audio/mp4, everything else webm —
	// getting this wrong is a provider-side decode error, not a CCSM one.
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	h.Set("Content-Type", contentType)
	part, err := mw.CreatePart(h)
	if err != nil {
		return "", errStatus(http.StatusInternalServerError, "build request: %v", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", errStatus(http.StatusInternalServerError, "build request: %v", err)
	}
	_ = mw.WriteField("model", p.STTModel)
	_ = mw.WriteField("response_format", "json")
	if cfg.Language != "" {
		_ = mw.WriteField("language", cfg.Language)
	}
	// The vocabulary hint is what keeps dictated jargon from coming back
	// phonetically mangled.
	if v := strings.TrimSpace(cfg.STT.Vocabulary); v != "" {
		_ = mw.WriteField("prompt", v)
	}
	if err := mw.Close(); err != nil {
		return "", errStatus(http.StatusInternalServerError, "build request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.baseURL+"/audio/transcriptions", &body)
	if err != nil {
		return "", errStatus(http.StatusInternalServerError, "build request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	raw, err := s.do(req, transcribeTimeout)
	if err != nil {
		return "", err
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", errStatus(http.StatusBadGateway, "transcription: unexpected response")
	}
	return strings.TrimSpace(out.Text), nil
}

// RewriteResult is what the rewriter produces.
type RewriteResult struct {
	Role      string   `json:"role"`
	Prompt    string   `json:"prompt"`
	Questions []string `json:"questions"`
}

// Answer is one clarifying question and what the user replied.
type Answer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Rewrite turns dictated text into a structured prompt.
//
// When answers are supplied this is the second pass of the clarification
// round, and the model is told it may not ask again — one round only, so a
// model that keeps finding things to ask cannot loop the UI forever.
func (s *Service) Rewrite(text, role string, answers []Answer) (*RewriteResult, error) {
	cfg := s.Cfg()
	if !cfg.Enabled {
		return nil, errStatus(http.StatusForbidden, "voice is disabled")
	}
	if !cfg.Rewrite.Enabled {
		return nil, errStatus(http.StatusForbidden, "prompt rewriting is disabled")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errStatus(http.StatusBadRequest, "nothing to rewrite")
	}
	if cfg.Rewrite.Provider == "" {
		return nil, errStatus(http.StatusBadRequest, "no rewrite provider configured")
	}
	p, r, err := s.provider(cfg.Rewrite.Provider)
	if err != nil {
		return nil, err
	}
	model := cfg.Rewrite.Model
	if model == "" {
		model = p.ChatModel
	}
	if model == "" {
		return nil, errStatus(http.StatusBadRequest, "provider %s has no chat model", cfg.Rewrite.Provider)
	}

	prompt, err := s.Store.Active()
	if err != nil {
		return nil, errStatus(http.StatusInternalServerError, "meta-prompt: %v", err)
	}
	if role == "" {
		role = cfg.Rewrite.DefaultRole
	}
	if role == "" {
		role = AutoRole
	}
	if role != AutoRole && !prompt.HasRole(role) {
		return nil, errStatus(http.StatusBadRequest, "unknown role %q (available: %s)",
			role, strings.Join(prompt.RoleIDs(), ", "))
	}

	maxQ := cfg.Rewrite.MaxQuestions
	if len(answers) > 0 {
		maxQ = 0
	}
	system := prompt.System(role, maxQ)
	if len(answers) > 0 {
		system += "\n\n# This is the second pass\n\n" +
			"The user has answered your clarifying questions; their answers are included " +
			"below the request. Fold them into the rewrite. You may not ask anything " +
			`further: return "questions": [].`
	}
	if v := strings.TrimSpace(cfg.STT.Vocabulary); v != "" {
		system += "\n\n# Vocabulary\n\nThese are the correct spellings of terms this " +
			"user dictates. Treat them as authoritative when a transcribed word is close " +
			"to one of them:\n" + v
	}

	var user strings.Builder
	user.WriteString(text)
	if len(answers) > 0 {
		user.WriteString("\n\n---\nAnswers to your questions:\n")
		for _, a := range answers {
			fmt.Fprintf(&user, "- Q: %s\n  A: %s\n", a.Question, a.Answer)
		}
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user.String()},
		},
		// Both Groq and DeepSeek honour this; the tolerant parser below covers
		// providers that only mostly honour it.
		"response_format": map[string]string{"type": "json_object"},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errStatus(http.StatusInternalServerError, "build request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, errStatus(http.StatusInternalServerError, "build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	raw, err := s.do(req, rewriteTimeout)
	if err != nil {
		return nil, err
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &completion); err != nil || len(completion.Choices) == 0 {
		return nil, errStatus(http.StatusBadGateway, "rewrite: unexpected response")
	}

	res, err := parseRewrite(completion.Choices[0].Message.Content)
	if err != nil {
		return nil, errStatus(http.StatusBadGateway, "rewrite: %v", err)
	}
	if res.Role == "" || (res.Role != AutoRole && !prompt.HasRole(res.Role)) {
		// The model invented a role name. The rewrite is still usable, so
		// report the role that was asked for rather than failing the request.
		res.Role = role
	}
	if len(answers) > 0 {
		res.Questions = nil
	}
	if maxQ >= 0 && len(res.Questions) > maxQ {
		res.Questions = res.Questions[:maxQ]
	}
	return res, nil
}

// parseRewrite reads the model's JSON, tolerating the two things models do to
// "reply with JSON and nothing else": wrap it in a markdown fence, or put a
// sentence before it.
func parseRewrite(content string) (*RewriteResult, error) {
	s := strings.TrimSpace(content)
	if s == "" {
		return nil, fmt.Errorf("empty response")
	}
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}
	if !strings.HasPrefix(s, "{") {
		start := strings.Index(s, "{")
		end := strings.LastIndex(s, "}")
		if start < 0 || end <= start {
			return nil, fmt.Errorf("response was not JSON")
		}
		s = s[start : end+1]
	}
	var res RewriteResult
	if err := json.Unmarshal([]byte(s), &res); err != nil {
		return nil, fmt.Errorf("response was not JSON")
	}
	res.Prompt = strings.TrimSpace(res.Prompt)
	if res.Prompt == "" {
		return nil, fmt.Errorf("response carried no prompt")
	}
	kept := res.Questions[:0]
	for _, q := range res.Questions {
		if q = strings.TrimSpace(q); q != "" {
			kept = append(kept, q)
		}
	}
	res.Questions = kept
	return &res, nil
}

// do runs a request and returns its body, mapping any failure to an Error.
//
// The provider's own error body is never forwarded verbatim: it can echo the
// request, and the request carries the Authorization header.
func (s *Service) do(req *http.Request, timeout time.Duration) ([]byte, error) {
	client := *s.client
	client.Timeout = timeout

	resp, err := client.Do(req)
	if err != nil {
		return nil, errStatus(http.StatusBadGateway, "provider unreachable")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, errStatus(http.StatusBadGateway, "provider response could not be read")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errStatus(http.StatusBadGateway, "provider returned %d", resp.StatusCode)
	}
	return body, nil
}
