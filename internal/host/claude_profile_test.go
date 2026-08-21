package host

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestAuthFieldsAltAuthAndEqual exercises the helper claudeProfile relies on
// to detect a credential change across a profile switch: which fields count
// as "alternate credentials", and whether two profiles resolve credentials
// the same way.
func TestAuthFieldsAltAuthAndEqual(t *testing.T) {
	deepseek := parseAuthFields([]byte(`{"apiKeyHelper":"/x/claude-apikey","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"}}`))
	deepseekSame := parseAuthFields([]byte(`{"apiKeyHelper":"/x/claude-apikey","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic","theme":"light"}}`))
	estandar := parseAuthFields([]byte(`{"model":"sonnet"}`))

	if !deepseek.altAuth() {
		t.Error("deepseek profile should count as alt auth")
	}
	if estandar.altAuth() {
		t.Error("clean profile should not count as alt auth")
	}
	if !deepseek.equal(deepseekSame) {
		t.Error("same credentials, different unrelated fields, should compare equal")
	}
	if deepseek.equal(estandar) {
		t.Error("deepseek vs estandar should not compare equal")
	}
}

// TestClaudeProfileRelaunchesOnAuthLoss: switching away from a profile with
// alternate credentials (apiKeyHelper) to a clean one must relaunch a live,
// idle session instead of just writing settings.json — a live process keeps
// using the stale key against the new endpoint and gets stuck retrying 401
// (visto 2026-08-15, sesión 0).
func TestClaudeProfileRelaunchesOnAuthLoss(t *testing.T) {
	id := "a1b2c3d4-1111-2222-3333-444455556666"
	pid := spawnRcSession(t, id)
	kills := t.TempDir() + "/kills.txt"
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST":         "3\t2026-01-01 00:00:00\tclaude\n",
		"FAKE_TMUX_LINE":         "  auto mode on (shift+tab to cycle) · esc to interrupt · ← for agents",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
		"FAKE_TMUX_KILLS":        kills,
		"FAKE_TMUX_NEW_NAME":     "5",
	})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"}}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.convPath+"/"+id+".jsonl", []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("claude-perfil", map[string]string{"profile": "estandar"})
	if err != nil {
		t.Fatalf("claude-perfil: %v", err)
	}
	out := data.(map[string]any)
	relaunched, _ := out["relaunched"].([]string)
	if len(relaunched) != 1 || relaunched[0] != "5" {
		t.Errorf("relaunched = %v, want [5]", relaunched)
	}
	kdata, _ := os.ReadFile(kills)
	if !strings.Contains(string(kdata), "3") {
		t.Errorf("old session not killed: %q", string(kdata))
	}
	if got := h.readSettings(t); strings.Contains(got, "apiKeyHelper") {
		t.Errorf("settings should be the clean target profile: %s", got)
	}
}

// TestClaudeProfileRelaunchKeepsSessionName: the relaunched session must keep
// its tmux name — before this it always jumped to a new auto-assigned number
// on every profile-triggered relaunch, which is confusing when nothing else
// about the session changed. FAKE_TMUX_KILL_MARKS_DEAD makes "3" report dead
// once actually killed, matching a real kill-session, so the request for that
// name back succeeds.
func TestClaudeProfileRelaunchKeepsSessionName(t *testing.T) {
	id := "a1b2c3d4-1111-2222-3333-444455556666"
	pid := spawnRcSession(t, id)
	kills := t.TempDir() + "/kills.txt"
	newArgs := t.TempDir() + "/newargs.txt"
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST":            "3\t2026-01-01 00:00:00\tclaude\n",
		"FAKE_TMUX_LINE":            "  auto mode on (shift+tab to cycle) · esc to interrupt · ← for agents",
		"FAKE_TMUX_PANE_SESSION":    "3",
		"FAKE_TMUX_PANE_PID":        strconv.Itoa(pid),
		"FAKE_TMUX_KILLS":           kills,
		"FAKE_TMUX_NEW_ARGS":        newArgs,
		"FAKE_TMUX_KILL_MARKS_DEAD": "1",
	})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"}}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.convPath+"/"+id+".jsonl", []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("claude-perfil", map[string]string{"profile": "estandar"})
	if err != nil {
		t.Fatalf("claude-perfil: %v", err)
	}
	out := data.(map[string]any)
	relaunched, _ := out["relaunched"].([]string)
	if len(relaunched) != 1 || relaunched[0] != "3" {
		t.Errorf("relaunched = %v, want [3] (same name as before)", relaunched)
	}
	nargs, _ := os.ReadFile(newArgs)
	if !strings.Contains(string(nargs), "-s 3") {
		t.Errorf("new-session should have requested the same name back (-s 3): %q", string(nargs))
	}
}

