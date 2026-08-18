package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeTmux is an env-driven stub of the tmux commands the Host runs. It lets
// tests exercise tmuxList, rcStatus, sessionAlive, tmuxKill and newSession
// without a real tmux.
const fakeTmux = `#!/bin/sh
case "$1" in
  list-sessions)
    [ -n "$FAKE_TMUX_NO_SESSIONS" ] && exit 1
    printf '%s' "$FAKE_TMUX_LIST"
    exit 0 ;;
  capture-pane)
    [ -n "$FAKE_TMUX_CAPTURE_ARGS" ] && printf '%s\n' "$*" >> "$FAKE_TMUX_CAPTURE_ARGS"
    [ -n "$FAKE_TMUX_CAPTURE_FAIL" ] && exit 1
    for a in "$@"; do
      [ "$a" = "-S" ] && { printf '%s' "$FAKE_TMUX_HIST"; exit 0; }
    done
    # Stateful mode wheel (discoverModeWheel tests): capture-pane returns the
    # footer for the mode whose index is stored in FAKE_TMUX_MODE_STATE. The
    # footer is idle (… ← for agents) so paneWorking reads false and no dialog
    # tokens so paneWaitingReason reads "".
    if [ -n "$FAKE_TMUX_MODE_WHEEL" ] && [ -n "$FAKE_TMUX_MODE_STATE" ]; then
      idx="$(cat "$FAKE_TMUX_MODE_STATE" 2>/dev/null)"
      [ -z "$idx" ] && idx=0
      # FAKE_TMUX_MODE_HIDE_UNTIL simulates an unreadable badge (blind-cycle
      # tests): while idx is below it, print a footer with no mode token.
      if [ -n "$FAKE_TMUX_MODE_HIDE_UNTIL" ] && [ "$idx" -lt "$FAKE_TMUX_MODE_HIDE_UNTIL" ]; then
        printf '  ⏵⏵ mode on (shift+tab to cycle) · esc to interrupt · ← for agents\n'
        exit 0
      fi
      mode=""; n=0; IFS=,
      for m in $FAKE_TMUX_MODE_WHEEL; do
        [ "$n" = "$idx" ] && { mode="$m"; break; }
        n=$((n + 1))
      done
      IFS=' '
      [ -z "$mode" ] && exit 1
      case "$mode" in
        accept-edits) badge="accept edits" ;;
        *) badge="$mode" ;;
      esac
      if [ "$mode" = "manual" ]; then
        # manual idle footer: "? for shortcuts" is paneWorking's manual-idle
        # token, so the probe's idle check passes instead of reading it as
        # "manual mode … esc to interrupt" (the manual WORKING footer).
        printf '  ⏵⏵ manual mode on (shift+tab to cycle) · ? for shortcuts · ← for agents\n'
      else
        printf '  ⏵⏵ %s mode on (shift+tab to cycle) · esc to interrupt · ← for agents\n' "$badge"
      fi
      exit 0
    fi
    # ensurePaneReady tests: capture-pane returns empty (not ready) for the
    # first FAKE_TMUX_READY_AFTER-1 calls, then a valid idle badge from then
    # on — simulating a session whose TUI takes a few reads to render its
    # first frame.
    if [ -n "$FAKE_TMUX_READY_AFTER" ] && [ -n "$FAKE_TMUX_READY_COUNTER" ]; then
      n="$(cat "$FAKE_TMUX_READY_COUNTER" 2>/dev/null)"
      [ -z "$n" ] && n=0
      n=$((n + 1))
      printf '%s' "$n" > "$FAKE_TMUX_READY_COUNTER"
      if [ "$n" -ge "$FAKE_TMUX_READY_AFTER" ]; then
        printf '  ⏵⏵ auto mode on (shift+tab to cycle) · esc to interrupt · ← for agents\n'
      fi
      exit 0
    fi
    printf '%s' "$FAKE_TMUX_LINE"
    exit 0 ;;
  has-session)
    [ "$3" = "=$FAKE_TMUX_DEAD" ] && exit 1
    # Opt-in stateful liveness: a name recorded in FAKE_TMUX_KILLS by an
    # earlier kill-session call reports dead, like a real kill would. Static
    # FAKE_TMUX_DEAD above still covers tests that want a name dead from the
    # start, without a kill in between.
    if [ -n "$FAKE_TMUX_KILL_MARKS_DEAD" ] && [ -n "$FAKE_TMUX_KILLS" ] && [ -f "$FAKE_TMUX_KILLS" ] && grep -qF "$3" "$FAKE_TMUX_KILLS"; then
      exit 1
    fi
    exit 0 ;;
  kill-session)
    [ -n "$FAKE_TMUX_KILL_FAIL" ] && { echo "no such session" >&2; exit 1; }
    echo "$3" >> "$FAKE_TMUX_KILLS"
    exit 0 ;;
  set-option)
    [ -n "$FAKE_TMUX_OPTS" ] && printf '%s\n' "$5" >> "$FAKE_TMUX_OPTS"
    exit 0 ;;
  new-session)
    [ -n "$FAKE_TMUX_NEW_FAIL" ] && { echo "create failed" >&2; exit 1; }
    [ -n "$FAKE_TMUX_NEW_ARGS" ] && printf '%s\n' "$*" >> "$FAKE_TMUX_NEW_ARGS"
    # Mirrors real tmux: an explicit -s <name> wins and is echoed back via
    # -P -F '#S'; only an unnamed request falls back to FAKE_TMUX_NEW_NAME
    # (simulating tmux's own auto-numbering).
    name="" prev=""
    for a in "$@"; do
      [ "$prev" = "-s" ] && { name="$a"; break; }
      prev="$a"
    done
    [ -z "$name" ] && name="${FAKE_TMUX_NEW_NAME:-3}"
    printf '%s\n' "$name"
    exit 0 ;;
  list-panes)
    [ -n "$FAKE_TMUX_LIST_PANES_FAIL" ] && exit 1
    printf '%s\t%s\t%s\n' "${FAKE_TMUX_PANE_SESSION:-x}" "${FAKE_TMUX_PANE_ID:-%0}" "$FAKE_TMUX_PANE_PID"
    exit 0 ;;
  display-message)
    [ -n "$FAKE_TMUX_CREATED" ] || exit 1
    printf '%s' "$FAKE_TMUX_CREATED"
    exit 0 ;;
  load-buffer)
    # sendText's paste path feeds the message on stdin. Always drain it, even
    # when the test is not capturing it, or the writing side sees EPIPE.
    if [ -n "$FAKE_TMUX_BUFFER" ]; then cat > "$FAKE_TMUX_BUFFER"; else cat > /dev/null; fi
    [ -n "$FAKE_TMUX_LOAD_FAIL" ] && exit 1
    exit 0 ;;
  paste-buffer)
    [ -n "$FAKE_TMUX_PASTE" ] && printf '%s\n' "$*" >> "$FAKE_TMUX_PASTE"
    [ -n "$FAKE_TMUX_PASTE_FAIL" ] && exit 1
    exit 0 ;;
  delete-buffer)
    [ -n "$FAKE_TMUX_DELETE" ] && printf '%s\n' "$*" >> "$FAKE_TMUX_DELETE"
    exit 0 ;;
  send-keys)
    [ -n "$FAKE_TMUX_SENDKEYS" ] && printf '%s\n' "$*" >> "$FAKE_TMUX_SENDKEYS"
    # Concurrency tests (TestSessionSendConcurrent): a delay between the
    # literal-text call and its own Enter widens the window for a second,
    # unlocked sessionSend to interleave its own literal in between —
    # exactly the race sessionSendLock exists to close.
    [ "$4" = "-l" ] && [ -n "$FAKE_TMUX_SENDKEYS_DELAY" ] && sleep "$FAKE_TMUX_SENDKEYS_DELAY"
    # A raw shift-tab (\e[Z, what rawShiftTab sends) advances the mode wheel.
    if [ -n "$FAKE_TMUX_MODE_WHEEL" ] && [ -n "$FAKE_TMUX_MODE_STATE" ]; then
      last=""
      for a in "$@"; do last="$a"; done
      if [ "$last" = "$(printf '\033')[Z" ]; then
        idx="$(cat "$FAKE_TMUX_MODE_STATE" 2>/dev/null)"
        [ -z "$idx" ] && idx=0
        count=0; IFS=,
        for _m in $FAKE_TMUX_MODE_WHEEL; do count=$((count + 1)); done
        IFS=' '
        idx=$(( (idx + 1) % count ))
        printf '%s' "$idx" > "$FAKE_TMUX_MODE_STATE"
      fi
    fi
    exit 0 ;;
esac
exit 1
`

