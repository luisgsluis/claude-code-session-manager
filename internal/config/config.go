package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all CCSM configuration.
type Config struct {
	Port           int      `yaml:"port"`
	SessionSecret  string   `yaml:"session_secret"`
	LANSubnets     []string `yaml:"lan_subnets"`
	TrustedProxies []string `yaml:"trusted_proxies"` // reverse proxies whose X-Forwarded-For we trust
	Users          []User   `yaml:"users"`

	// Agent connection
	AgentSocket    string `yaml:"agent_socket"`
	AgentSecret    string `yaml:"agent_secret"`
	HostAttachAddr string `yaml:"host_attach_addr"` // shown in UI: "ssh admin@host -t tmux a -t N"

	// Paths used by direct (package) mode — "auto" resolves against $HOME.
	// In agent mode they are hints; the agent uses its own flags.
	ConversationsPath string `yaml:"conversations_path"`
	ProfilesPath      string `yaml:"profiles_path"`
	SettingsPath      string `yaml:"settings_path"`
	AuditPath         string `yaml:"audit_path"` // JSONL audit log; "auto" → $HOME/.ccsm/audit.jsonl

	// Host binaries for direct (package) mode. Empty → $HOME/.local/bin/claude,
	// /usr/bin/tmux, /usr/bin/bash. Ignored in agent mode.
	ClaudeBinary string `yaml:"claude_binary"`
	TmuxBinary   string `yaml:"tmux_binary"`
	BashBinary   string `yaml:"bash_binary"`

	// Remote Control bootstrap
	Rc RcConfig `yaml:"rc"`

	// Voice dictation and prompt rewriting
	Voice VoiceConfig `yaml:"voice"`
}

// VoiceConfig configures the dictation button and the prompt rewriter.
//
// The split between this and VoiceProvider is deliberate: everything here is
// a choice ("which provider", "which mode") and is hot-reloadable through
// PATCH /api/config, while Providers holds credentials and is file-only. That
// is what lets the UI switch provider without ever being able to read, write
// or leak a key.
type VoiceConfig struct {
	Enabled bool `yaml:"enabled"`
	// Language is a hint for the transcriber only ("" = autodetect). It does
	// NOT decide the language of the rewritten prompt: the meta-prompt detects
	// the language of what was said and answers in that same language.
	Language    string                   `yaml:"language"`
	STT         VoiceSTTConfig           `yaml:"stt"`
	Rewrite     VoiceRewriteConfig       `yaml:"rewrite"`
	PromptsPath string                   `yaml:"prompts_path"`
	Providers   map[string]VoiceProvider `yaml:"providers"`
}

// VoiceSTTConfig selects how speech becomes text.
type VoiceSTTConfig struct {
	// Mode is "whisper" (record and transcribe server-side), "webspeech" (the
	// browser's own recogniser, no server involvement) or "whisper_fallback"
	// (whisper when the browser can record and a provider is configured,
	// otherwise webspeech).
	Mode     string `yaml:"mode"`
	Provider string `yaml:"provider"`
	// Vocabulary is passed to the transcriber as a hint. It is what stops
	// dictated jargon coming back as "sonar", "mac blanc" and "sistema D".
	Vocabulary string `yaml:"vocabulary"`
}

// VoiceRewriteConfig selects who turns dictated text into a prompt.
type VoiceRewriteConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Provider    string `yaml:"provider"`
	Model       string `yaml:"model"`
	DefaultRole string `yaml:"default_role"`
}

