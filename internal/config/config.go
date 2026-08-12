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
}

// RcConfig is the Remote Control bootstrap configuration.
type RcConfig struct {
	BootstrapProfile string `yaml:"bootstrap_profile"`
	WaitSeconds      int    `yaml:"wait_seconds"`
	PollSeconds      int    `yaml:"poll_seconds"`
	SettleSeconds    int    `yaml:"settle_seconds"` // margin after resume idle+bridge before restoring the target profile
}

// User defines an authenticated user.
type User struct {
	Username     string `yaml:"username"`
	PasswordHash string `yaml:"password_hash"`
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
}