// fakeSystemdRun stubs systemd-run for newSession's launch: skip its leading
// flags (--user --scope --collect --quiet --slice=...) and exec the real
// command (tmuxBinary + args) that follows, same as the real systemd-run
// would run it as the scope's main process.
const fakeSystemdRun = `#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in
    -*) shift ;;
    *) break ;;
  esac
done
exec "$@"
`

// fakePs is a stub of `ps -eo comm,args` driven by FAKE_PS_OUT / FAKE_PS_FAIL.
const fakePs = `#!/bin/sh
[ -n "$FAKE_PS_FAIL" ] && exit 1
printf '%s' "$FAKE_PS_OUT"
exit 0
`

// fakeHost builds a Host whose binaries are stub scripts in a temp dir, plus a
// profiles dir and settings path. env maps environment variables for the stubs.
func fakeHost(t *testing.T, env map[string]string) *Host {
	t.Helper()
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	os.MkdirAll(binDir, 0700)

	writeExe := func(name, script string) string {
		p := filepath.Join(binDir, name)
		if err := os.WriteFile(p, []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
		return p
	}
	tmux := writeExe("tmux", fakeTmux)
	systemdRun := writeExe("systemd-run", fakeSystemdRun)
	_ = writeExe("claude", "#!/bin/sh\nexit 0")
	_ = writeExe("bash", "#!/bin/sh\nexit 0")

	for k, v := range env {
		t.Setenv(k, v)
	}

	profiles := filepath.Join(base, "perfiles")
	os.MkdirAll(profiles, 0700)
	settings := filepath.Join(base, "settings.json")
	conv := filepath.Join(base, "conv")
	os.MkdirAll(conv, 0700)

	h := New(Options{
		ProfilesPath:     profiles,
		SettingsPath:     settings,
		ConvPath:         conv,
		ClaudeBinary:     filepath.Join(binDir, "claude"),
		TmuxBinary:       tmux,
		SystemdRunBinary: systemdRun,
		BashBinary:       filepath.Join(binDir, "bash"),
		RcBootstrap:      "estandar",
		RcWaitSeconds:    2,
		RcPollSeconds:    0,
		RcSettleSeconds:  0,
		Home:             base,
	})
	// ensurePaneReady's wait is a real-world timing concern (a session's own
	// TUI booting), not something the fake tmux simulates: its default
	// capture-pane is empty, which paneReady reads as "not ready" and would
	// otherwise stall every sessionSend/sessionMode test for paneReadyTimeout.
	// Tests targeting ensurePaneReady itself set this back explicitly.
	h.paneReadyTimeout = 0
	return h
}

func (h *Host) writeProfile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(h.profilesPath+"/"+name+".json", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func (h *Host) readSettings(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(h.settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	return string(b)
}

func TestTmuxListSortNoiseAndStatus(t *testing.T) {
	h := fakeHost(t, map[string]string{
		// name, created_string, pane_title, project, session_created, window_activity
		"FAKE_TMUX_LIST": "10\t2024-01-02 11:00:00\tclaude\t\t1700000000\t1700000500\n3\t2024-01-01 10:00:00\t✳ analizando logs\t\t1699990000\t1699990100\n",
		"FAKE_TMUX_LINE": "/rc connected",
	})

	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatalf("tmux-ls: %v", err)
	}
	sessions := data.([]map[string]any)
	if len(sessions) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(sessions))
	}
	// Sorted by last activity (most recent interaction first): 10 touched
	// after 3, so it leads despite the higher creation number.
	if sessions[0]["name"] != "10" || sessions[1]["name"] != "3" {
		t.Errorf("order: %v, %v", sessions[0]["name"], sessions[1]["name"])
	}
	if sessions[0]["last_activity"] != int64(1700000500) {
		t.Errorf("last_activity: %v", sessions[0]["last_activity"])
	}
	for _, s := range sessions {
		if s["name"] == "3" {
			if s["task"] != "analizando logs" {
				t.Errorf("noise not stripped: %q", s["task"])
			}
			if s["status"] != "rc_connected" {
				t.Errorf("status: %v", s["status"])
			}
		}
	}
}

