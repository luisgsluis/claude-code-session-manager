package host

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// spawnWithRc starts a long-running shell whose argv carries a --remote-control
// token (as claude's does when launched with Remote Control). Returns its pid.
func spawnWithRc(t *testing.T, withRc bool) int {
	t.Helper()
	args := []string{"sh", "-c", "sleep 60"}
	if withRc {
		args = append(args, "--remote-control")
	}
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	return cmd.Process.Pid
}

// TestRCStatusArgvFallback: an alive pane process started with --remote-control
// is reported rc_connected even when the status bar shows no /rc badge (the
// badge is hidden by the mode hint). The wait loop must NOT trust that signal:
// rcStatusLive stays pending, since argv is RC intent, not a confirmed bridge.
func TestRCStatusArgvFallback(t *testing.T) {
	pid := spawnWithRc(t, true)
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE":         "idle",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
	})
	if got := h.rcStatus("3"); got != "rc_connected" {
		t.Errorf("rcStatus with --remote-control argv = %q, want rc_connected", got)
	}
	if got := h.rcStatusLive("3"); got != "rc_pending" {
		t.Errorf("rcStatusLive = %q, want rc_pending", got)
	}
}

func TestRCStatusArgvFallbackNoFlag(t *testing.T) {
	pid := spawnWithRc(t, false)
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE":         "idle",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
	})
	if got := h.rcStatus("3"); got != "rc_pending" {
		t.Errorf("rcStatus without flag = %q, want rc_pending", got)
	}
}

