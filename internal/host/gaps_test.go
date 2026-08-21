package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRCStateIdempotent(t *testing.T) {
	cases := map[string]string{
		"ok":           "rc_connected",
		"fail":         "rc_failed",
		"rc_connected": "rc_connected",
		"rc_failed":    "rc_failed",
		"timeout":      "rc_pending",
		"dead":         "rc_pending",
		"rc_pending":   "rc_pending",
	}
	for in, want := range cases {
		if got := rcState(in); got != want {
			t.Errorf("rcState(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClaudeNewNoActiveProfile(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected"})
	// No settings.json → activeProfileName "". Plain --remote-control start.
	data, err := h.Exec("claude-nueva", nil)
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	if out := data.(map[string]string); out["session"] != "3" || out["status"] != "rc_connected" {
		t.Errorf("out: %+v", out)
	}
}

func TestClaudeResumeCleanActiveProfile(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	if err := h.applyProfile("estandar"); err != nil {
		t.Fatal(err)
	}
	id := "00000000-0000-0000-0000-0000000000dd"
	os.WriteFile(h.convPath+"/"+id+".jsonl", []byte(`{"type":"user","message":{"content":"hola"}}`+"\n"), 0600)

	data, err := h.Exec("claude-resume", map[string]string{"id": id})
	if err != nil {
		t.Fatalf("claude-resume: %v", err)
	}
	if out := data.(map[string]string); out["status"] != "rc_connected" {
		t.Errorf("clean active profile should use the plain path, got %+v", out)
	}
}

func TestClaudeResumeNewSessionFailure(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_NEW_FAIL": "1"})
	id := "00000000-0000-0000-0000-0000000000ee"
	os.WriteFile(h.convPath+"/"+id+".jsonl", []byte(`{"type":"user","message":{"content":"hola"}}`+"\n"), 0600)

	_, err := h.Exec("claude-resume", map[string]string{"id": id})
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestClaudePerfilApplySuccess(t *testing.T) {
	h := fakeHost(t, nil)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	if _, err := h.Exec("claude-perfil", map[string]string{"profile": "estandar"}); err != nil {
		t.Fatalf("claude-perfil: %v", err)
	}
	if got := h.readSettings(t); !strings.Contains(got, "sonnet") {
		t.Errorf("settings: %s", got)
	}
}

func TestApplyProfileWriteError(t *testing.T) {
	h := fakeHost(t, nil)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	// settings path whose parent directory does not exist → write fails.
	h.settingsPath = filepath.Join(t.TempDir(), "nope", "settings.json")
	err := h.applyProfile("estandar")
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestActiveProfileNameJunkEntries(t *testing.T) {
	h := fakeHost(t, nil)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	// Junk that the loop must skip.
	os.MkdirAll(h.profilesPath+"/subdir", 0700)
	os.WriteFile(h.profilesPath+"/notjson.txt", []byte("{}"), 0600)
	os.WriteFile(h.profilesPath+"/BAd name.json", []byte(`{}`), 0600)
	os.WriteFile(h.settingsPath, []byte(`{"other":1}`), 0600)
	if got := h.activeProfileName(); got != "" {
		t.Errorf("expected no match, got %q", got)
	}
}

func TestConversationsListSkipsSubdirs(t *testing.T) {
	h := fakeHost(t, nil)
	os.MkdirAll(h.convPath+"/subdir", 0700)
	data, err := h.Exec("conversations-ls", nil)
	if err != nil {
		t.Fatal(err)
	}
	if list := data.([]map[string]any); len(list) != 0 {
		t.Errorf("subdir leaked: %v", list)
	}
}

func TestConversationGetLinesLimit(t *testing.T) {
	h := fakeHost(t, nil)
	id := "00000000-0000-0000-0000-0000000000ee"
	var buf strings.Builder
	for i := 0; i < 10; i++ {
		buf.WriteString(`{"type":"user","cwd":"/home/admin/x","message":{"content":"msg` + string(rune('0'+i)) + `"}}` + "\n")
	}
	os.WriteFile(h.convPath+"/"+id+".jsonl", []byte(buf.String()), 0600)

	data, err := h.Exec("conversation-get", map[string]string{"id": id, "lines": "5"})
	if err != nil {
		t.Fatal(err)
	}
	conv := data.(map[string]any)
	msgs := conv["messages"].([]map[string]any)
	if len(msgs) != 5 {
		t.Fatalf("want last 5, got %d", len(msgs))
	}
	if msgs[0]["content"].(string) != "msg5" {
		t.Errorf("first kept message: %v", msgs[0]["content"])
	}
}

func TestNewUsesHomeEnv(t *testing.T) {
	t.Setenv("HOME", "/home/x")
	h := New(Options{})
	if h.profilesPath != "/home/x/claude-shared/claude-perfiles" {
		t.Errorf("profiles: %s", h.profilesPath)
	}
	if h.home != "/home/x" {
		t.Errorf("home: %s", h.home)
	}
}

func TestAliveConversationsConvDirMissing(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	os.MkdirAll(binDir, 0700)
	os.WriteFile(binDir+"/ps", []byte(fakePs), 0700)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	h := fakeHost(t, map[string]string{"FAKE_PS_OUT": "claude /home/x/claude\n"})
	os.RemoveAll(h.convPath)
	if len(h.aliveConversations()) != 0 {
		t.Error("expected empty with conv dir gone")
	}
}

func TestTmuxListNonNumericNames(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LIST": "xyz\t2024-01-01 10:00:00\tt\nabc\t2024-01-01 10:00:00\tt\n"})
	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatal(err)
	}
	sessions := data.([]map[string]any)
	if len(sessions) != 2 || sessions[0]["name"] != "abc" || sessions[1]["name"] != "xyz" {
		t.Errorf("alphabetical fallback: %v", sessions)
	}
}

func TestTmuxListNameOnlyLine(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LIST": "3\n"})
	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatal(err)
	}
	sessions := data.([]map[string]any)
	if len(sessions) != 1 || sessions[0]["name"] != "3" {
		t.Fatalf("sessions: %v", sessions)
	}
	if _, hasCreated := sessions[0]["created"]; hasCreated {
		t.Error("created set for name-only line")
	}
	if _, hasTask := sessions[0]["task"]; hasTask {
		t.Error("task set for name-only line")
	}
}

func TestLanzarConStagingNewSessionFailureRestores(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_NEW_FAIL": "1"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)

	_, err := h.Exec("claude-nueva", map[string]string{"profile": "deepseek"})
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
	// Best-effort restore of the target profile.
	if got := h.readSettings(t); !strings.Contains(got, "apiKeyHelper") {
		t.Errorf("target not restored after failure: %s", got)
	}
}