// TestTmuxListSortByActivityFallback: sessions without a window_activity value
// fall back to session_created, and unknown activity keeps the creation-order
// tie-break (numeric, then alphabetical).
func TestTmuxListSortByActivityFallback(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			"activity over creation",
			"9\t2024-01-01 10:00:00\tt\t\t1700000000\t1700000900\n0\t2024-01-01 10:00:00\tt\t\t1700000000\t1700000100\n",
			[]string{"9", "0"},
		},
		{
			"creation fallback when no activity",
			"9\t2024-01-01 10:00:00\tt\t\t1700000100\t\n0\t2024-01-01 10:00:00\tt\t\t1700000000\t\n",
			[]string{"9", "0"},
		},
		{
			"numeric tie-break on equal activity",
			"10\t2024-01-01 10:00:00\tt\t\t1700000000\t1700000000\n3\t2024-01-01 10:00:00\tt\t\t1700000000\t1700000000\n",
			[]string{"3", "10"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := fakeHost(t, map[string]string{"FAKE_TMUX_LIST": c.in})
			data, err := h.Exec("tmux-ls", nil)
			if err != nil {
				t.Fatalf("tmux-ls: %v", err)
			}
			sessions := data.([]map[string]any)
			if len(sessions) != len(c.want) {
				t.Fatalf("want %d sessions, got %d: %v", len(c.want), len(sessions), sessions)
			}
			for i, name := range c.want {
				if sessions[i]["name"] != name {
					t.Fatalf("order: %v, want %v", sessions[i]["name"], c.want)
				}
			}
		})
	}
}