// TestRCStatusLiveSessionFile: since 2.1.228 the status bar no longer paints a
// /rc badge, so rcStatusLive must read the bridgeSessionId from the per-process
// session file (~/.claude/sessions/<pid>.json) — present = the mobile-app
// bridge registered, regardless of what the pane's status line shows.
func TestRCStatusLiveSessionFile(t *testing.T) {
	pid := spawnWithRc(t, true)
	home := t.TempDir()
	sessionsDir := filepath.Join(home, ".claude", "sessions")
	if err := os.MkdirAll(sessionsDir, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(hasBridge bool) string {
		sf := fmt.Sprintf(`{"pid":%d,"bridgeSessionId":"","status":"idle"}`, pid)
		if hasBridge {
			sf = fmt.Sprintf(`{"pid":%d,"bridgeSessionId":"session_abc123","status":"idle"}`, pid)
		}
		p := filepath.Join(sessionsDir, fmt.Sprintf("%d.json", pid))
		if err := os.WriteFile(p, []byte(sf), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("bridgeSessionId present → connected", func(t *testing.T) {
		write(true)
		h := fakeHostWithHome(t, home, map[string]string{
			"FAKE_TMUX_LINE":         "idle", // 2.1.228: no /rc badge in the bar
			"FAKE_TMUX_PANE_SESSION": "3",
			"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
		})
		if got := h.rcStatusLive("3"); got != "rc_connected" {
			t.Errorf("rcStatusLive with bridgeSessionId = %q, want rc_connected", got)
		}
	})

	t.Run("no bridgeSessionId → pending", func(t *testing.T) {
		write(false)
		h := fakeHostWithHome(t, home, map[string]string{
			"FAKE_TMUX_LINE":         "idle",
			"FAKE_TMUX_PANE_SESSION": "3",
			"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
		})
		if got := h.rcStatusLive("3"); got != "rc_pending" {
			t.Errorf("rcStatusLive without bridgeSessionId = %q, want rc_pending", got)
		}
	})
}

// fakeHostWithHome is fakeHost with the Host.home field forced to home, so the
// session-file path (~/.claude/sessions) resolves where the test wrote it.
func fakeHostWithHome(t *testing.T, home string, env map[string]string) *Host {
	t.Helper()
	h := fakeHost(t, env)
	h.home = home
	return h
}

// TestSessionRcPresses: /remote-control toggles, so a pending session gets one
// press (enable) and a connected one gets two (off+on → fresh re-registration).
func TestSessionRcPresses(t *testing.T) {
	check := func(t *testing.T, line string, wantPresses, wantRcLines int) {
		t.Helper()
		sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
		env := map[string]string{
			"FAKE_TMUX_SENDKEYS":     sendkeys,
			"FAKE_TMUX_PANE_SESSION": "3",
		}
		if line != "" {
			env["FAKE_TMUX_LINE"] = line
		}
		h := fakeHost(t, env)
		out, err := h.sessionRc("3")
		if err != nil {
			t.Fatal(err)
		}
		if presses, _ := out["presses"].(int); presses != wantPresses {
			t.Errorf("presses = %v, want %d", out["presses"], wantPresses)
		}
		data, _ := os.ReadFile(sendkeys)
		rcLines := 0
		for _, l := range strings.Split(string(data), "\n") {
			if strings.Contains(l, "/remote-control") {
				rcLines++
			}
		}
		if rcLines != wantRcLines {
			t.Errorf("literal /remote-control presses = %d, want %d", rcLines, wantRcLines)
		}
	}

	t.Run("pending → 1 press", func(t *testing.T) { check(t, "", 1, 1) })
	t.Run("connected → 2 presses", func(t *testing.T) { check(t, "/rc connected", 2, 2) })
}

// TestSessionRcStaging: a session under a non-bootstrap profile (deepseek) has no
// /remote-control command on its endpoint, so sessionRc must stage it: apply the
// bootstrap profile, send the command, wait for the real bridge, restore the
// target profile. Mirrors lanzarConStaging.
func TestSessionRcStaging(t *testing.T) {
	sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_SENDKEYS":     sendkeys,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_LINE":         "/rc connected",
	})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey"}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	out, err := h.sessionRc("3")
	if err != nil {
		t.Fatal(err)
	}
	if out["staging"] != true {
		t.Errorf("staging flag = %v, want true", out["staging"])
	}
	// Staging restores the target profile to settings.json.
	if got := h.readSettings(t); !strings.Contains(got, "apiKeyHelper") {
		t.Errorf("settings not restored to target: %s", got)
	}
	if out["status"] != "ok" {
		t.Errorf("wait outcome = %v, want ok", out["status"])
	}
	if presses, _ := out["presses"].(int); presses != 2 {
		t.Errorf("presses = %v, want 2", out["presses"])
	}
}

// TestLanzarConStagingNoBootstrap: sin el perfil de staging (estandar) el
// lanzamiento no debe fallar con "bootstrap profile": crea la sesión sin
// --remote-control y asume que no habrá bridge (status "fail").
func TestLanzarConStagingNoBootstrap(t *testing.T) {
	newargs := filepath.Join(t.TempDir(), "newargs.txt")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_NEW_ARGS": newargs,
	})
	// El perfil de staging "estandar" NO se escribe; solo un perfil sin RC.
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey"}`)

	session, status, err := h.lanzarConStaging("deepseek", "--session-id xyz", "prueba", h.home)
	if err != nil {
		t.Fatalf("lanzarConStaging sin bootstrap falló: %v", err)
	}
	if session == "" {
		t.Error("no se devolvió sesión")
	}
	if status != "fail" {
		t.Errorf("status = %q, want fail (asumir que no hay bridge)", status)
	}
	data, _ := os.ReadFile(newargs)
	if strings.Contains(string(data), "--remote-control") {
		t.Errorf("se lanzó con --remote-control pese a no haber perfil de staging: %s", data)
	}
}

// TestSessionRcNoBootstrap: una sesión bajo un perfil sin RC pero sin el perfil
// de staging no puede re-registrar el bridge. Debe asumir que no habrá bridge
// (staging:false, status:"fail") sin enviar /remote-control ni auto-recovery.
func TestSessionRcNoBootstrap(t *testing.T) {
	sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_SENDKEYS":     sendkeys,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_LINE":         "/rc connected",
	})
	// Perfil sin RC activo, sin perfil de staging "estandar".
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey"}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	out, err := h.sessionRc("3")
	if err != nil {
		t.Fatal(err)
	}
	if out["staging"] != false {
		t.Errorf("staging = %v, want false", out["staging"])
	}
	if out["status"] != "fail" {
		t.Errorf("status = %v, want fail", out["status"])
	}
	if presses, _ := out["presses"].(int); presses != 0 {
		t.Errorf("presses = %v, want 0", out["presses"])
	}
	data, _ := os.ReadFile(sendkeys)
	if strings.Contains(string(data), "/remote-control") {
		t.Errorf("se envió /remote-control sin perfil de staging: %s", data)
	}
}