// VoiceProvider is one OpenAI-compatible endpoint. Groq and DeepSeek both
// speak that dialect, so a single client covers them and anything added later.
//
// The four key sources are mutually exclusive and exactly one must be set;
// Validate enforces that so a provider can never silently authenticate with
// the wrong credential. None of them is ever exposed by the API.
type VoiceProvider struct {
	BaseURL string `yaml:"base_url"`

	APIKey       string `yaml:"api_key"`        // literal, in config.yaml (already 0600)
	APIKeyEnv    string `yaml:"api_key_env"`    // name of an environment variable
	APIKeyHelper string `yaml:"api_key_helper"` // executable printing the key on stdout
	FromProfile  string `yaml:"from_profile"`   // a Claude Code profile to borrow apiKeyHelper/base URL from

	STTModel  string `yaml:"stt_model"`
	ChatModel string `yaml:"chat_model"`
}

// KeySources counts how many credential origins this provider declares.
func (p VoiceProvider) KeySources() int {
	n := 0
	for _, s := range []string{p.APIKey, p.APIKeyEnv, p.APIKeyHelper, p.FromProfile} {
		if s != "" {
			n++
		}
	}
	return n
}

// CanSTT reports whether this provider is usable for transcription.
func (p VoiceProvider) CanSTT() bool { return p.STTModel != "" }

// CanChat reports whether this provider is usable for rewriting.
func (p VoiceProvider) CanChat() bool { return p.ChatModel != "" }

// Valid STT modes, exported so the server can reject anything else with a
// message naming the alternatives rather than failing obscurely later.
var VoiceSTTModes = []string{"whisper", "webspeech", "whisper_fallback"}

func validSTTMode(mode string) bool {
	for _, m := range VoiceSTTModes {
		if m == mode {
			return true
		}
	}
	return false
}

// RcConfig is the Remote Control bootstrap configuration.
type RcConfig struct {
	BootstrapProfile string `yaml:"bootstrap_profile"`
	WaitSeconds      int    `yaml:"wait_seconds"`
	PollSeconds      int    `yaml:"poll_seconds"`
	SettleSeconds    int    `yaml:"settle_seconds"` // margin after resume idle+bridge before restoring the target profile
}

// User defines an authenticated user. TOTPSecret holds the base32 TOTP secret
// when the user enrolled a second factor; empty means 2FA is off for them
// (2FA is opt-in per user). One field rather than a separate enabled flag, so
// "enabled without a secret" is not a state that can exist. omitempty matters:
// writeConfig re-serializes the whole Config, and without it every user would
// grow a `totp_secret: ""` line.
type User struct {
	Username     string `yaml:"username"`
	PasswordHash string `yaml:"password_hash"`
	TOTPSecret   string `yaml:"totp_secret,omitempty"`
}