func TestTmuxListProject(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST": "3\t2024-01-01 10:00:00\tclaude\tprojects/ccsm\n4\t2024-01-01 10:00:00\tclaude\n",
	})
	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatalf("tmux-ls: %v", err)
	}
	sessions := data.([]map[string]any)
	if len(sessions) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(sessions))
	}
	if sessions[0]["project"] != "projects/ccsm" {
		t.Errorf("project: %q", sessions[0]["project"])
	}
	if p, ok := sessions[1]["project"]; ok && p != "" {
		t.Errorf("untagged session project: %q", p)
	}
}

func TestTmuxListHostnameTask(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST": "3\t2024-01-01 10:00:00\trb\n",
	})
	// hostname is resolved via os.Hostname() at construction, not $HOSTNAME
	// (systemd services don't set that env var) — set it directly, as New()
	// would have from the real syscall.
	h.hostname = "rb"
	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatal(err)
	}
	sessions := data.([]map[string]any)
	if sessions[0]["task"] != "(no task)" {
		t.Errorf("task: %q", sessions[0]["task"])
	}
}

// TestTmuxListHostnameIgnoresEnv guards against reintroducing the old
// os.Getenv("HOSTNAME") read: under ccsm-agent (a systemd service) that env
// var is unset, so the comparison never matched and the raw hostname leaked
// into the session list as its task (visto 2026-08-15). $HOSTNAME here is
// deliberately wrong to prove it plays no part.
func TestTmuxListHostnameIgnoresEnv(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST": "3\t2024-01-01 10:00:00\tpi\n",
	})
	t.Setenv("HOSTNAME", "not-pi")
	h.hostname = "pi"

	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatal(err)
	}
	sessions := data.([]map[string]any)
	if sessions[0]["task"] != "(no task)" {
		t.Errorf("task: %q, want hidden via h.hostname regardless of $HOSTNAME", sessions[0]["task"])
	}
}

// TestNewResolvesHostname confirms New() wires h.hostname from the real
// os.Hostname() syscall (not $HOSTNAME, which is what production actually
// runs on — the fake-tmux tests all override the field directly instead).
func TestNewResolvesHostname(t *testing.T) {
	want, err := os.Hostname()
	if err != nil {
		t.Skip("os.Hostname unavailable in this environment")
	}
	h := New(Options{Home: t.TempDir()})
	if h.hostname != want {
		t.Errorf("hostname = %q, want %q", h.hostname, want)
	}
}

func TestTmuxListNoSessions(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_NO_SESSIONS": "1"})
	data, err := h.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatalf("tmux-ls: %v", err)
	}
	if len(data.([]map[string]any)) != 0 {
		t.Errorf("expected empty, got %v", data)
	}
}

func TestRCStatusBranches(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"connected", map[string]string{"FAKE_TMUX_LINE": "/rc connected"}, "rc_connected"},
		{"failed", map[string]string{"FAKE_TMUX_LINE": "/rc failed"}, "rc_failed"},
		{"history", map[string]string{"FAKE_TMUX_LINE": "hello", "FAKE_TMUX_HIST": "… /remote-control is active"}, "rc_connected"},
		{"pending", map[string]string{"FAKE_TMUX_LINE": "hello"}, "rc_pending"},
		{"capture error", map[string]string{"FAKE_TMUX_CAPTURE_FAIL": "1"}, "rc_pending"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := fakeHost(t, c.env)
			if got := h.rcStatus("3"); got != c.want {
				t.Errorf("rcStatus = %q, want %q", got, c.want)
			}
		})
	}
}