// TestTmuxKillArchiveStaging: archiving (killing) a session while the active
// profile is not the bootstrap profile must stage through the bootstrap
// profile so the kill reaches the Claude app as an archive, not a bare
// disconnect, then restore
// the profile that was active. Settle margins zeroed here so this only checks
// the end state (kill happened, target profile restored) — the timing itself
// is covered separately below.
func TestTmuxKillArchiveStaging(t *testing.T) {
	old1, old2 := archiveSettleBeforeKill, archiveSettleAfterKill
	archiveSettleBeforeKill, archiveSettleAfterKill = 0, 0
	defer func() { archiveSettleBeforeKill, archiveSettleAfterKill = old1, old2 }()

	kills := filepath.Join(t.TempDir(), "kills.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_KILLS": kills})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey"}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	if _, err := h.Exec("tmux-kill", map[string]string{"name": "3"}); err != nil {
		t.Fatalf("tmux-kill: %v", err)
	}
	got, _ := os.ReadFile(kills)
	if !strings.Contains(string(got), "=3") {
		t.Errorf("kills marker: %q", got)
	}
	// The target profile must be restored after the staged kill.
	if settings := h.readSettings(t); !strings.Contains(settings, "apiKeyHelper") {
		t.Errorf("settings not restored to target after archive: %s", settings)
	}
}

// TestTmuxKillArchiveStagingSettles: the staged archive is change → settle →
// kill → settle → change back. Neither settle point has anything to poll for
// from the host side (not the live process picking up settings.json, not the
// killed process's own shutdown — tmux forgets the session the instant the
// kill signal is sent, well before the process actually exits), so both are
// fixed pauses (archiveSettleBeforeKill/archiveSettleAfterKill). Forcing them
// down here proves both actually run — a plain apply→kill→restore with no
// wait at all would finish in well under a millisecond.
func TestTmuxKillArchiveStagingSettles(t *testing.T) {
	old1, old2 := archiveSettleBeforeKill, archiveSettleAfterKill
	archiveSettleBeforeKill, archiveSettleAfterKill = 50*time.Millisecond, 50*time.Millisecond
	defer func() { archiveSettleBeforeKill, archiveSettleAfterKill = old1, old2 }()

	kills := filepath.Join(t.TempDir(), "kills.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_KILLS": kills})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey"}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if _, err := h.Exec("tmux-kill", map[string]string{"name": "3"}); err != nil {
		t.Fatalf("tmux-kill: %v", err)
	}
	if d := time.Since(start); d < 90*time.Millisecond {
		t.Errorf("staged archive took %v; want >= ~100ms (two 50ms settle margins, before and after the kill)", d)
	}
	got, _ := os.ReadFile(kills)
	if !strings.Contains(string(got), "=3") {
		t.Errorf("kills marker: %q", got)
	}
	if settings := h.readSettings(t); !strings.Contains(settings, "apiKeyHelper") {
		t.Errorf("settings not restored to target after archive: %s", settings)
	}
}

// TestTmuxKillArchiveNoBootstrap: without the staging profile the archive
// dance is impossible, so the kill must still go through untouched rather
// than fail the whole archive over a missing bootstrap profile.
func TestTmuxKillArchiveNoBootstrap(t *testing.T) {
	kills := filepath.Join(t.TempDir(), "kills.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_KILLS": kills})
	// No "estandar" bootstrap profile written.
	h.writeProfile(t, "deepseek", `{"apiKeyHelper":"/x/claude-apikey"}`)
	if err := h.applyProfile("deepseek"); err != nil {
		t.Fatal(err)
	}

	if _, err := h.Exec("tmux-kill", map[string]string{"name": "3"}); err != nil {
		t.Fatalf("tmux-kill sin bootstrap falló: %v", err)
	}
	got, _ := os.ReadFile(kills)
	if !strings.Contains(string(got), "=3") {
		t.Errorf("kills marker: %q", got)
	}
	if settings := h.readSettings(t); !strings.Contains(settings, "apiKeyHelper") {
		t.Errorf("settings should stay untouched without a bootstrap profile: %s", settings)
	}
}