// Validate checks the Config for invalid values. Returns nil if valid,
// or an error describing the first problem found.
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be 1-65535, got %d", c.Port)
	}
	if c.Rc.WaitSeconds < 5 || c.Rc.WaitSeconds > 120 {
		return fmt.Errorf("rc.wait_seconds must be 5-120, got %d", c.Rc.WaitSeconds)
	}
	if c.Rc.PollSeconds < 1 || c.Rc.PollSeconds > 10 {
		return fmt.Errorf("rc.poll_seconds must be 1-10, got %d", c.Rc.PollSeconds)
	}
	if c.Rc.SettleSeconds < 0 || c.Rc.SettleSeconds > 60 {
		return fmt.Errorf("rc.settle_seconds must be 0-60, got %d", c.Rc.SettleSeconds)
	}
	if len(c.SessionSecret) < 16 {
		return fmt.Errorf("session_secret must be set (min 16 chars) — generate one with 'ccsm --generate-secret'")
	}
	if c.AgentSocket != "" && len(c.AgentSecret) < 16 {
		return fmt.Errorf("agent_secret must be set (min 16 chars) when agent_socket is set — generate one with 'ccsm --generate-agent-secret'")
	}
	for _, u := range c.Users {
		if !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`).MatchString(u.Username) {
			return fmt.Errorf("invalid username: %s", u.Username)
		}
	}
	for _, s := range c.LANSubnets {
		if _, _, err := net.ParseCIDR(s); err != nil {
			return fmt.Errorf("invalid CIDR: %s", s)
		}
	}
	if err := c.Voice.validate(); err != nil {
		return err
	}
	return nil
}

// validate checks the voice block. Every failure here is a startup failure by
// design: a provider that names a key source it does not have, or a mode that
// does not exist, would otherwise surface as a confusing 502 the first time
// someone pressed the microphone.
func (v *VoiceConfig) validate() error {
	if !validSTTMode(v.STT.Mode) {
		return fmt.Errorf("voice.stt.mode must be one of %s, got %q",
			strings.Join(VoiceSTTModes, ", "), v.STT.Mode)
	}
	for name, p := range v.Providers {
		if !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`).MatchString(name) {
			return fmt.Errorf("invalid voice provider name: %s", name)
		}
		switch p.KeySources() {
		case 1:
			// exactly one, as required
		case 0:
			return fmt.Errorf("voice provider %q sets no key source (use one of api_key, api_key_env, api_key_helper, from_profile)", name)
		default:
			return fmt.Errorf("voice provider %q sets more than one key source; exactly one of api_key, api_key_env, api_key_helper, from_profile is allowed", name)
		}
		// from_profile supplies the base URL, so only the other three need one
		// declared up front.
		if p.BaseURL == "" && p.FromProfile == "" {
			return fmt.Errorf("voice provider %q needs a base_url", name)
		}
	}
	// Only check the selected providers when the feature is on: a disabled
	// voice block with a half-written provider must not stop CCSM booting.
	if !v.Enabled {
		return nil
	}
	if v.STT.Mode != "webspeech" && v.STT.Provider != "" {
		p, ok := v.Providers[v.STT.Provider]
		if !ok {
			return fmt.Errorf("voice.stt.provider %q is not defined in voice.providers", v.STT.Provider)
		}
		if !p.CanSTT() {
			return fmt.Errorf("voice.stt.provider %q has no stt_model", v.STT.Provider)
		}
	}
	if v.Rewrite.Enabled && v.Rewrite.Provider != "" {
		p, ok := v.Providers[v.Rewrite.Provider]
		if !ok {
			return fmt.Errorf("voice.rewrite.provider %q is not defined in voice.providers", v.Rewrite.Provider)
		}
		if !p.CanChat() && v.Rewrite.Model == "" {
			return fmt.Errorf("voice.rewrite.provider %q has no chat_model and voice.rewrite.model is empty", v.Rewrite.Provider)
		}
	}
	return nil
}

// Defaults returns a Config with sensible defaults.
func Defaults() *Config {
	return &Config{
		Port:              8080,
		AgentSocket:       "/run/ccsm/agent.sock",
		HostAttachAddr:    "localhost",
		ConversationsPath: "auto",
		ProfilesPath:      "auto",
		SettingsPath:      "auto",
		AuditPath:         "auto",
		Rc: RcConfig{
			BootstrapProfile: "estandar",
			WaitSeconds:      25,
			PollSeconds:      2,
			SettleSeconds:    1,
		},
		Voice: VoiceConfig{
			// Off by default: the feature needs a provider and a key, and an
			// install that has neither should not show a button that 502s.
			Enabled:     false,
			STT:         VoiceSTTConfig{Mode: "whisper_fallback"},
			Rewrite:     VoiceRewriteConfig{Enabled: true, DefaultRole: "auto"},
			PromptsPath: "auto",
		},
	}
}