func TestClaudeNewRCCleanProfile(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected"})
	h.writeProfile(t, "estandar", `{"model":"sonnet","remoteControlAtStartup":true}`)

	data, err := h.Exec("claude-nueva", map[string]string{"profile": "estandar"})
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	out := data.(map[string]string)
	if out["session"] != "3" || out["status"] != "rc_connected" {
		t.Errorf("out: %+v", out)
	}
	// RC-clean profile is applied per-session, settings.json stays untouched.
	if _, err := os.Stat(h.settingsPath); !os.IsNotExist(err) {
		t.Errorf("settings.json should not be written for RC-clean profiles")
	}
}

func TestClaudeNewSinRCStaging(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey"}`)

	data, err := h.Exec("claude-nueva", map[string]string{"profile": "deepseek"})
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	out := data.(map[string]string)
	if out["session"] != "3" || out["status"] != "rc_connected" {
		t.Errorf("out: %+v", out)
	}
	// Staging restores the target profile to settings.json.
	if got := h.readSettings(t); !strings.Contains(got, "apiKeyHelper") {
		t.Errorf("settings not restored to target: %s", got)
	}
}

func TestClaudeNewNoProfileUsesActive(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)
	// Make deepseek the active profile → claude-nueva without profile stages it.
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("claude-nueva", nil)
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	if out := data.(map[string]string); out["status"] != "rc_connected" {
		t.Errorf("out: %+v", out)
	}
}

func TestClaudeNewStagingDead(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected", "FAKE_TMUX_DEAD": "3"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)

	data, err := h.Exec("claude-nueva", map[string]string{"profile": "deepseek"})
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	if out := data.(map[string]string); out["status"] != "rc_pending" {
		t.Errorf("dead session maps to rc_pending, got %+v", out)
	}
}

func TestClaudeNewStagingFailed(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc failed"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)

	data, err := h.Exec("claude-nueva", map[string]string{"profile": "deepseek"})
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	if out := data.(map[string]string); out["status"] != "rc_failed" {
		t.Errorf("out: %+v", out)
	}
}

func TestClaudeNewStagingTimeout(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "idle"})
	h.rcWaitSeconds = 0 // deadline already past → timeout
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)

	data, err := h.Exec("claude-nueva", map[string]string{"profile": "deepseek"})
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	if out := data.(map[string]string); out["status"] != "rc_pending" {
		t.Errorf("timeout maps to rc_pending, got %+v", out)
	}
}

func TestClaudeNewStagingMissingBootstrap(t *testing.T) {
	h := fakeHost(t, nil)
	h.rcBootstrap = "nope"
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)

	data, err := h.Exec("claude-nueva", map[string]string{"profile": "deepseek"})
	if err != nil {
		t.Fatalf("claude-nueva sin perfil de staging no debe fallar: %v", err)
	}
	// Sin perfil de staging se asume que no habrá bridge, no un error.
	if out := data.(map[string]string); out["status"] != "rc_failed" {
		t.Errorf("sin perfil de staging se asume no-bridge, got %+v", out)
	}
}