// TestTmuxKillArchiveNoStagingWhenRCProfile: archiving under the bootstrap
// profile itself (no staging needed) must not touch settings.json at all.
func TestTmuxKillArchiveNoStagingWhenRCProfile(t *testing.T) {
	kills := filepath.Join(t.TempDir(), "kills.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_KILLS": kills})
	h.writeProfile(t, "estandar", `{"model":"sonnet"}`)
	if err := h.applyProfile("estandar"); err != nil {
		t.Fatal(err)
	}

	if _, err := h.Exec("tmux-kill", map[string]string{"name": "3"}); err != nil {
		t.Fatalf("tmux-kill: %v", err)
	}
	if settings := h.readSettings(t); !strings.Contains(settings, "sonnet") {
		t.Errorf("settings changed without needing staging: %s", settings)
	}
}

// spawnRcSession starts a long-running shell whose argv carries --remote-control
// and a pinned --session-id, as a CCSM-launched Claude does.
func spawnRcSession(t *testing.T, id string) int {
	t.Helper()
	cmd := exec.Command("sh", "-c", "sleep 60", "--remote-control", "--session-id", id)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	return cmd.Process.Pid
}

// TestSessionRcAutoRecover: when the bridge doesn't come back after
// /remote-control, sessionRc relaunches the session: kills it and resumes the
// conversation, whose two-phase launch does register the bridge. This is the
// recovery path for a staged (non-bootstrap) session that lost its worker role (code 4090).
// The relaunched session keeps the same tmux name — FAKE_TMUX_KILL_MARKS_DEAD
// makes "3" report dead once actually killed, matching a real kill-session,
// so the resume's request for that name back succeeds instead of falling back
// to a new number.
func TestSessionRcAutoRecover(t *testing.T) {
	id := "a1b2c3d4-1111-2222-3333-444455556666"
	pid := spawnRcSession(t, id)
	sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
	kills := filepath.Join(t.TempDir(), "kills.txt")
	newArgs := filepath.Join(t.TempDir(), "newargs.txt")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_SENDKEYS":        sendkeys,
		"FAKE_TMUX_KILLS":           kills,
		"FAKE_TMUX_NEW_ARGS":        newArgs,
		"FAKE_TMUX_PANE_SESSION":    "3",
		"FAKE_TMUX_PANE_PID":        strconv.Itoa(pid),
		"FAKE_TMUX_KILL_MARKS_DEAD": "1",
	})
	if err := os.WriteFile(filepath.Join(h.convPath, id+".jsonl"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	out, err := h.sessionRc("3")
	if err != nil {
		t.Fatal(err)
	}
	if out["recovered"] != true {
		t.Fatalf("recovered = %v, want true", out["recovered"])
	}
	if s, _ := out["session"].(string); s != "3" {
		t.Errorf("session = %q, want 3 (relaunched under its own name)", s)
	}
	kdata, _ := os.ReadFile(kills)
	if !strings.Contains(string(kdata), "3") {
		t.Errorf("kill-session of the old session not recorded: %q", string(kdata))
	}
	nargs, _ := os.ReadFile(newArgs)
	if !strings.Contains(string(nargs), "-s 3") {
		t.Errorf("new-session should have requested the same name back (-s 3): %q", string(nargs))
	}
}

// TestSessionRcAutoRecoverNameTaken: if the requested name is somehow still
// taken (FAKE_TMUX_DEAD unset → has-session reports "3" alive even after the
// kill), the recovery falls back to auto-naming instead of failing outright.
func TestSessionRcAutoRecoverNameTaken(t *testing.T) {
	id := "a1b2c3d4-1111-2222-3333-444455556666"
	pid := spawnRcSession(t, id)
	kills := filepath.Join(t.TempDir(), "kills.txt")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_KILLS":        kills,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
		"FAKE_TMUX_NEW_NAME":     "5",
	})
	if err := os.WriteFile(filepath.Join(h.convPath, id+".jsonl"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	out, err := h.sessionRc("3")
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := out["session"].(string); s != "5" {
		t.Errorf("session = %q, want 5 (fell back to auto-naming)", s)
	}
}

// TestClaudeResumeRejectsActiveConv: resuming a conversation that another
// session already has open with the mobile bridge registered would disconnect
// it (code 4090 "no longer the active worker"); the resume must return 409 and
// warn, not kill the other one silently.
func TestClaudeResumeRejectsActiveConv(t *testing.T) {
	id := "a1b2c3d4-1111-2222-3333-444455556666"
	pid := spawnRcSession(t, id)
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LIST":         "3\t2026-01-01 00:00:00\tclaude\n",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
	})
	if err := os.WriteFile(filepath.Join(h.convPath, id+".jsonl"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	sdir := filepath.Join(h.home, ".claude", "sessions")
	if err := os.MkdirAll(sdir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sdir, fmt.Sprintf("%d.json", pid)), []byte(`{"bridgeSessionId":"sess"}`), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := h.claudeResume(id)
	if err == nil {
		t.Fatal("claudeResume: want 409 for active conversation, got nil")
	}
	var e *Error
	if !errors.As(err, &e) || e.Status != 409 {
		t.Fatalf("want *Error{409}, got %v", err)
	}
}

// TestWaitRCBridgeSettledWaitsForIdle: a resume only survives the profile
// restore once its process has finished loading (status idle). While busy,
// waitRCBridgeSettled keeps waiting; once idle with the bridge, ok.
func TestWaitRCBridgeSettledWaitsForIdle(t *testing.T) {
	pid := spawnRcSession(t, "a1b2c3d4-1111-2222-3333-444455556666")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
	})
	sdir := filepath.Join(h.home, ".claude", "sessions")
	if err := os.MkdirAll(sdir, 0700); err != nil {
		t.Fatal(err)
	}
	sf := filepath.Join(sdir, fmt.Sprintf("%d.json", pid))
	if err := os.WriteFile(sf, []byte(`{"status":"busy","bridgeSessionId":"sess"}`), 0600); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(sf, []byte(`{"status":"idle","bridgeSessionId":"sess"}`), 0600)
	}()
	if got := h.waitRCBridgeSettled("3"); got != "ok" {
		t.Errorf("settled = %q, want ok", got)
	}
}