// Load reads config from a YAML file and overlays environment variables.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read config: %w", err)
			}
			// File not found: use defaults + env
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config: %w", err)
			}
		}
	}

	applyEnv(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("CCSM_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}
	if v := os.Getenv("CCSM_SESSION_SECRET"); v != "" {
		cfg.SessionSecret = v
	}
	if v := os.Getenv("CCSM_LAN_SUBNETS"); v != "" {
		parts := strings.Split(v, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		cfg.LANSubnets = parts
	}
	if v := os.Getenv("CCSM_AGENT_SOCKET"); v != "" {
		cfg.AgentSocket = v
	}
	if v := os.Getenv("CCSM_AGENT_SECRET"); v != "" {
		cfg.AgentSecret = v
	}
	if v := os.Getenv("CCSM_HOST_ATTACH_ADDR"); v != "" {
		cfg.HostAttachAddr = v
	}
	if v := os.Getenv("CCSM_CONVERSATIONS_PATH"); v != "" {
		cfg.ConversationsPath = v
	}
	if v := os.Getenv("CCSM_PROFILES_PATH"); v != "" {
		cfg.ProfilesPath = v
	}
	if v := os.Getenv("CCSM_SETTINGS_PATH"); v != "" {
		cfg.SettingsPath = v
	}
	if v := os.Getenv("CCSM_AUDIT_PATH"); v != "" {
		cfg.AuditPath = v
	}
	if v := os.Getenv("CCSM_CLAUDE_BINARY"); v != "" {
		cfg.ClaudeBinary = v
	}
	if v := os.Getenv("CCSM_TMUX_BINARY"); v != "" {
		cfg.TmuxBinary = v
	}
	if v := os.Getenv("CCSM_BASH_BINARY"); v != "" {
		cfg.BashBinary = v
	}
	if v := os.Getenv("CCSM_RC_BOOTSTRAP_PROFILE"); v != "" {
		cfg.Rc.BootstrapProfile = v
	}
	if v := os.Getenv("CCSM_RC_WAIT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Rc.WaitSeconds = n
		}
	}
	if v := os.Getenv("CCSM_RC_POLL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Rc.PollSeconds = n
		}
	}
	if v := os.Getenv("CCSM_RC_SETTLE_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Rc.SettleSeconds = n
		}
	}
	if v := os.Getenv("CCSM_VOICE_ENABLED"); v != "" {
		cfg.Voice.Enabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv("CCSM_VOICE_LANGUAGE"); v != "" {
		cfg.Voice.Language = v
	}
	if v := os.Getenv("CCSM_VOICE_STT_MODE"); v != "" {
		cfg.Voice.STT.Mode = v
	}
	if v := os.Getenv("CCSM_VOICE_STT_PROVIDER"); v != "" {
		cfg.Voice.STT.Provider = v
	}
	if v := os.Getenv("CCSM_VOICE_STT_VOCABULARY"); v != "" {
		cfg.Voice.STT.Vocabulary = v
	}
	if v := os.Getenv("CCSM_VOICE_REWRITE_PROVIDER"); v != "" {
		cfg.Voice.Rewrite.Provider = v
	}
	if v := os.Getenv("CCSM_VOICE_REWRITE_MODEL"); v != "" {
		cfg.Voice.Rewrite.Model = v
	}
	if v := os.Getenv("CCSM_VOICE_PROMPTS_PATH"); v != "" {
		cfg.Voice.PromptsPath = v
	}
	// Per-provider key from the environment, so a container can be handed a
	// key without it ever touching config.yaml:
	//   CCSM_VOICE_KEY_GROQ=gsk_...  →  providers["groq"].api_key
	for _, e := range os.Environ() {
		const prefix = "CCSM_VOICE_KEY_"
		if !strings.HasPrefix(e, prefix) {
			continue
		}
		kv := strings.SplitN(strings.TrimPrefix(e, prefix), "=", 2)
		if len(kv) != 2 || kv[1] == "" {
			continue
		}
		name := strings.ToLower(kv[0])
		p, ok := cfg.Voice.Providers[name]
		if !ok {
			continue // only override providers that exist in the file
		}
		// Replaces whatever source the file declared, keeping "exactly one".
		p.APIKey, p.APIKeyEnv, p.APIKeyHelper, p.FromProfile = kv[1], "", "", ""
		cfg.Voice.Providers[name] = p
	}
}