func TestClaudeNewNewSessionFailure(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_NEW_FAIL": "1"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)

	_, err := h.Exec("claude-nueva", map[string]string{"profile": "estandar"})
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestClaudeResumeNoActiveProfile(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected"})
	id := "00000000-0000-0000-0000-0000000000aa"
	os.WriteFile(h.convPath+"/"+id+".jsonl", []byte(`{"type":"user","cwd":"/home/luis/x","message":{"content":"hola"}}`+"\n"), 0600)

	data, err := h.Exec("claude-resume", map[string]string{"id": id})
	if err != nil {
		t.Fatalf("claude-resume: %v", err)
	}
	out := data.(map[string]string)
	if out["session"] != "3" || out["status"] != "rc_connected" {
		t.Errorf("out: %+v", out)
	}
}

func TestClaudeResumeSinRCStaging(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_LINE": "/rc connected"})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}
	id := "00000000-0000-0000-0000-0000000000bb"
	os.WriteFile(h.convPath+"/"+id+".jsonl", []byte(`{"type":"user","message":{"content":"hola"}}`+"\n"), 0600)

	data, err := h.Exec("claude-resume", map[string]string{"id": id})
	if err != nil {
		t.Fatalf("claude-resume: %v", err)
	}
	if out := data.(map[string]string); out["status"] != "rc_connected" {
		t.Errorf("out: %+v", out)
	}
}

func TestTmuxKillSuccessAndFailure(t *testing.T) {
	kills := filepath.Join(t.TempDir(), "kills.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_KILLS": kills})

	if _, err := h.Exec("tmux-kill", map[string]string{"name": "3"}); err != nil {
		t.Fatalf("tmux-kill: %v", err)
	}
	got, _ := os.ReadFile(kills)
	if !strings.Contains(string(got), "=3") {
		t.Errorf("kills marker: %q", got)
	}

	h2 := fakeHost(t, map[string]string{"FAKE_TMUX_KILL_FAIL": "1"})
	_, err := h2.Exec("tmux-kill", map[string]string{"name": "3"})
	if status := errStatus(err); status != 500 {
		t.Errorf("status=%d want 500 (err=%v)", status, err)
	}
}

func TestSessionAlive(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_DEAD": "3"})
	if h.sessionAlive("3") {
		t.Error("session 3 should be dead")
	}
	if !h.sessionAlive("4") {
		t.Error("session 4 should be alive")
	}
}

func TestProfilesListSkipsJunkAndFlagsActive(t *testing.T) {
	h := fakeHost(t, nil)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)
	// Junk that must be skipped.
	os.WriteFile(h.profilesPath+"/notjson.txt", []byte("x"), 0600)
	os.MkdirAll(h.profilesPath+"/subdir", 0700)
	os.WriteFile(h.profilesPath+"/BAd name.json", []byte(`{}`), 0600)
	// Make deepseek the active profile.
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("profiles-ls", nil)
	if err != nil {
		t.Fatalf("profiles-ls: %v", err)
	}
	profiles := data.([]map[string]any)
	if len(profiles) != 2 {
		t.Fatalf("want 2 profiles, got %d: %v", len(profiles), profiles)
	}
	for _, p := range profiles {
		if p["name"] == "deepseek" && p["is_active"] != true {
			t.Errorf("deepseek should be active: %v", p)
		}
		if p["name"] == "estandar" && p["is_active"] != false {
			t.Errorf("estandar should be inactive: %v", p)
		}
	}
}