// TestWaitRCBridgeSettledDoesNotBurnBudgetOnManualDialog: a session that comes
// up blocked on an interactive TUI dialog (trust prompt, command/file-edit
// approval) never registers the bridge until a human answers it — that's a
// manual wait, not a stuck bridge. waitRCBridgeSettled must not let that time
// count against rcWaitSeconds: the pane here shows a blocking dialog for
// longer than the (tiny) configured budget, and only registers the bridge
// once it clears. Without paneWaitingDetail pushing the deadline forward
// while blocked, this would time out well before the dialog ever clears.
func TestWaitRCBridgeSettledDoesNotBurnBudgetOnManualDialog(t *testing.T) {
	pid := spawnRcSession(t, "a1b2c3d4-1111-2222-3333-444455556667")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     strconv.Itoa(pid),
		"FAKE_TMUX_LINE":         "Do you trust the files in this folder?\n❯ 1. Yes\n  2. No\n",
	})
	h.rcWaitSeconds = 1 // smaller than the dialog's simulated duration below

	go func() {
		time.Sleep(1300 * time.Millisecond)
		os.Setenv("FAKE_TMUX_LINE", "idle")
		sdir := filepath.Join(h.home, ".claude", "sessions")
		os.MkdirAll(sdir, 0700)
		sf := filepath.Join(sdir, fmt.Sprintf("%d.json", pid))
		os.WriteFile(sf, []byte(`{"status":"idle","bridgeSessionId":"sess"}`), 0600)
	}()

	if got := h.waitRCBridgeSettled("3"); got != "ok" {
		t.Errorf("settled = %q, want ok (a blocking dialog must not consume the wait budget)", got)
	}
}
