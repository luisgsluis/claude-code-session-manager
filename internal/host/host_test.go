package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPerfilSinRC(t *testing.T) {
	cases := []struct {
		name string
		json string
		want bool
	}{
		{"apiKeyHelper", `{"apiKeyHelper":"/x/claude-apikey"}`, true},
		{"env api key", `{"env":{"ANTHROPIC_API_KEY":"sk-x"}}`, true},
		{"env auth token", `{"env":{"ANTHROPIC_AUTH_TOKEN":"tok"}}`, true},
		{"non-anthropic base url", `{"env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"}}`, true},
		{"anthropic base url", `{"env":{"ANTHROPIC_BASE_URL":"https://api.anthropic.com"}}`, false},
		{"anthropic bare host", `{"env":{"ANTHROPIC_BASE_URL":"api.anthropic.com"}}`, false},
		{"null apiKeyHelper", `{"apiKeyHelper":null}`, false},
		{"clean profile", `{"model":"sonnet","remoteControlAtStartup":true}`, false},
		{"empty env", `{"env":{}}`, false},
		{"missing file", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "p.json")
			if c.json != "" {
				if err := os.WriteFile(path, []byte(c.json), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if got := perfilSinRC(path); got != c.want {
				t.Errorf("perfilSinRC(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestRCState(t *testing.T) {
	cases := map[string]string{
		"ok": "rc_connected", "fail": "rc_failed",
		"timeout": "rc_pending", "dead": "rc_pending", "pending": "rc_pending", "": "rc_pending",
	}
	for in, want := range cases {
		if got := rcState(in); got != want {
			t.Errorf("rcState(%q) = %q, want %q", in, got, want)
		}
	}
}

func newTestHost(t *testing.T) (*Host, string, string) {
	t.Helper()
	base := t.TempDir()
	profiles := filepath.Join(base, "perfiles")
	settings := filepath.Join(base, "settings.json")
	conv := filepath.Join(base, "conv")
	for _, d := range []string{profiles, conv} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return New(Options{
		ProfilesPath: profiles,
		SettingsPath: settings,
		ConvPath:     conv,
		Home:         base,
	}), base, settings
}

func writeProfile(t *testing.T, h *Host, name, content string) {
	t.Helper()
	if err := os.WriteFile(h.profilesPath+"/"+name+".json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestApplyProfile(t *testing.T) {
	h, _, settings := newTestHost(t)

	t.Run("applies and preserves symlink", func(t *testing.T) {
		writeProfile(t, h, "estandar", `{"model":"sonnet"}`)
		// settings.json as a symlink (Syncthing shared), like the real setup.
		target := filepath.Join(t.TempDir(), "real-settings.json")
		if err := os.WriteFile(target, []byte(`{"old":true}`), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, settings); err != nil {
			t.Fatal(err)
		}
		if err := h.applyProfile("estandar"); err != nil {
			t.Fatalf("applyProfile: %v", err)
		}
		got, _ := os.ReadFile(target)
		if string(got) != `{"model":"sonnet"}` {
			t.Errorf("target = %s", got)
		}
		if fi, err := os.Lstat(settings); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("settings.json no longer a symlink")
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		writeProfile(t, h, "roto", `{"model":`)
		if err := h.applyProfile("roto"); err == nil || !strings.Contains(err.Error(), "not valid JSON") {
			t.Errorf("expected JSON error, got %v", err)
		}
	})

	t.Run("missing profile is 404", func(t *testing.T) {
		err := h.applyProfile("nohay")
		if err == nil {
			t.Fatal("expected error")
		}
		var he *Error
		if !errors.As(err, &he) || he.Status != 404 {
			t.Errorf("expected 404 status, got %v", err)
		}
	})
}

func TestActiveProfileName(t *testing.T) {
	h, _, _ := newTestHost(t)
	writeProfile(t, h, "deepseek", `{"apiKeyHelper":"/x","env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com"}}`)
	writeProfile(t, h, "estandar", `{"model":"sonnet","remoteControlAtStartup":true}`)

	if got := h.activeProfileName(); got != "" {
		t.Errorf("no settings yet: got %q, want empty", got)
	}
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}
	if got := h.activeProfileName(); got != "deepseek" {
		t.Errorf("got %q, want deepseek", got)
	}
}

func TestHostNewDefaults(t *testing.T) {
	h := New(Options{Home: "/home/test"})
	if h.profilesPath != "/home/test/claude-shared/claude-perfiles" {
		t.Errorf("profiles: %s", h.profilesPath)
	}
	if h.settingsPath != "/home/test/.claude/settings.json" {
		t.Errorf("settings: %s", h.settingsPath)
	}
	if h.convPath != "/home/test/.claude/projects/-home-admin" {
		t.Errorf("conv: %s", h.convPath)
	}
	if h.claudeBinary != "/home/test/.local/bin/claude" {
		t.Errorf("claude: %s", h.claudeBinary)
	}
	if h.tmuxBinary != "/usr/bin/tmux" || h.bashBinary != "/usr/bin/bash" {
		t.Errorf("binaries: tmux=%s bash=%s", h.tmuxBinary, h.bashBinary)
	}
	if h.systemdRunBinary != "/usr/bin/systemd-run" {
		t.Errorf("systemdRun: %s", h.systemdRunBinary)
	}
	if h.rcBootstrap != "estandar" {
		t.Errorf("rcBootstrap: %s", h.rcBootstrap)
	}
}

func TestHostNewCustomPaths(t *testing.T) {
	h := New(Options{
		Home:         "/home/test",
		ProfilesPath: "/x/profiles",
		SettingsPath: "/x/settings.json",
		ConvPath:     "/x/conv",
		ClaudeBinary: "/x/claude",
		TmuxBinary:   "/x/tmux",
		BashBinary:   "/x/bash",
	})
	if h.profilesPath != "/x/profiles" || h.claudeBinary != "/x/claude" {
		t.Errorf("custom paths not honored: %+v", h)
	}
}

// errStatus extracts the *Error status from an error, or -1 if it isn't one.
func errStatus(err error) int {
	var he *Error
	if errors.As(err, &he) {
		return he.Status
	}
	return -1
}

func TestExecValidation(t *testing.T) {
	h, _, _ := newTestHost(t)
	cases := []struct {
		cmd    string
		args   map[string]string
		status int
		msg    string
	}{
		{"nonesuch", nil, 400, "unknown command"},
		{"tmux-kill", map[string]string{"name": "a;rm -rf"}, 400, "invalid session name"},
		{"tmux-kill", map[string]string{}, 400, "invalid session name"},
		{"claude-nueva", map[string]string{"profile": "../evil"}, 400, "invalid profile name"},
		{"claude-resume", map[string]string{"id": "notauuid"}, 400, "invalid conversation id"},
		{"claude-resume", map[string]string{}, 400, "invalid conversation id"},
		{"claude-perfil", map[string]string{"profile": "b/../x"}, 400, "invalid profile name"},
		{"conversation-get", map[string]string{"id": "zz"}, 400, "invalid conversation id"},
	}
	for _, c := range cases {
		_, err := h.Exec(c.cmd, c.args)
		if status := errStatus(err); status != c.status {
			t.Errorf("%s %v: status=%d want %d (err=%v)", c.cmd, c.args, status, c.status, err)
		}
		if err != nil && !strings.Contains(err.Error(), c.msg) {
			t.Errorf("%s: msg %q not in %q", c.cmd, c.msg, err.Error())
		}
	}
}

func TestExecClaudeNuevaProfileNotFound(t *testing.T) {
	h, _, _ := newTestHost(t)
	_, err := h.Exec("claude-nueva", map[string]string{"profile": "nohay"})
	if status := errStatus(err); status != 404 {
		t.Errorf("status=%d want 404 (err=%v)", status, err)
	}
}

func TestExecClaudePerfilNotFound(t *testing.T) {
	h, _, _ := newTestHost(t)
	_, err := h.Exec("claude-perfil", map[string]string{"profile": "nohay"})
	if status := errStatus(err); status != 404 {
		t.Errorf("status=%d want 404 (err=%v)", status, err)
	}
}

func TestExecClaudeResumeNotFound(t *testing.T) {
	h, _, _ := newTestHost(t)
	id := "00000000-0000-0000-0000-000000000001"
	_, err := h.Exec("claude-resume", map[string]string{"id": id})
	if status := errStatus(err); status != 404 {
		t.Errorf("status=%d want 404 (err=%v)", status, err)
	}
}

func TestExecConversationGetNotFound(t *testing.T) {
	h, _, _ := newTestHost(t)
	id := "00000000-0000-0000-0000-000000000001"
	_, err := h.Exec("conversation-get", map[string]string{"id": id})
	if status := errStatus(err); status != 404 {
		t.Errorf("status=%d want 404 (err=%v)", status, err)
	}
}

func TestExecProfilesListEmpty(t *testing.T) {
	h, _, _ := newTestHost(t)
	data, err := h.Exec("profiles-ls", nil)
	if err != nil {
		t.Fatalf("profiles-ls: %v", err)
	}
	if len(data.([]map[string]any)) != 0 {
		t.Errorf("expected empty profiles, got %v", data)
	}
}

func TestExecConversationsListMissingDir(t *testing.T) {
	h, _, _ := newTestHost(t)
	os.RemoveAll(h.convPath)
	_, err := h.Exec("conversations-ls", nil)
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestExecProfilesListMissingDir(t *testing.T) {
	h, _, _ := newTestHost(t)
	os.RemoveAll(h.profilesPath)
	_, err := h.Exec("profiles-ls", nil)
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestExecTmuxListNoTmux(t *testing.T) {
	// A Host with a bogus tmux binary: list-sessions fails, which tmux treats
	// as "no sessions" → empty list, not an error.
	h := New(Options{Home: t.TempDir(), TmuxBinary: "/nonexistent/tmux"})
	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatalf("tmux-ls: %v", err)
	}
	if len(data.([]map[string]any)) != 0 {
		t.Errorf("expected empty sessions, got %v", data)
	}
}

func TestExecTmuxKillNoTmux(t *testing.T) {
	h := New(Options{Home: t.TempDir(), TmuxBinary: "/nonexistent/tmux"})
	_, err := h.Exec("tmux-kill", map[string]string{"name": "x"})
	if err == nil {
		t.Fatal("expected error killing with bogus tmux")
	}
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestExecNewSessionNoTmux(t *testing.T) {
	// newSession launches via systemd-run --user --scope now (see newSession's
	// comment): a bogus TmuxBinary alone wouldn't be reached, since it's an
	// argument to systemd-run, not the exec'd binary. A bogus SystemdRunBinary
	// keeps the failure fast and deterministic without touching the real
	// systemd-run (which would need a user bus that may not exist in CI).
	h := New(Options{Home: t.TempDir(), TmuxBinary: "/nonexistent/tmux", SystemdRunBinary: "/nonexistent/systemd-run"})
	_, err := h.Exec("claude-nueva", nil)
	if err == nil {
		t.Fatal("expected error creating session with bogus tmux")
	}
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestConversationSummaryAndGet(t *testing.T) {
	h, _, _ := newTestHost(t)
	id := "00000000-0000-0000-0000-000000000002"
	lines := []string{
		`{"type":"user","isMeta":true,"message":{"content":"meta"}}`,
		`{"type":"user","cwd":"/home/admin/x","message":{"content":"Hola\nmundo"}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"respuesta"}]}}`,
		`{"type":"user","cwd":"/home/admin/x","message":{"content":null}}`,
	}
	path := h.convPath + "/" + id + ".jsonl"
	os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)

	text, cwd, ok := conversationSummary(path)
	if !ok || text != "Hola\nmundo" || cwd != "/home/admin/x" {
		t.Errorf("summary: %q %q %v", text, cwd, ok)
	}

	data, err := h.Exec("conversation-get", map[string]string{"id": id})
	if err != nil {
		t.Fatalf("conversation-get: %v", err)
	}
	conv := data.(map[string]any)
	if conv["origin"] != "pi" {
		t.Errorf("origin: %v", conv["origin"])
	}
	msgs := conv["messages"].([]map[string]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0]["role"] != "user" || msgs[1]["role"] != "assistant" {
		t.Errorf("roles: %v %v", msgs[0]["role"], msgs[1]["role"])
	}
	if !strings.Contains(msgs[0]["content"].(string), "Hola mundo") {
		t.Errorf("content flattened: %v", msgs[0]["content"])
	}
}

func TestSafeTitle(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"Hola mundo", true},
		{"testing.123", true},
		{"", false},
		{"a", true},
		{strings.Repeat("x", 80), true},
		{strings.Repeat("x", 81), false},
		{"hello\nworld", false},
	}
	for _, tt := range tests {
		if got := safeTitle(tt.in); got != tt.want {
			t.Errorf("safeTitle(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestExecValidationExtras(t *testing.T) {
	h, _, _ := newTestHost(t)
	cases := []struct {
		cmd    string
		args   map[string]string
		status int
		msg    string
	}{
		{"tmux-rename", map[string]string{"name": "x", "new_name": "!!"}, 400, "invalid session name"},
		{"tmux-rename", map[string]string{"name": "", "new_name": "y"}, 400, "invalid session name"},
		{"claude-rename", map[string]string{"session": "", "title": "x"}, 400, "invalid session name"},
		{"claude-rename", map[string]string{"session": "x", "title": ""}, 400, "invalid title"},
		{"claude-rename", map[string]string{"session": "x", "title": "\n"}, 400, "invalid title"},
		{"claude-nueva", map[string]string{"name": "a;rm"}, 400, "invalid session name"},
		{"claude-nueva", map[string]string{"name": strings.Repeat("x", 33)}, 400, "invalid session name"},
	}
	for _, c := range cases {
		_, err := h.Exec(c.cmd, c.args)
		if status := errStatus(err); status != c.status {
			t.Errorf("%s %v: status=%d want %d (err=%v)", c.cmd, c.args, status, c.status, err)
		}
		if err != nil && !strings.Contains(err.Error(), c.msg) {
			t.Errorf("%s: msg %q not in %q", c.cmd, c.msg, err.Error())
		}
	}
}

func TestExecTmuxRenameNoTmux(t *testing.T) {
	h := New(Options{Home: t.TempDir(), TmuxBinary: "/nonexistent/tmux"})
	_, err := h.Exec("tmux-rename", map[string]string{"name": "x", "new_name": "y"})
	if err == nil {
		t.Fatal("expected error")
	}
	if status := errStatus(err); status != 404 {
		t.Errorf("status=%d want 404 (err=%v)", status, err)
	}
}

func TestExecClaudeRenameNoTmux(t *testing.T) {
	h := New(Options{Home: t.TempDir(), TmuxBinary: "/nonexistent/tmux"})
	_, err := h.Exec("claude-rename", map[string]string{"session": "x", "title": "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	if status := errStatus(err); status != 404 {
		t.Errorf("status=%d want 404 (err=%v)", status, err)
	}
}

func TestExecNewSessionNamedNoTmux(t *testing.T) {
	h := New(Options{Home: t.TempDir(), TmuxBinary: "/nonexistent/tmux"})
	_, err := h.Exec("claude-nueva", map[string]string{"name": "mysession"})
	if err == nil {
		t.Fatal("expected error")
	}
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestConversationsListPaginationAndSearch(t *testing.T) {
	h, _, _ := newTestHost(t)
	for i := 0; i < 5; i++ {
		id := "00000000-0000-0000-0000-00000000000" + string(rune('0'+i))
		path := h.convPath + "/" + id + ".jsonl"
		content := `{"type":"user","cwd":"/home/admin/x","message":{"content":"proyecto sonarr"}}`
		if i%2 == 1 {
			content = `{"type":"user","cwd":"/home/luis/x","message":{"content":"bug parser"}}`
		}
		os.WriteFile(path, []byte(content+"\n"), 0600)
	}

	data, err := h.Exec("conversations-ls", map[string]string{"page": "1", "per_page": "2"})
	if err != nil {
		t.Fatalf("conversations-ls: %v", err)
	}
	list := data.([]map[string]any)
	if len(list) != 2 {
		t.Fatalf("expected 2 on page 1, got %d", len(list))
	}

	data, err = h.Exec("conversations-ls", map[string]string{"q": "parser"})
	if err != nil {
		t.Fatal(err)
	}
	list = data.([]map[string]any)
	if len(list) != 2 {
		t.Errorf("expected 2 matching 'parser', got %d", len(list))
	}
	for _, c := range list {
		if c["origin"] != "pc" {
			t.Errorf("origin for parser: %v", c["origin"])
		}
	}
}

func TestConversationsListFilters(t *testing.T) {
	h, _, _ := newTestHost(t)
	// Three conversations: pi/pc origins and distinct mtimes (-2d, -1d, now).
	ids := []string{
		"10000000-0000-0000-0000-000000000001",
		"10000000-0000-0000-0000-000000000002",
		"10000000-0000-0000-0000-000000000003",
	}
	content := []string{
		`{"type":"user","cwd":"/home/admin/x","message":{"content":"sonarr"}}`,
		`{"type":"user","cwd":"/home/luis/x","message":{"content":"parser"}}`,
		`{"type":"user","cwd":"/home/admin/y","message":{"content":"kodi"}}`,
	}
	now := time.Now()
	for i, id := range ids {
		path := h.convPath + "/" + id + ".jsonl"
		if err := os.WriteFile(path, []byte(content[i]+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		os.Chtimes(path, now, now.Add(time.Duration(i-2)*24*time.Hour))
	}

	list := func(args map[string]string) []map[string]any {
		t.Helper()
		data, err := h.Exec("conversations-ls", args)
		if err != nil {
			t.Fatalf("conversations-ls %v: %v", args, err)
		}
		return data.([]map[string]any)
	}

	if got := list(map[string]string{"origin": "pi"}); len(got) != 2 {
		t.Errorf("origin=pi: expected 2, got %d", len(got))
	}
	if got := list(map[string]string{"origin": "pc"}); len(got) != 1 {
		t.Errorf("origin=pc: expected 1, got %d", len(got))
	}
	if got := list(map[string]string{"origin": "?"}); len(got) != 0 {
		t.Errorf("origin=? (no such cwd): expected 0, got %d", len(got))
	}

	// from=today keeps only the "now" conversation; to=yesterday keeps the two older.
	today := now.Format("02/01/2006")
	yesterday := now.Add(-24 * time.Hour).Format("02/01/2006")
	if got := list(map[string]string{"from": today}); len(got) != 1 {
		t.Errorf("from=%s: expected 1, got %d", today, len(got))
	}
	if got := list(map[string]string{"to": yesterday}); len(got) != 2 {
		t.Errorf("to=%s: expected 2, got %d", yesterday, len(got))
	}
	if got := list(map[string]string{"from": today, "to": yesterday}); len(got) != 0 {
		t.Errorf("from after to: expected 0, got %d", len(got))
	}
}

func TestConversationExport(t *testing.T) {
	h, _, _ := newTestHost(t)
	id := "20000000-0000-0000-0000-000000000001"
	path := h.convPath + "/" + id + ".jsonl"
	content := `{"type":"user","message":{"content":"hola"}}
{"type":"assistant","message":{"content":"respuesta"}}
{"type":"user","isMeta":true,"message":{"content":"meta"}}
{"type":"tool_result","message":{"content":"ignored"}}
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("conversation-export", map[string]string{"id": id, "format": "jsonl"})
	if err != nil {
		t.Fatalf("export jsonl: %v", err)
	}
	exp := data.(map[string]any)
	if exp["filename"] != id+".jsonl" {
		t.Errorf("jsonl filename: %v", exp["filename"])
	}
	if exp["content"] != content {
		t.Errorf("jsonl content should be raw file, got %q", exp["content"])
	}

	data, err = h.Exec("conversation-export", map[string]string{"id": id, "format": "txt"})
	if err != nil {
		t.Fatalf("export txt: %v", err)
	}
	exp = data.(map[string]any)
	if exp["filename"] != id+".txt" {
		t.Errorf("txt filename: %v", exp["filename"])
	}
	body := exp["content"].(string)
	if !strings.Contains(body, "[user] hola") || !strings.Contains(body, "[assistant] respuesta") {
		t.Errorf("txt export missing messages: %q", body)
	}
	if strings.Contains(body, "meta") || strings.Contains(body, "ignored") {
		t.Errorf("txt export should skip meta/tool lines: %q", body)
	}

	if _, err := h.Exec("conversation-export", map[string]string{"id": "20000000-0000-0000-0000-000000000099", "format": "jsonl"}); err == nil {
		t.Error("expected error for missing conversation")
	}
}

func TestConversationMeta(t *testing.T) {
	h, _, _ := newTestHost(t)
	id := "20000000-0000-0000-0000-000000000002"
	os.WriteFile(h.convPath+"/"+id+".jsonl", []byte("{}"), 0600)

	// Defaults when no sidecar exists yet.
	data, err := h.Exec("conversation-meta-get", map[string]string{"id": id})
	if err != nil {
		t.Fatalf("meta-get: %v", err)
	}
	got := data.(map[string]any)
	if got["tags"] == nil || got["pinned"] != false || got["archived"] != false {
		t.Errorf("defaults wrong: %+v", got)
	}

	// Round-trip set.
	_, err = h.Exec("conversation-meta-set", map[string]string{
		"id": id, "tags": "infra, kodi ,infra", "notes": "revisar en casa", "pinned": "true",
	})
	if err != nil {
		t.Fatalf("meta-set: %v", err)
	}
	data, _ = h.Exec("conversation-meta-get", map[string]string{"id": id})
	got = data.(map[string]any)
	if got["pinned"] != true {
		t.Errorf("pinned not persisted: %+v", got)
	}
	tags := got["tags"].([]string)
	if len(tags) != 2 || tags[0] != "infra" || tags[1] != "kodi" {
		t.Errorf("tags dedup/parse wrong: %v", tags)
	}
	if got["notes"] != "revisar en casa" {
		t.Errorf("notes: %v", got["notes"])
	}

	// Validation.
	if _, err := h.Exec("conversation-meta-set", map[string]string{"id": id, "tags": "bad tag!"}); err == nil {
		t.Error("expected error for invalid tag")
	}
	long := strings.Repeat("x", 501)
	if _, err := h.Exec("conversation-meta-set", map[string]string{"id": id, "notes": long}); err == nil {
		t.Error("expected error for too-long notes")
	}
	// Missing conversation.
	if _, err := h.Exec("conversation-meta-set", map[string]string{"id": "20000000-0000-0000-0000-000000000099"}); err == nil {
		t.Error("expected error for missing conversation")
	}
}

func TestConversationsListArchiveAndPin(t *testing.T) {
	h, _, _ := newTestHost(t)
	old := "30000000-0000-0000-0000-000000000001"
	recent := "30000000-0000-0000-0000-000000000002"
	archived := "30000000-0000-0000-0000-000000000003"
	now := time.Now()
	for i, id := range []string{old, recent, archived} {
		path := h.convPath + "/" + id + ".jsonl"
		os.WriteFile(path, []byte(`{"type":"user","cwd":"/home/admin/x","message":{"content":"proyecto"}}`+"\n"), 0600)
		os.Chtimes(path, now, now.Add(-time.Duration(i)*24*time.Hour))
	}
	// recent is pinned; archived is archived.
	h.Exec("conversation-meta-set", map[string]string{"id": recent, "pinned": "true"})
	h.Exec("conversation-meta-set", map[string]string{"id": archived, "archived": "true"})

	list := func(args map[string]string) []map[string]any {
		t.Helper()
		data, err := h.Exec("conversations-ls", args)
		if err != nil {
			t.Fatalf("conversations-ls %v: %v", args, err)
		}
		return data.([]map[string]any)
	}

	// Default hides archived; pinned floats to the top.
	got := list(map[string]string{})
	if len(got) != 2 {
		t.Fatalf("default should hide archived: got %d", len(got))
	}
	if got[0]["id"] != recent {
		t.Errorf("pinned should be first, got %v", got[0]["id"])
	}
	if got[0]["pinned"] != true {
		t.Errorf("entry missing pinned flag: %+v", got[0])
	}

	// archived=all includes everything.
	if got := list(map[string]string{"archived": "all"}); len(got) != 3 {
		t.Errorf("archived=all: expected 3, got %d", len(got))
	}
	// archived=only shows just the archived one.
	if got := list(map[string]string{"archived": "only"}); len(got) != 1 || got[0]["id"] != archived {
		t.Errorf("archived=only: %v", got)
	}
}

func TestConversationsListLiveBypassesArchive(t *testing.T) {
	h, _, _ := newTestHost(t)
	// Make aliveConversations treat the current process as "claude running" so
	// the recent-modification fallback marks files modified in the last minute
	// as alive.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	h.claudeBinary = exe

	live := "30000000-0000-0000-0000-000000000004"
	dead := "30000000-0000-0000-0000-000000000005"
	now := time.Now()
	for id, age := range map[string]time.Duration{live: 0, dead: 24 * time.Hour} {
		path := h.convPath + "/" + id + ".jsonl"
		if err := os.WriteFile(path, []byte(`{"type":"ai-title","aiTitle":"Prueba viva","sessionId":"`+id+`"}`+"\n"+`{"type":"user","cwd":"/home/admin/x","message":{"content":"x"}}`+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, now, now.Add(-age)); err != nil {
			t.Fatal(err)
		}
		if _, err := h.Exec("conversation-meta-set", map[string]string{"id": id, "archived": "true"}); err != nil {
			t.Fatalf("archive %s: %v", id, err)
		}
	}
	ids := func(rows []map[string]any) []string {
		var out []string
		for _, r := range rows {
			out = append(out, fmt.Sprintf("%v", r["id"]))
		}
		return out
	}
	list := func(args map[string]string) []map[string]any {
		t.Helper()
		data, err := h.Exec("conversations-ls", args)
		if err != nil {
			t.Fatalf("conversations-ls %v: %v", args, err)
		}
		return data.([]map[string]any)
	}
	got := list(map[string]string{})
	if len(got) != 1 || got[0]["id"] != live {
		t.Fatalf("a live conversation must not be hidden by archive: got %v", ids(got))
	}
	if got[0]["is_alive"] != true {
		t.Errorf("live conversation must be marked alive: %+v", got[0])
	}
	if got[0]["title"] != "Prueba viva" {
		t.Errorf("title must be read from the jsonl: %+v", got[0]["title"])
	}
	if l := list(map[string]string{"alive": "1"}); len(l) != 1 || l[0]["id"] != live {
		t.Errorf("alive=1 must include the live archived conversation: got %v", ids(l))
	}
}

func TestExecConversationGetTitleAndAlive(t *testing.T) {
	h, _, _ := newTestHost(t)
	// Make aliveConversations treat the current process as "claude running" so
	// the recent-modification fallback marks this file as alive.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	h.claudeBinary = exe

	id := "30000000-0000-0000-0000-000000000009"
	path := h.convPath + "/" + id + ".jsonl"
	content := `{"type":"user","cwd":"/home/admin/x","message":{"content":"primero"}}` + "\n" +
		`{"type":"ai-title","aiTitle":"Con título","sessionId":"` + id + `"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := h.Exec("conversation-get", map[string]string{"id": id, "lines": "10"})
	if err != nil {
		t.Fatalf("conversation-get: %v", err)
	}
	row := data.(map[string]any)
	if row["title"] != "Con título" {
		t.Errorf("conversation-get title: got %v", row["title"])
	}
	if row["is_alive"] != true {
		t.Errorf("conversation-get is_alive: got %v", row["is_alive"])
	}
}

func TestConversationTitle(t *testing.T) {
	dir := t.TempDir()
	// Custom title must win over the AI one.
	p1 := dir + "/a.jsonl"
	os.WriteFile(p1, []byte(
		`{"type":"user","cwd":"/home/admin/x","message":{"content":"hola"}}`+"\n"+
			`{"type":"ai-title","aiTitle":"Titulo IA","sessionId":"a"}`+"\n"+
			`{"type":"custom-title","customTitle":"Mi nombre","sessionId":"a"}`+"\n"), 0600)
	if got := conversationTitle(p1); got != "Mi nombre" {
		t.Errorf("custom title must win: got %q", got)
	}
	// Without custom title, the AI one is used.
	p2 := dir + "/b.jsonl"
	os.WriteFile(p2, []byte(`{"type":"ai-title","aiTitle":"Solo IA","sessionId":"b"}`+"\n"), 0600)
	if got := conversationTitle(p2); got != "Solo IA" {
		t.Errorf("ai title fallback: got %q", got)
	}
	// Missing file is empty.
	if got := conversationTitle(dir + "/no.jsonl"); got != "" {
		t.Errorf("missing file must be empty: got %q", got)
	}
}

func TestConversationsListFiltersAliveInvariant(t *testing.T) {
	h, _, _ := newTestHost(t)
	// alive depends on ps output and the recent-modification fallback, which
	// varies with the environment. Assert the invariant instead: everything
	// returned must be alive, and only the recently-modified file qualifies.
	recent := ""
	now := time.Now()
	for i, id := range []string{
		"10000000-0000-0000-0000-000000000001",
		"10000000-0000-0000-0000-000000000002",
		"10000000-0000-0000-0000-000000000003",
	} {
		cwd := "/home/admin/x"
		if i == 1 {
			cwd = "/home/luis/x"
		}
		if i == 2 {
			recent = id
		}
		path := h.convPath + "/" + id + ".jsonl"
		os.WriteFile(path, []byte(`{"type":"user","cwd":"`+cwd+`","message":{"content":"x"}}`+"\n"), 0600)
		os.Chtimes(path, now, now.Add(time.Duration(i-2)*24*time.Hour))
	}
	list := func(args map[string]string) []map[string]any {
		t.Helper()
		data, err := h.Exec("conversations-ls", args)
		if err != nil {
			t.Fatalf("conversations-ls %v: %v", args, err)
		}
		return data.([]map[string]any)
	}
	got := list(map[string]string{"alive": "1"})
	if len(got) > 1 {
		t.Errorf("alive=1: at most the recent conversation can qualify, got %d", len(got))
	}
	for _, c := range got {
		if c["is_alive"] != true {
			t.Errorf("alive filter returned non-alive conversation: %v", c["id"])
		}
		if c["id"] != recent {
			t.Errorf("alive filter returned unexpected id %v (only the recent one can be alive)", c["id"])
		}
	}
}

func TestMetrics(t *testing.T) {
	h, base, _ := newTestHost(t)
	now := time.Now()

	// Audit: two session_new today (one with profile=estandar, one with profile=kodi)
	// and one session_new yesterday.
	auditPath := filepath.Join(base, "audit.jsonl")
	auditLines := []string{
		fmt.Sprintf(`{"time":"%s","action":"session_new","user":"luis","detail":"session=x, profile=estandar"}`, now.Format(time.RFC3339)),
		fmt.Sprintf(`{"time":"%s","action":"session_new","user":"luis","detail":"session=y, profile=kodi"}`, now.Format(time.RFC3339)),
		fmt.Sprintf(`{"time":"%s","action":"session_new","user":"luis","detail":"session=z, profile=estandar"}`, now.AddDate(0, 0, -1).Format(time.RFC3339)),
		`{"time":"2020-01-01T00:00:00Z","action":"login","user":"luis","detail":""}`,
	}
	if err := os.WriteFile(auditPath, []byte(strings.Join(auditLines, "\n")), 0600); err != nil {
		t.Fatal(err)
	}

	// Conversation file with usage data today.
	convID := "40000000-0000-0000-0000-000000000001"
	convLines := []string{
		`{"type":"user","timestamp":"` + now.Format(time.RFC3339) + `","message":{"content":"hi"}}`,
		`{"type":"assistant","timestamp":"` + now.Format(time.RFC3339) + `","message":{"model":"claude-sonnet-4-5","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":25,"cache_creation_input_tokens":10}}}`,
		`{"type":"assistant","timestamp":"2020-01-01T00:00:00Z","message":{"model":"claude-haiku-4-5","usage":{"input_tokens":900,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`, // outside window
	}
	if err := os.WriteFile(h.convPath+"/"+convID+".jsonl", []byte(strings.Join(convLines, "\n")), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("metrics", map[string]string{"audit_path": auditPath})
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	m := data.(map[string]any)

	// Sessions per day: today should be 2.
	sessions := m["sessions_per_day"].([]map[string]any)
	if len(sessions) != metricsDays {
		t.Fatalf("expected %d days, got %d", metricsDays, len(sessions))
	}
	todayKey := now.Format("02/01/2006")
	yesterdayKey := now.AddDate(0, 0, -1).Format("02/01/2006")
	if sessions[metricsDays-1]["date"] != todayKey || sessions[metricsDays-1]["count"].(int) != 2 {
		t.Errorf("today entry = %+v, want date %s count 2", sessions[metricsDays-1], todayKey)
	}
	if sessions[metricsDays-2]["date"] != yesterdayKey || sessions[metricsDays-2]["count"].(int) != 1 {
		t.Errorf("yesterday entry = %+v, want date %s count 1", sessions[metricsDays-2], yesterdayKey)
	}

	// Top profiles: estandar=2 first, kodi=1 second.
	profiles := m["top_profiles"].([]map[string]any)
	if len(profiles) != 2 || profiles[0]["name"] != "estandar" || profiles[1]["name"] != "kodi" {
		t.Errorf("top_profiles = %+v, want [estandar kodi]", profiles)
	}

	// Token usage: today input=100, output=50, cache=35; yesterday all zero.
	tokens := m["token_usage_per_day"].([]map[string]any)
	if tokens[metricsDays-1]["input"].(int) != 100 {
		t.Errorf("today input = %d, want 100", tokens[metricsDays-1]["input"].(int))
	}
	if tokens[metricsDays-1]["output"].(int) != 50 {
		t.Errorf("today output = %d, want 50", tokens[metricsDays-1]["output"].(int))
	}
	if tokens[metricsDays-1]["cache"].(int) != 35 {
		t.Errorf("today cache = %d, want 35", tokens[metricsDays-1]["cache"].(int))
	}

	// Per-model: only the in-window sonnet line counts; the haiku line is 2020.
	byModel := m["token_usage_by_model"].([]map[string]any)
	if len(byModel) != 1 || byModel[0]["model"] != "claude-sonnet-4-5" {
		t.Fatalf("token_usage_by_model = %+v, want [claude-sonnet-4-5]", byModel)
	}
	if byModel[0]["input"].(int) != 100 || byModel[0]["output"].(int) != 50 || byModel[0]["cache"].(int) != 35 || byModel[0]["messages"].(int) != 1 {
		t.Errorf("byModel[0] = %+v, want input 100 output 50 cache 35 messages 1", byModel[0])
	}
}

func TestMetricsModelFilter(t *testing.T) {
	h, _, _ := newTestHost(t)
	now := time.Now()
	convID := "41000000-0000-0000-0000-000000000001"
	convLines := []string{
		`{"type":"assistant","timestamp":"` + now.Format(time.RFC3339) + `","message":{"model":"claude-sonnet-4-5","usage":{"input_tokens":100,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
		`{"type":"assistant","timestamp":"` + now.Format(time.RFC3339) + `","message":{"model":"claude-haiku-4-5","usage":{"input_tokens":300,"output_tokens":10,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
	}
	if err := os.WriteFile(h.convPath+"/"+convID+".jsonl", []byte(strings.Join(convLines, "\n")), 0600); err != nil {
		t.Fatal(err)
	}

	// Without filter: both models counted.
	data, err := h.Exec("metrics", map[string]string{"audit_path": "/nonexistent/audit.jsonl"})
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	m := data.(map[string]any)
	byModel := m["token_usage_by_model"].([]map[string]any)
	if len(byModel) != 2 {
		t.Fatalf("token_usage_by_model = %+v, want 2 models", byModel)
	}
	if byModel[0]["model"] != "claude-haiku-4-5" || byModel[0]["input"].(int) != 300 {
		t.Errorf("byModel[0] = %+v, want haiku input 300 first (sorted by input desc)", byModel[0])
	}

	// With model filter: only that model in per-day and per-model.
	data, err = h.Exec("metrics", map[string]string{"audit_path": "/nonexistent/audit.jsonl", "model": "claude-sonnet-4-5"})
	if err != nil {
		t.Fatalf("metrics filtered: %v", err)
	}
	m = data.(map[string]any)
	byModel = m["token_usage_by_model"].([]map[string]any)
	if len(byModel) != 1 || byModel[0]["model"] != "claude-sonnet-4-5" {
		t.Fatalf("filtered token_usage_by_model = %+v, want [claude-sonnet-4-5]", byModel)
	}
	tokens := m["token_usage_per_day"].([]map[string]any)
	if tokens[metricsDays-1]["input"].(int) != 100 {
		t.Errorf("filtered today input = %d, want 100", tokens[metricsDays-1]["input"].(int))
	}
}

func TestMetricsNoAudit(t *testing.T) {
	h, _, _ := newTestHost(t)
	data, err := h.Exec("metrics", map[string]string{"audit_path": "/nonexistent/audit.jsonl"})
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	m := data.(map[string]any)
	sessions := m["sessions_per_day"].([]map[string]any)
	if len(sessions) != metricsDays {
		t.Errorf("expected %d days, got %d", metricsDays, len(sessions))
	}
	for _, d := range sessions {
		if d["count"].(int) != 0 {
			t.Errorf("count should be 0 with no audit file: %+v", d)
		}
	}
}

func TestSessionChatCapsToRecent(t *testing.T) {
	h, _, _ := newTestHost(t)
	name := fmt.Sprintf("ccsmtest%d", time.Now().UnixNano())
	if err := exec.Command("tmux", "new-session", "-d", "-s", name).Run(); err != nil {
		t.Skipf("tmux unavailable: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", name).Run()

	// No --session-id in argv -> lifetimeConv: the most recent transcript in
	// the conversations directory (isolated per TempDir) is the one chosen.
	id := "00000000-0000-0000-0000-0000000000aa"
	var b strings.Builder
	total := maxChatMsgs + 50
	for i := 0; i < total; i++ {
		fmt.Fprintf(&b, `{"type":"user","message":{"content":"msg %d"}}`+"\n", i)
	}
	path := h.convPath + "/" + id + ".jsonl"
	if err := os.WriteFile(path, []byte(b.String()), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("session-chat", map[string]string{"name": name})
	if err != nil {
		t.Fatalf("session-chat: %v", err)
	}
	msgs := data.(map[string]any)["messages"].([]map[string]any)
	if len(msgs) != maxChatMsgs {
		t.Fatalf("expected %d recent messages, got %d", maxChatMsgs, len(msgs))
	}
	// The newest message (the last one written) must be present.
	if got := msgs[len(msgs)-1]["content"].(string); got != fmt.Sprintf("msg %d", total-1) {
		t.Errorf("newest message missing: got %q", got)
	}
	// The oldest one (msg 0) was dropped.
	if got := msgs[0]["content"].(string); got == "msg 0" {
		t.Errorf("oldest message should have been dropped, got %q", got)
	}
}

func TestChatRoleAndText(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		wantRole   string
		wantText   string
		wantSource string
		wantOK     bool
	}{
		{
			name:       "user message",
			line:       `{"type":"user","message":{"content":"hola"}}`,
			wantRole:   "user",
			wantText:   "hola",
			wantSource: "user",
			wantOK:     true,
		},
		{
			name:       "user with explicit null isMeta shown",
			line:       `{"type":"user","isMeta":null,"message":{"content":"ok"}}`,
			wantRole:   "user",
			wantText:   "ok",
			wantSource: "user",
			wantOK:     true,
		},
		{
			name:       "assistant message",
			line:       `{"type":"assistant","message":{"content":"hola mundo"}}`,
			wantRole:   "assistant",
			wantText:   "hola mundo",
			wantSource: "assistant",
			wantOK:     true,
		},
		{
			name:   "user meta skipped",
			line:   `{"type":"user","isMeta":true,"message":{"content":"ignored"}}`,
			wantOK: false,
		},
		{
			name:   "tool_result skipped",
			line:   `{"type":"user","message":{"content":[{"type":"tool_result","content":"ls"}]}}`,
			wantOK: false,
		},
		{
			name:       "queue-operation enqueue is the user's mid-turn message",
			line:       `{"type":"queue-operation","operation":"enqueue","content":"sigo pendiente"}`,
			wantRole:   "user",
			wantText:   "sigo pendiente",
			wantSource: "enqueue",
			wantOK:     true,
		},
		{
			name:   "queue-operation remove skipped (drain, not a message)",
			line:   `{"type":"queue-operation","operation":"remove","content":"sigo pendiente"}`,
			wantOK: false,
		},
		{
			name:   "queue-operation control command skipped",
			line:   `{"type":"queue-operation","operation":"enqueue","content":"/remote-control"}`,
			wantOK: false,
		},
		{
			name:   "queue-operation empty skipped",
			line:   `{"type":"queue-operation","operation":"enqueue","content":"  "}`,
			wantOK: false,
		},
		{
			name:   "unrelated type skipped",
			line:   `{"type":"attachment","content":"x"}`,
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var line convLine
			if err := json.Unmarshal([]byte(c.line), &line); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			role, text, source, ok := chatRoleAndText(line)
			if ok != c.wantOK {
				t.Fatalf("ok=%v, want %v", ok, c.wantOK)
			}
			if ok {
				if role != c.wantRole {
					t.Errorf("role=%q, want %q", role, c.wantRole)
				}
				if text != c.wantText {
					t.Errorf("text=%q, want %q", text, c.wantText)
				}
				if source != c.wantSource {
					t.Errorf("source=%q, want %q", source, c.wantSource)
				}
			}
		})
	}
}

func TestChatDedup(t *testing.T) {
	cases := []struct {
		name  string
		lines []string // transcript lines in order
		want  []string // rendered roles+text: "user:Sigue", "assistant:ok", ...
	}{
		{
			name: "drained mid-turn message shows once (enqueue + dequeue + user)",
			lines: []string{
				`{"type":"queue-operation","operation":"enqueue","content":"Sigue"}`,
				`{"type":"queue-operation","operation":"dequeue"}`,
				`{"type":"user","message":{"content":"Sigue"},"promptSource":"queued"}`,
			},
			want: []string{"user:Sigue"},
		},
		{
			name: "queued_command mid-turn message keeps its enqueue (no user line)",
			lines: []string{
				`{"type":"queue-operation","operation":"enqueue","content":"Avanza"}`,
				`{"type":"queue-operation","operation":"remove","content":"Avanza"}`,
				`{"type":"assistant","message":{"content":"hecho"}}`,
			},
			want: []string{"user:Avanza", "assistant:hecho"},
		},
		{
			name: "two drained messages each show once",
			lines: []string{
				`{"type":"queue-operation","operation":"enqueue","content":"A"}`,
				`{"type":"queue-operation","operation":"enqueue","content":"B"}`,
				`{"type":"user","message":{"content":"A"}}`,
				`{"type":"user","message":{"content":"B"}}`,
			},
			want: []string{"user:A", "user:B"},
		},
		{
			name: "repeated text sent later while idle is a real second message",
			lines: []string{
				`{"type":"user","message":{"content":"Sigue"}}`,
				`{"type":"assistant","message":{"content":"ok"}}`,
				`{"type":"user","message":{"content":"Sigue"}}`,
			},
			want: []string{"user:Sigue", "assistant:ok", "user:Sigue"},
		},
		{
			name: "assistant streaming snapshots with same message.id collapse",
			lines: []string{
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"draft one"}]}}`,
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"draft one, fuller"}]}}`,
			},
			want: []string{"assistant:draft one, fuller"},
		},
		{
			name: "assistant split blocks with same message.id keep the text line",
			lines: []string{
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"thinking","thinking":"..."}]}}`,
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"the reply"}]}}`,
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"tool_use","id":"t1"}]}}`,
			},
			want: []string{"assistant:the reply"},
		},
		{
			name: "distinct assistant messages stay separate",
			lines: []string{
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"first"}]}}`,
				`{"type":"assistant","message":{"id":"m2","content":[{"type":"text","text":"second"}]}}`,
			},
			want: []string{"assistant:first", "assistant:second"},
		},
		{
			name: "mixed: enqueue drains and reply snapshot collapses",
			lines: []string{
				`{"type":"queue-operation","operation":"enqueue","content":"Haz commit"}`,
				`{"type":"queue-operation","operation":"dequeue"}`,
				`{"type":"user","message":{"content":"Haz commit"}}`,
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"v1"}]}}`,
				`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"v2 completo"}]}}`,
			},
			want: []string{"user:Haz commit", "assistant:v2 completo"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var dd chatDedup
			for _, raw := range c.lines {
				var l convLine
				if err := json.Unmarshal([]byte(raw), &l); err != nil {
					t.Fatalf("unmarshal %s: %v", raw, err)
				}
				role, text, source, ok := chatRoleAndText(l)
				if ok {
					dd.add(role, text, source, l.Message.ID)
				}
			}
			dd.flushPending()
			got := make([]string, 0, len(dd.turns))
			for _, tr := range dd.turns {
				got = append(got, tr.role+":"+tr.text)
			}
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("turn %d = %q, want %q (all: %v)", i, got[i], c.want[i], got)
				}
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"model feedback bold/dim", // real /model output with bold and dim
			"\x1b[1mdeepseek-v4-flash\x1b[22m and saved for new sessions\x1b[2m\x1b[22m\n",
			"deepseek-v4-flash and saved for new sessions\n",
		},
		{
			"clean text untouched",
			"hola\nmundo",
			"hola\nmundo",
		},
		{
			"cursor hide + clear + home",
			"\x1b[?25l\x1b[2J\x1b[Hpronto",
			"pronto",
		},
		{
			"OSC title terminated by BEL",
			"\x1b]0;ccsm\x07ready",
			"ready",
		},
		{
			"literal [1m] in text is NOT stripped", // the model id deepseek-v4-pro[1m] carries a literal [1m]
			"Set model to \x1b[1mdeepseek-v4-pro[1m]\x1b[22m ok",
			"Set model to deepseek-v4-pro[1m] ok",
		},
	}
	for _, c := range cases {
		if got := stripANSI(c.in); got != c.want {
			t.Errorf("%s: stripANSI(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestModeFromBadge(t *testing.T) {
	cases := []struct{ name, badge, want string }{
		{"manual", "  ⏸ manual mode on · ? for shortcuts · ← for agents", "manual"},
		{"accept-edits con espacio", "  ⏵⏵ accept edits on (shift+tab to cycle) · ← for agents", "accept-edits"},
		{"plan", "  ⏸ plan mode on (shift+tab to cycle) · ← for agents", "plan"},
		{"auto", "  ⏵⏵ auto mode on (shift+tab to cycle) · ← for agents", "auto"},
		{"bypassPermissions", "  ⏵⏵ bypassPermissions mode on (shift+tab to cycle) · ← for agents", "bypassPermissions"},
		{"approval dialog (no mode)", " Esc to cancel · Tab to amend · ctrl+e to explain", ""},
		{"empty line", "", ""},
	}
	for _, c := range cases {
		if got := modeFromBadge(c.badge); got != c.want {
			t.Errorf("%s: modeFromBadge(%q) = %q, want %q", c.name, c.badge, got, c.want)
		}
	}
}

func TestModeDistance(t *testing.T) {
	// Distances verified empirically on 2.1.227 from the plan anchor:
	// plan→auto 1, plan→manual 2, plan→accept-edits 3.
	cases := []struct {
		from, to string
		want     int
	}{
		{"plan", "auto", 1},
		{"plan", "manual", 2},
		{"plan", "accept-edits", 3},
		{"manual", "accept-edits", 1},
		{"accept-edits", "manual", 3},
		{"auto", "auto", 0},
		{"auto", "plan", 3},
	}
	for _, c := range cases {
		n, ok := modeDistance(c.from, c.to)
		if !ok {
			t.Errorf("modeDistance(%s, %s): ok=false, want true", c.from, c.to)
			continue
		}
		if n != c.want {
			t.Errorf("modeDistance(%s, %s) = %d, want %d", c.from, c.to, n, c.want)
		}
	}
	if _, ok := modeDistance("", "plan"); ok {
		t.Error("modeDistance('', plan): ok=true, want false")
	}
	if _, ok := modeDistance("manual", "nope"); ok {
		t.Error("modeDistance(manual, nope): ok=true, want false")
	}
}

func TestPaneWaitingReason(t *testing.T) {
	approval := " Verify [1m] applied\n This command requires approval\n Do you want to proceed?\n ❯ 1. Yes\n Esc to cancel · Tab to amend · ctrl+e to explain"
	if got := paneWaitingReason(approval); got != "approval" {
		t.Errorf("paneWaitingReason(dialog) = %q, want approval", got)
	}
	// File-edit approval dialog (Claude Code 2.1.228 wording).
	edit := " Edit file\n claude.sh\n Do you want to make this edit to claude.sh?\n ❯ 1. Yes\n   2. Yes, allow all edits during this session (shift+tab)\n   3. No\n Esc to cancel · Tab to amend"
	if got := paneWaitingReason(edit); got != "approval" {
		t.Errorf("paneWaitingReason(edit dialog) = %q, want approval", got)
	}
	// Choice dialog (AskUserQuestion): option selection, not yes/no approval.
	choice := "☐ Comentario\n¿Qué comentario quieres dejar en claude.sh?\n❯ 1. Mantener el actual\n  2. Describe acción exacta\nEnter to select · ↑/↓ to navigate · n to add notes · Esc to cancel"
	if got := paneWaitingReason(choice); got != "choice" {
		t.Errorf("paneWaitingReason(choice dialog) = %q, want choice", got)
	}
	if got := paneWaitingReason("  Esc to cancel · Tab to amend"); got != "approval" {
		t.Errorf("paneWaitingReason(choice footer) = %q, want approval", got)
	}
	if got := paneWaitingReason("  ⏵⏵ auto mode on (shift+tab to cycle) · ← for agents"); got != "" {
		t.Errorf("paneWaitingReason(footer normal) = %q, want empty", got)
	}
	// Tool-permission dialogs whose wording isn't hardcoded anywhere (Fetch,
	// and a made-up future tool) must still be detected generically via the
	// cursor picker, not by matching each tool's question text.
	fetch := " Fetch\n\n   url: \"https://example.com\",\n   Claude wants to fetch content from example.com\n\n" +
		" Do you want to allow Claude to fetch this content?\n" +
		" ❯ 1. Yes\n   2. Yes, and don't ask again for example.com\n   3. No, and tell Claude what to do differently (esc)"
	if got := paneWaitingReason(fetch); got != "approval" {
		t.Errorf("paneWaitingReason(fetch dialog) = %q, want approval", got)
	}
	futureTool := " Do you want to let Claude use SomeFutureTool?\n ❯ 1. Yes\n   2. No"
	if got := paneWaitingReason(futureTool); got != "approval" {
		t.Errorf("paneWaitingReason(generic future-tool dialog) = %q, want approval", got)
	}
	// Auto-mode environment setup wizard: a checkbox form, not a Y/N approval.
	// Must NOT be classified as "approval" (a blind Enter would toggle the
	// focused checkbox instead of dismissing it).
	setup := " Set up auto mode for your environment?\n\n" +
		" How you use Claude here    ◄ Mixed ►\n" +
		" › Also scan shell history   [✓]\n" +
		"   Also scan your other repos [ ]\n\n" +
		"   Continue\n\n" +
		" ←/→ to change usage · Enter to continue · Esc to cancel"
	if got := paneWaitingReason(setup); got != "setup" {
		t.Errorf("paneWaitingReason(setup wizard) = %q, want setup", got)
	}
}

// TestPaneWaitingDetailID proves paneWaitingDetail's id — the identity the
// turn watcher needs to tell one dialog from a different one of the same
// reason (see watcher.go) — is populated for every waiting reason, not just
// "choice". Approval and plan-approval dialogs render through the identical
// numbered "❯ N. label" picker AskUserQuestion does, so the same paneChoice
// parse that extracts a choice's question line works unchanged for them;
// "setup" has no such line (a checkbox wizard, not a numbered list), so it
// falls back to the reason string itself.
func TestPaneWaitingDetailID(t *testing.T) {
	approval := " Verify [1m] applied\n This command requires approval\n Do you want to proceed?\n ❯ 1. Yes\n Esc to cancel · Tab to amend · ctrl+e to explain"
	if _, _, id := parsePaneWaitingDetail(approval); id != "Do you want to proceed?" {
		t.Errorf("approval id = %q, want the question line", id)
	}

	edit := " Edit file\n claude.sh\n Do you want to make this edit to claude.sh?\n ❯ 1. Yes\n   2. Yes, allow all edits during this session (shift+tab)\n   3. No\n Esc to cancel · Tab to amend"
	if _, _, id := parsePaneWaitingDetail(edit); id != "Do you want to make this edit to claude.sh?" {
		t.Errorf("edit-approval id = %q, want the question line", id)
	}

	choice := "☐ Comentario\n¿Qué comentario quieres dejar en claude.sh?\n❯ 1. Mantener el actual\n  2. Describe acción exacta\nEnter to select · ↑/↓ to navigate · n to add notes · Esc to cancel"
	reason, c, id := parsePaneWaitingDetail(choice)
	if reason != "choice" || id != "¿Qué comentario quieres dejar en claude.sh?" {
		t.Errorf("choice reason/id = %q/%q", reason, id)
	}
	if c == nil {
		t.Error("choice map should still be populated for reason=choice")
	}

	setup := " Set up auto mode for your environment?\n\n" +
		" How you use Claude here    ◄ Mixed ►\n" +
		" › Also scan shell history   [✓]\n" +
		"   Also scan your other repos [ ]\n\n" +
		"   Continue\n\n" +
		" ←/→ to change usage · Enter to continue · Esc to cancel"
	if reason, _, id := parsePaneWaitingDetail(setup); reason != "setup" || id != "setup" {
		t.Errorf("setup reason/id = %q/%q, want setup/setup", reason, id)
	}

	if reason, _, id := parsePaneWaitingDetail("  ⏵⏵ auto mode on (shift+tab to cycle) · ← for agents"); reason != "" || id != "" {
		t.Errorf("idle pane reason/id = %q/%q, want empty/empty", reason, id)
	}
}

func TestPaneChoice(t *testing.T) {
	// Real AskUserQuestion picker shape (Claude Code 2.1.228): title line,
	// question, numbered options with the current one prefixed by ❯, footer.
	pane := " ☐ Comentario\n" +
		"¿Qué comentario quieres dejar en claude.sh?\n" +
		"❯ 1. Mantener el actual\n" +
		"  2. Describe acción exacta\n" +
		"  3. Explica el propósito\n" +
		"  4. Uso operativo\n" +
		"Enter to select · ↑/↓ to navigate · n to add notes · Esc to cancel"
	q, opts, sel, ok := paneChoice(pane)
	if !ok {
		t.Fatal("paneChoice: ok=false, want true")
	}
	if q != "¿Qué comentario quieres dejar en claude.sh?" {
		t.Errorf("question = %q", q)
	}
	want := []string{"Mantener el actual", "Describe acción exacta", "Explica el propósito", "Uso operativo"}
	if len(opts) != len(want) {
		t.Fatalf("options = %v, want %v", opts, want)
	}
	for i := range want {
		if opts[i] != want[i] {
			t.Errorf("option[%d] = %q, want %q", i, opts[i], want[i])
		}
	}
	if sel != 0 {
		t.Errorf("selected = %d, want 0 (❯ on option 1)", sel)
	}

	// Cursor elsewhere and no trailing panel noise.
	pane2 := "¿Cuál?\n  1. uno\n❯ 2. dos\n  3. tres\nEnter to select"
	_, opts2, sel2, ok := paneChoice(pane2)
	if !ok || len(opts2) != 3 || opts2[1] != "dos" || sel2 != 1 {
		t.Errorf("paneChoice2 = %v sel=%d ok=%v", opts2, sel2, ok)
	}

	// No numbered options → not ok.
	if _, _, _, ok := paneChoice("just text\nno options"); ok {
		t.Error("paneChoice without options: ok=true, want false")
	}
}

func TestConvFilesSubfolders(t *testing.T) {
	h, _, _ := newTestHost(t)
	flatID := "11111111-0000-0000-0000-000000000001"
	projID := "22222222-0000-0000-0000-000000000002"

	// One transcript directly in h.convPath and another in a project subfolder
	// (the session cwd slug).
	if err := os.WriteFile(h.convPath+"/"+flatID+".jsonl", []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(h.convProjectsDir(), "-home-admin-projects-demo")
	if err := os.MkdirAll(proj, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proj+"/"+projID+".jsonl", []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// convFiles lists both exactly once (no h.convPath duplication).
	files, err := h.convFiles()
	if err != nil {
		t.Fatalf("convFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("convFiles() len = %d, want 2 (sin duplicados): %+v", len(files), files)
	}
	got := map[string]string{}
	for _, f := range files {
		got[f.id] = f.path
	}
	if got[flatID] != h.convPath+"/"+flatID+".jsonl" {
		t.Errorf("convFiles() flat = %q, want %q", got[flatID], h.convPath+"/"+flatID+".jsonl")
	}
	if got[projID] != proj+"/"+projID+".jsonl" {
		t.Errorf("convFiles() proj = %q, want %q", got[projID], proj+"/"+projID+".jsonl")
	}

	// convFileFor resolves the subfolder when it applies.
	if p := h.convFileFor(projID); p != proj+"/"+projID+".jsonl" {
		t.Errorf("convFileFor(projID) = %q, want %q", p, proj+"/"+projID+".jsonl")
	}
	if p := h.convFileFor(flatID); p != h.convPath+"/"+flatID+".jsonl" {
		t.Errorf("convFileFor(flatID) = %q, want %q", p, h.convPath+"/"+flatID+".jsonl")
	}
}

func TestCwdSlug(t *testing.T) {
	cases := map[string]string{
		"/home/admin":               "-home-admin",
		"/home/admin/projects/ccsm": "-home-admin-projects-ccsm",
		"":                          "",
		"/":                         "",
	}
	for in, want := range cases {
		if got := cwdSlug(in); got != want {
			t.Errorf("cwdSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConvFilesInRestrictsByCwd(t *testing.T) {
	h, _, _ := newTestHost(t)
	other := filepath.Join(h.convProjectsDir(), "-home-admin-otro-proyecto")
	proj := filepath.Join(h.convProjectsDir(), "-home-admin-projects-demo")
	for _, d := range []string{other, proj} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeJSONL := func(p string) {
		if err := os.WriteFile(p, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeJSONL(filepath.Join(h.convPath, "11111111-0000-0000-0000-000000000001.jsonl"))
	writeJSONL(filepath.Join(other, "22222222-0000-0000-0000-000000000002.jsonl"))
	writeJSONL(filepath.Join(proj, "33333333-0000-0000-0000-000000000003.jsonl"))

	// The cwd-matched folder wins; no global mixing.
	files := h.convFilesIn("/home/admin/projects/demo")
	if len(files) != 1 || files[0].id != "33333333-0000-0000-0000-000000000003" {
		t.Fatalf("convFilesIn(proj) = %+v, want only the demo transcript", files)
	}
	// Unknown cwd -> full scan fallback (flat + all subfolders).
	files = h.convFilesIn("/ruta/desconocida")
	if len(files) != 3 {
		t.Fatalf("convFilesIn(unknown) = %d files, want 3 (global fallback)", len(files))
	}
}

func TestFileBirthTime(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.jsonl")
	if err := os.WriteFile(p, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if bt := fileBirthTime(p); bt <= 0 {
		t.Errorf("fileBirthTime(fresh) = %d, want > 0", bt)
	}
	if bt := fileBirthTime(filepath.Join(t.TempDir(), "nope.jsonl")); bt != 0 {
		t.Errorf("fileBirthTime(missing) = %d, want 0", bt)
	}
}

// TestSessionPaneColor: the colour flag is opt-in. Without it the behaviour is
// the historical one (ANSI stripped, no -e); with it the SGR codes survive and
// capture-pane is asked for them with -e — both are required, since capture-pane
// omits escape sequences unless -e is given.
func TestSessionPaneColor(t *testing.T) {
	const raw = "\x1b[31mred\x1b[0m plain"
	argsFile := filepath.Join(t.TempDir(), "capture-args.txt")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_HIST":         raw,
		"FAKE_TMUX_CAPTURE_ARGS": argsFile,
	})

	t.Run("plain: ANSI stripped, no -e", func(t *testing.T) {
		out, err := h.sessionPane("3", false)
		if err != nil {
			t.Fatal(err)
		}
		if out["content"] != "red plain" {
			t.Errorf("content = %q, want ANSI stripped", out["content"])
		}
		last := lastCaptureArgs(t, argsFile)
		if strings.Contains(last, " -e") {
			t.Errorf("plain capture-pane passed -e: %q", last)
		}
	})

	t.Run("color: ANSI preserved and -e passed", func(t *testing.T) {
		out, err := h.sessionPane("3", true)
		if err != nil {
			t.Fatal(err)
		}
		if out["content"] != raw {
			t.Errorf("content = %q, want the raw ANSI %q", out["content"], raw)
		}
		last := lastCaptureArgs(t, argsFile)
		if !strings.Contains(last, " -e") {
			t.Errorf("colour capture-pane did not pass -e: %q", last)
		}
	})
}

// lastCaptureArgs returns the last recorded capture-pane invocation.
func lastCaptureArgs(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read capture args: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines[len(lines)-1]
}

// TestExecSessionPaneColorArg: the colour flag reaches sessionPane through the
// generic args map, so it works in both deployment modes without any change to
// the agent protocol.
func TestExecSessionPaneColorArg(t *testing.T) {
	const raw = "\x1b[32mgreen\x1b[0m"
	h := fakeHost(t, map[string]string{"FAKE_TMUX_HIST": raw})

	data, err := h.Exec("session-pane", map[string]string{"name": "3", "color": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := data.(map[string]string)["content"]; got != raw {
		t.Errorf("color=1 content = %q, want raw ANSI", got)
	}

	data, err = h.Exec("session-pane", map[string]string{"name": "3"})
	if err != nil {
		t.Fatal(err)
	}
	if got := data.(map[string]string)["content"]; got != "green" {
		t.Errorf("default content = %q, want stripped", got)
	}
}

// TestSessionSendCtrlO: ctrl-o is whitelisted so a client can trigger Claude
// Code's own collapse/expand of verbose output in the pane.
func TestSessionSendCtrlO(t *testing.T) {
	sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_SENDKEYS": sendkeys})

	if _, err := h.sessionSend("3", "", "ctrl-o"); err != nil {
		t.Fatalf("sessionSend ctrl-o: %v", err)
	}
	data, _ := os.ReadFile(sendkeys)
	if !strings.Contains(string(data), "C-o") {
		t.Errorf("C-o not sent to the pane: %q", string(data))
	}
	if _, err := h.sessionSend("3", "", "ctrl-x"); err == nil {
		t.Error("expected an unknown key to still be rejected")
	}
}

// TestSessionSendEnterIsLiteral: "enter" must reach the pane as the raw byte
// \r (send-keys -l), not tmux's named "Enter" key — reported in production as
// needing two presses to dismiss an approval/choice dialog. Same class of fix
// as rawShiftTab (\x1b[Z): the named key doesn't always reach Claude's input
// loop, only the literal byte reliably does.
func TestSessionSendEnterIsLiteral(t *testing.T) {
	sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_SENDKEYS": sendkeys})

	if _, err := h.sessionSend("3", "", "enter"); err != nil {
		t.Fatalf("sessionSend enter: %v", err)
	}
	data, _ := os.ReadFile(sendkeys)
	got := string(data) // don't trim: the literal \r itself is the thing under test
	if !strings.Contains(got, "-l") || !strings.Contains(got, "\r") {
		t.Errorf("enter not sent as a literal \\r byte: %q", got)
	}
	if strings.Contains(got, "Enter") {
		t.Errorf("enter must not be sent as the named tmux key: %q", got)
	}
}

// TestSessionSendTextEnterIsLiteral: a plain chat message shares the exact
// same submit mechanics as the "enter" special key above (sessionSend's text
// branch calls sendKeys, which types the literal text then presses Enter) —
// so it must go through the same raw \r, not tmux's named "Enter" key. Before
// this, sendKeys had its own separate, unfixed named-key Enter, so a typed
// chat answer to an approval/choice dialog (e.g. typing "1" instead of
// clicking the option) could sit unsubmitted exactly like the bug
// TestSessionSendEnterIsLiteral covers.
func TestSessionSendTextEnterIsLiteral(t *testing.T) {
	sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_SENDKEYS": sendkeys})

	if _, err := h.sessionSend("3", "1", ""); err != nil {
		t.Fatalf("sessionSend text: %v", err)
	}
	data, _ := os.ReadFile(sendkeys)
	got := string(data)
	if !strings.Contains(got, "-l") || !strings.Contains(got, "\r") {
		t.Errorf("trailing enter not sent as a literal \\r byte: %q", got)
	}
	if strings.Contains(got, "Enter") {
		t.Errorf("trailing enter must not be sent as the named tmux key: %q", got)
	}
}

// TestForEachLineSurvivesOversizedLine is the regression for the frozen-grid
// bug: a transcript line holding a pasted image (base64 in the message
// content) measured 1.6MB, and bufio.Scanner with a 1MB ceiling aborts the
// whole walk on the first such line (ErrTooLong), silently dropping every line
// after it — the tile's history froze at the image. forEachLine must keep
// walking past it and the parsers built on it must still see later messages.
func TestForEachLineSurvivesOversizedLine(t *testing.T) {
	huge := "{" + `"type":"user","message":{"role":"user","content":[{"type":"image","source":{"data":"` + strings.Repeat("A", 2*1024*1024) + `"}}]}` + "}"
	var got []string
	if err := forEachLine(strings.NewReader(
		`{"type":"user","message":{"role":"user","content":"antes"}}`+"\n"+
			huge+"\n"+
			`{"type":"user","message":{"role":"user","content":"despues"}}`),
		func(line []byte) bool { got = append(got, string(line)); return true }); err != nil {
		t.Fatalf("forEachLine: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d lines, want 3 (walk aborted on the oversized line?)", len(got))
	}
	if !strings.Contains(got[2], "despues") {
		t.Errorf("line after the oversized one was dropped: %q", got[2])
	}
}

// TestConversationReadsSurviveOversizedLine: conversationSummary and
// conversationTitle also read transcripts line by line; they must not abort
// when an image line exceeds the old 1MB scanner ceiling either.
func TestConversationReadsSurviveOversizedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	big := "{" + `"type":"user","message":{"role":"user","content":[{"type":"image","source":{"data":"` + strings.Repeat("A", 2*1024*1024) + `"}}]}` + "}"
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"primer mensaje"}}`,
		big,
		`{"type":"ai-title","aiTitle":"Titulo Nuevo"}`,
		`{"type":"user","message":{"role":"user","content":"segundo mensaje"}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	summary, _, ok := conversationSummary(path)
	if !ok || summary != "primer mensaje" {
		t.Errorf("conversationSummary = %q ok=%v, want primer mensaje", summary, ok)
	}
	if title := conversationTitle(path); title != "Titulo Nuevo" {
		t.Errorf("conversationTitle = %q, want Titulo Nuevo (image line aborted the tail scan?)", title)
	}
}