func TestProfilesListNoSettings(t *testing.T) {
	h := fakeHost(t, nil)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	data, err := h.Exec("profiles-ls", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p := data.([]map[string]any); len(p) != 1 || p[0]["is_active"] != false {
		t.Errorf("profiles: %v", p)
	}
}

// TestProfilesListOnlyOneActive guards the UI sync between the active-profile
// tick and settings.json: two catalog profiles with identical content must not
// both be flagged is_active (the old per-file equality check did exactly that),
// and the flagged one must be the one activeProfileName resolves.
func TestProfilesListOnlyOneActive(t *testing.T) {
	h := fakeHost(t, nil)
	// Two byte-identical profiles: whichever is listed first wins, never both.
	h.writeProfile(t, "clone", `{"model":"sonnet"}`)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	if err := h.applyProfile("estandar"); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("profiles-ls", nil)
	if err != nil {
		t.Fatalf("profiles-ls: %v", err)
	}
	profiles := data.([]map[string]any)
	if len(profiles) != 2 {
		t.Fatalf("want 2 profiles, got %d: %v", len(profiles), profiles)
	}
	activeCount := 0
	for _, p := range profiles {
		if p["is_active"] == true {
			activeCount++
			if p["name"] != h.activeProfileName() {
				t.Errorf("flagged profile %v disagrees with activeProfileName %q", p, h.activeProfileName())
			}
		}
	}
	if activeCount != 1 {
		t.Errorf("want exactly 1 active profile, got %d: %v", activeCount, profiles)
	}
}

// TestProfilesListUnmatchedSettings: when settings.json doesn't match any
// catalog profile (hand-edited, or applied outside the catalog), no profile is
// flagged active — the tick must reflect settings.json's content truthfully.
func TestProfilesListUnmatchedSettings(t *testing.T) {
	h := fakeHost(t, nil)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x"}`)
	// A settings.json that is neither profile (e.g. an extra hand-added key).
	if err := os.WriteFile(h.settingsPath, []byte(`{"model":"sonnet","tui":"fullscreen"}`), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("profiles-ls", nil)
	if err != nil {
		t.Fatalf("profiles-ls: %v", err)
	}
	for _, p := range data.([]map[string]any) {
		if p["is_active"] == true {
			t.Errorf("no profile should be active when settings.json matches none: %v", p)
		}
	}
	if got := h.activeProfileName(); got != "" {
		t.Errorf("activeProfileName should be empty, got %q", got)
	}
}

func TestActiveProfileNameNoMatch(t *testing.T) {
	h := fakeHost(t, nil)
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	os.WriteFile(h.settingsPath, []byte(`{"env":{"ANTHROPIC_BASE_URL":"https://x.com"}}`), 0600)
	if got := h.activeProfileName(); got != "" {
		t.Errorf("no match expected, got %q", got)
	}

	// profiles dir gone → "".
	os.RemoveAll(h.profilesPath)
	if got := h.activeProfileName(); got != "" {
		t.Errorf("missing profiles dir: got %q", got)
	}
}

// TestSessionStatusWorkingUsesStatusWord: the turn watcher's "working" signal
// comes from the generation status word line ("✻ cooking… (4m · ↓ tokens)"),
// not from the footer-hint heuristic of paneWorking. A status word present
// means working; an idle footer — even the real one that hugs the 80-col edge
// and ends in a wrapped "/rc" fragment — means not. The footer heuristic read
// only the last non-blank line and, when the long working footer wrapped, got
// a bare fragment back, so it reported idle mid-generation and the watcher
// announced turn_complete as soon as the assistant message id appeared.
func TestSessionStatusWorkingUsesStatusWord(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{"status word present → working", "  ✻ Razzle-dazzling… (4m 39s · ↓ 23.9k tokens)", true},
		// The verb is matched by shape, not language: an accented status word
		// from a localized UI still reads as working.
		{"accented status word → working", "  ✻ Préparant… (2m · ↓ 5.1k tokens)", true},
		{"idle footer (wrapped /rc tail) → not working", "  ⏵⏵ auto mode on (shift+tab to cycle) · esc to interrupt · ← for agents    /rc", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := fakeHost(t, map[string]string{
				"FAKE_TMUX_LINE": c.line,
				"FAKE_TMUX_LIST": "3\t2024-01-01 10:00:00\tclaude\tprojects/ccsm\n",
			})
			out, err := h.sessionStatus("3")
			if err != nil {
				t.Fatal(err)
			}
			if got := out["working"]; got != c.want {
				t.Errorf("working = %v, want %v", got, c.want)
			}
		})
	}
}

