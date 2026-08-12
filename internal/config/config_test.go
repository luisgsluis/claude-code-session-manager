package config

import (
	"os"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.AgentSocket != "/run/ccsm/agent.sock" {
		t.Errorf("expected default socket, got %s", cfg.AgentSocket)
	}
	if cfg.Rc.BootstrapProfile != "estandar" {
		t.Errorf("expected estandar, got %s", cfg.Rc.BootstrapProfile)
	}
	if cfg.Rc.WaitSeconds != 25 {
		t.Errorf("expected 25, got %d", cfg.Rc.WaitSeconds)
	}
	if cfg.Rc.PollSeconds != 2 {
		t.Errorf("expected 2, got %d", cfg.Rc.PollSeconds)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(empty): %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected 8080, got %d", cfg.Port)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/ccsm-config.yaml")
	if err != nil {
		t.Fatalf("Load(missing): %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port on missing file, got %d", cfg.Port)
	}
}

func TestLoadValidYAML(t *testing.T) {
	tmp := t.TempDir() + "/test.yaml"
	data := `
port: 9090
session_secret: "abc123"
lan_subnets:
  - "10.0.0.0/8"
users:
  - username: "luis"
    password_hash: "$2a$10$hash"
agent_socket: "/tmp/agent.sock"
agent_secret: "s3cr3t"
host_attach_addr: "admin@myhost"
rc:
  bootstrap_profile: "deepseek"
  wait_seconds: 30
  poll_seconds: 5
`
	if err := os.WriteFile(tmp, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load(valid): %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("port: %d", cfg.Port)
	}
	if cfg.SessionSecret != "abc123" {
		t.Errorf("secret: %s", cfg.SessionSecret)
	}
	if len(cfg.LANSubnets) != 1 || cfg.LANSubnets[0] != "10.0.0.0/8" {
		t.Errorf("subnets: %v", cfg.LANSubnets)
	}
	if len(cfg.Users) != 1 || cfg.Users[0].Username != "luis" {
		t.Errorf("users: %v", cfg.Users)
	}
	if cfg.AgentSocket != "/tmp/agent.sock" {
		t.Errorf("socket: %s", cfg.AgentSocket)
	}
	if cfg.AgentSecret != "s3cr3t" {
		t.Errorf("agent secret: %s", cfg.AgentSecret)
	}
	if cfg.HostAttachAddr != "admin@myhost" {
		t.Errorf("attach: %s", cfg.HostAttachAddr)
	}
	if cfg.Rc.BootstrapProfile != "deepseek" {
		t.Errorf("rc profile: %s", cfg.Rc.BootstrapProfile)
	}
	if cfg.Rc.WaitSeconds != 30 {
		t.Errorf("rc wait: %d", cfg.Rc.WaitSeconds)
	}
	if cfg.Rc.PollSeconds != 5 {
		t.Errorf("rc poll: %d", cfg.Rc.PollSeconds)
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("CCSM_PORT", "7070")
	os.Setenv("CCSM_SESSION_SECRET", "envsecret")
	os.Setenv("CCSM_LAN_SUBNETS", "192.168.1.0/24,10.0.0.0/8")
	os.Setenv("CCSM_AGENT_SOCKET", "/env/sock")
	os.Setenv("CCSM_AGENT_SECRET", "envagent")
	os.Setenv("CCSM_HOST_ATTACH_ADDR", "env@host")
	os.Setenv("CCSM_CONVERSATIONS_PATH", "/env/convs")
	os.Setenv("CCSM_PROFILES_PATH", "/env/profiles")
	os.Setenv("CCSM_SETTINGS_PATH", "/env/settings.json")
	os.Setenv("CCSM_RC_BOOTSTRAP_PROFILE", "envprofile")
	os.Setenv("CCSM_RC_WAIT_SECONDS", "42")
	os.Setenv("CCSM_RC_POLL_SECONDS", "3")
	os.Setenv("CCSM_CLAUDE_BINARY", "/env/claude")
	os.Setenv("CCSM_TMUX_BINARY", "/env/tmux")
	os.Setenv("CCSM_BASH_BINARY", "/env/bash")
	defer func() {
		for _, e := range []string{
			"CCSM_PORT", "CCSM_SESSION_SECRET", "CCSM_LAN_SUBNETS",
			"CCSM_AGENT_SOCKET", "CCSM_AGENT_SECRET", "CCSM_HOST_ATTACH_ADDR",
			"CCSM_CONVERSATIONS_PATH", "CCSM_PROFILES_PATH", "CCSM_SETTINGS_PATH",
			"CCSM_RC_BOOTSTRAP_PROFILE", "CCSM_RC_WAIT_SECONDS", "CCSM_RC_POLL_SECONDS",
			"CCSM_CLAUDE_BINARY", "CCSM_TMUX_BINARY", "CCSM_BASH_BINARY",
		} {
			os.Unsetenv(e)
		}
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 7070 {
		t.Errorf("port: %d", cfg.Port)
	}
	if cfg.SessionSecret != "envsecret" {
		t.Errorf("secret: %s", cfg.SessionSecret)
	}
	if len(cfg.LANSubnets) != 2 || cfg.LANSubnets[0] != "192.168.1.0/24" {
		t.Errorf("subnets: %v", cfg.LANSubnets)
	}
	if cfg.AgentSocket != "/env/sock" {
		t.Errorf("socket: %s", cfg.AgentSocket)
	}
	if cfg.AgentSecret != "envagent" {
		t.Errorf("agent: %s", cfg.AgentSecret)
	}
	if cfg.HostAttachAddr != "env@host" {
		t.Errorf("attach: %s", cfg.HostAttachAddr)
	}
	if cfg.ConversationsPath != "/env/convs" {
		t.Errorf("conversations: %s", cfg.ConversationsPath)
	}
	if cfg.ProfilesPath != "/env/profiles" {
		t.Errorf("profiles: %s", cfg.ProfilesPath)
	}
	if cfg.SettingsPath != "/env/settings.json" {
		t.Errorf("settings: %s", cfg.SettingsPath)
	}
	if cfg.Rc.BootstrapProfile != "envprofile" {
		t.Errorf("rc: %s", cfg.Rc.BootstrapProfile)
	}
	if cfg.ClaudeBinary != "/env/claude" {
		t.Errorf("claude: %s", cfg.ClaudeBinary)
	}
	if cfg.TmuxBinary != "/env/tmux" {
		t.Errorf("tmux: %s", cfg.TmuxBinary)
	}
	if cfg.BashBinary != "/env/bash" {
		t.Errorf("bash: %s", cfg.BashBinary)
	}
	if cfg.Rc.WaitSeconds != 42 {
		t.Errorf("rc wait: %d", cfg.Rc.WaitSeconds)
	}
	if cfg.Rc.PollSeconds != 3 {
		t.Errorf("rc poll: %d", cfg.Rc.PollSeconds)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmp := t.TempDir() + "/bad.yaml"
	if err := os.WriteFile(tmp, []byte("[unclosed"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(tmp)
	if err == nil {
		t.Fatal("expected parse error for invalid YAML")
	}
}

func TestLoadUnreadableFile(t *testing.T) {
	// os.ReadFile on a directory fails with an error that is not IsNotExist,
	// covering the "read config" error branch.
	_, err := Load(t.TempDir())
	if err == nil {
		t.Fatal("expected read error for directory path")
	}
}

func TestEnvInvalidNumbers(t *testing.T) {
	os.Setenv("CCSM_PORT", "not-a-number")
	os.Setenv("CCSM_RC_WAIT_SECONDS", "many")
	os.Setenv("CCSM_RC_POLL_SECONDS", "many")
	defer func() {
		os.Unsetenv("CCSM_PORT")
		os.Unsetenv("CCSM_RC_WAIT_SECONDS")
		os.Unsetenv("CCSM_RC_POLL_SECONDS")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("invalid CCSM_PORT should fall back to default, got %d", cfg.Port)
	}
	if cfg.Rc.WaitSeconds != 25 {
		t.Errorf("invalid wait should fall back to default, got %d", cfg.Rc.WaitSeconds)
	}
	if cfg.Rc.PollSeconds != 2 {
		t.Errorf("invalid poll should fall back to default, got %d", cfg.Rc.PollSeconds)
	}
}

func validConfig() *Config {
	cfg := Defaults()
	cfg.SessionSecret = "0123456789abcdef"
	cfg.AgentSecret = "0123456789abcdef"
	cfg.Users = []User{{Username: "luis", PasswordHash: "$2a$10$hash"}}
	return cfg
}

func TestValidateOK(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Errorf("expected valid config to pass, got %v", err)
	}
}

func TestValidateDirectModeNoAgentSecretRequired(t *testing.T) {
	cfg := validConfig()
	cfg.AgentSocket = ""
	cfg.AgentSecret = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("direct mode (empty agent_socket) should not require agent_secret: %v", err)
	}
}

func TestValidateRejectsEmptySessionSecret(t *testing.T) {
	cfg := validConfig()
	cfg.SessionSecret = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty session_secret")
	}
}

func TestValidateRejectsMissingAgentSecretInAgentMode(t *testing.T) {
	cfg := validConfig()
	cfg.AgentSocket = "/run/ccsm/agent.sock"
	cfg.AgentSecret = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing agent_secret with agent_socket set")
	}
}

func TestValidateAllowsNoUsers(t *testing.T) {
	// LAN-only deployments can skip the login form entirely (LANBypass), so
	// an empty user list is valid — e.g. the e2e config runs with users: [].
	cfg := validConfig()
	cfg.Users = nil
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for empty users (LAN-only deployment), got %v", err)
	}
}

func TestValidateRejectsBadPort(t *testing.T) {
	cfg := validConfig()
	cfg.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestValidateRejectsBadSettleSeconds(t *testing.T) {
	cfg := validConfig()
	cfg.Rc.SettleSeconds = -1
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative rc.settle_seconds")
	}
}

func TestEnvOverridesYAML(t *testing.T) {
	tmp := t.TempDir() + "/test.yaml"
	data := `port: 9090`
	if err := os.WriteFile(tmp, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("CCSM_PORT", "1234")
	defer os.Unsetenv("CCSM_PORT")

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 1234 {
		t.Errorf("env should override yaml, got %d", cfg.Port)
	}
}