// TestClaudeProfileNoRelaunchSameAuth: switching between two profiles that
// resolve credentials identically (only cosmetic fields differ) must not
// touch any live session.
func TestClaudeProfileNoRelaunchSameAuth(t *testing.T) {
	kills := t.TempDir() + "/kills.txt"
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST":         "3\t2026-01-01 00:00:00\tclaude\n",
		"FAKE_TMUX_LINE":         "  auto mode on (shift+tab to cycle) · esc to interrupt · ← for agents",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_KILLS":        kills,
	})
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"},"theme":"light"}`)
	h.writeProfile(t, "deepseek2", `{"apiKeyHelper":"/x","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"},"theme":"dark"}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("claude-perfil", map[string]string{"profile": "deepseek2"})
	if err != nil {
		t.Fatalf("claude-perfil: %v", err)
	}
	out := data.(map[string]any)
	if relaunched, _ := out["relaunched"].([]string); len(relaunched) != 0 {
		t.Errorf("relaunched = %v, want none (same credentials)", relaunched)
	}
	if kdata, _ := os.ReadFile(kills); len(kdata) != 0 {
		t.Errorf("no session should have been killed: %q", string(kdata))
	}
}

// TestClaudeProfileNoRelaunchEnteringAltAuth: switching FROM a clean profile
// INTO one with alternate credentials is the direction that's confirmed to
// hot-reload correctly (apiKeyHelper is invoked fresh) — must not relaunch.
func TestClaudeProfileNoRelaunchEnteringAltAuth(t *testing.T) {
	kills := t.TempDir() + "/kills.txt"
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST":         "3\t2026-01-01 00:00:00\tclaude\n",
		"FAKE_TMUX_LINE":         "  auto mode on (shift+tab to cycle) · esc to interrupt · ← for agents",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_KILLS":        kills,
	})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"}}`)
	if err := h.applyProfile("estandar"); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("claude-perfil", map[string]string{"profile": "deepseek"})
	if err != nil {
		t.Fatalf("claude-perfil: %v", err)
	}
	out := data.(map[string]any)
	if relaunched, _ := out["relaunched"].([]string); len(relaunched) != 0 {
		t.Errorf("relaunched = %v, want none (entering alt auth hot-reloads fine)", relaunched)
	}
	if kdata, _ := os.ReadFile(kills); len(kdata) != 0 {
		t.Errorf("no session should have been killed: %q", string(kdata))
	}
}

// TestClaudeProfileSkipsBusySession: a session mid-generation must not be
// killed out from under a running turn; it's left on stale credentials until
// it's idle (matches sessionRc's same guard).
func TestClaudeProfileSkipsBusySession(t *testing.T) {
	kills := t.TempDir() + "/kills.txt"
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST":         "3\t2026-01-01 00:00:00\tclaude\n",
		"FAKE_TMUX_LINE":         "  auto mode on (shift+tab to cycle) · esc to interrupt · ctrl+t to hide tasks",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_KILLS":        kills,
	})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"}}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("claude-perfil", map[string]string{"profile": "estandar"})
	if err != nil {
		t.Fatalf("claude-perfil: %v", err)
	}
	out := data.(map[string]any)
	if relaunched, _ := out["relaunched"].([]string); len(relaunched) != 0 {
		t.Errorf("relaunched = %v, want none (session busy)", relaunched)
	}
	if kdata, _ := os.ReadFile(kills); len(kdata) != 0 {
		t.Errorf("busy session should not have been killed: %q", string(kdata))
	}
}