func TestAliveConversations(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	os.MkdirAll(binDir, 0700)
	psPath := filepath.Join(binDir, "ps")
	if err := os.WriteFile(psPath, []byte(fakePs), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	uuid := "00000000-0000-0000-0000-0000000000cc"

	t.Run("resume uuid in argv", func(t *testing.T) {
		h := fakeHost(t, map[string]string{
			"FAKE_PS_OUT": "claude /home/x/claude --resume " + uuid + "\n",
		})
		alive := h.aliveConversations()
		if !alive[uuid] {
			t.Errorf("resume uuid not marked alive: %v", alive)
		}
	})

	t.Run("ps failure", func(t *testing.T) {
		h := fakeHost(t, map[string]string{"FAKE_PS_FAIL": "1"})
		if len(h.aliveConversations()) != 0 {
			t.Error("expected empty on ps failure")
		}
	})

	t.Run("fresh session marks recent file", func(t *testing.T) {
		h := fakeHost(t, map[string]string{"FAKE_PS_OUT": "claude /home/x/claude\n"})
		os.WriteFile(h.convPath+"/"+uuid+".jsonl", []byte("x\n"), 0600)
		alive := h.aliveConversations()
		if !alive[uuid] {
			t.Errorf("recent file not marked alive: %v", alive)
		}
	})

	t.Run("old file not alive", func(t *testing.T) {
		h := fakeHost(t, map[string]string{"FAKE_PS_OUT": "claude /home/x/claude\n"})
		old := time.Now().Add(-2 * time.Hour)
		os.Chtimes(h.convPath+"/"+uuid+".jsonl", old, old)
		alive := h.aliveConversations()
		if alive[uuid] {
			t.Error("old file marked alive")
		}
	})

	t.Run("non-uuid file ignored", func(t *testing.T) {
		h := fakeHost(t, map[string]string{"FAKE_PS_OUT": "claude /home/x/claude\n"})
		os.WriteFile(h.convPath+"/report.jsonl", []byte("x\n"), 0600)
		if len(h.aliveConversations()) != 0 {
			t.Error("non-uuid file leaked into alive set")
		}
	})
}

func TestConversationsListSkipsConflictsAndPages(t *testing.T) {
	h := fakeHost(t, nil)
	id1 := "00000000-0000-0000-0000-0000000000dd"
	id2 := "00000000-0000-0000-0000-0000000000ee"
	id3 := "00000000-0000-0000-0000-0000000000ff"
	content := func(txt string) string {
		return `{"type":"user","cwd":"/home/admin/x","message":{"content":"` + txt + `"}}` + "\n"
	}
	os.WriteFile(h.convPath+"/"+id1+".jsonl", []byte(content("primer proyecto")), 0600)
	os.WriteFile(h.convPath+"/"+id2+".jsonl", []byte(content("SEGUNDO Proyecto")), 0600)
	os.WriteFile(h.convPath+"/"+id3+".jsonl", []byte(content("tercero")), 0600)
	// Syncthing-style conflict files and non-uuid files must be ignored.
	os.WriteFile(h.convPath+"/"+id1+".sync-conflict-20240101-000000.jsonl", []byte(content("conflicto")), 0600)
	os.WriteFile(h.convPath+"/nota.txt", []byte("x"), 0600)

	// Case-insensitive search.
	data, err := h.Exec("conversations-ls", map[string]string{"q": "proyecto"})
	if err != nil {
		t.Fatal(err)
	}
	list := data.([]map[string]any)
	if len(list) != 2 {
		t.Errorf("want 2 matching, got %d", len(list))
	}

	// Page beyond range → empty.
	data, err = h.Exec("conversations-ls", map[string]string{"page": "9", "per_page": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if list := data.([]map[string]any); len(list) != 0 {
		t.Errorf("page 9 should be empty, got %d", len(list))
	}
}

func TestExtractTextBranches(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
		ok      bool
	}{
		{"empty", "", "", false},
		{"plain string", `"hola"`, "hola", true},
		{"text blocks", `[{"type":"text","text":"uno"},{"type":"text","text":"dos"}]`, "uno dos", true},
		{"only tool blocks", `[{"type":"tool_use","name":"Bash"}]`, "", false},
		{"not json", `[{`, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := extractText([]byte(c.content))
			if ok != c.ok || got != c.want {
				t.Errorf("extractText(%s) = %q,%v want %q,%v", c.name, got, ok, c.want, c.ok)
			}
		})
	}
}

func TestConversationSummaryBranches(t *testing.T) {
	h := fakeHost(t, nil)
	path := h.convPath + "/sum.jsonl"
	lines := []string{
		`not json`,
		`{"type":"assistant","message":{"content":"soy la IA"}}`,
		`{"type":"user","isMeta":true,"message":{"content":"meta"}}`,
		`{"type":"user","message":{"content":"<system>prompt</system>"}}`,
		`{"type":"user","message":{"content":"real message"}}`,
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)

	text, _, ok := conversationSummary(path)
	if !ok || text != "real message" {
		t.Errorf("summary: %q, %v", text, ok)
	}

	// Missing file.
	if _, _, ok := conversationSummary(path + ".nope"); ok {
		t.Error("missing file should not be ok")
	}

	// Too many non-matching lines (>200) → not ok.
	longPath := h.convPath + "/long.jsonl"
	var buf strings.Builder
	for i := 0; i < 250; i++ {
		buf.WriteString(`{"type":"user","isMeta":true,"message":{"content":"x"}}` + "\n")
	}
	os.WriteFile(longPath, []byte(buf.String()), 0600)
	if _, _, ok := conversationSummary(longPath); ok {
		t.Error("250 meta lines should yield no summary")
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("corto", 10); got != "corto" {
		t.Errorf("short: %q", got)
	}
	if got := truncateRunes("hola mundo largo", 6); !strings.HasSuffix(got, "…") {
		t.Errorf("long: %q", got)
	}
}

func TestOriginFor(t *testing.T) {
	if originFor("/home/admin/x") != "pi" {
		t.Error("admin → pi")
	}
	if originFor("/home/luis/x") != "pc" {
		t.Error("luis → pc")
	}
	if originFor("/tmp/x") != "?" {
		t.Error("unknown → ?")
	}
}
