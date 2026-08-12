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

// TestSessionRcStaging: a session under a perfilSinRC profile (deepseek) has no
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

// TestSessionRcAutoRecover: cuando el bridge no vuelve tras el /remote-control,
// sessionRc relanza la sesión: la mata y retoma la conversación, cuyo arranque
// en dos fases sí registra el bridge. Es la vía recuperable para una sesión con
// perfil sin RC que perdió su rol de worker (code 4090).
func TestSessionRcAutoRecover(t *testing.T) {
	id := "a1b2c3d4-1111-2222-3333-444455556666"
	pid := spawnRcSession(t, id)
	sendkeys := filepath.Join(t.TempDir(), "sendkeys.txt")
	kills := filepath.Join(t.TempDir(), "kills.txt")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_SENDKEYS":     sendkeys,
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
	if out["recovered"] != true {
		t.Fatalf("recovered = %v, want true", out["recovered"])
	}
	if s, _ := out["session"].(string); s != "5" {
		t.Errorf("session = %q, want 5 (la relanzada)", s)
	}
	kdata, _ := os.ReadFile(kills)
	if !strings.Contains(string(kdata), "3") {
		t.Errorf("kill-session de la sesión vieja no registrado: %q", string(kdata))
	}
}

// TestClaudeResumeRejectsActiveConv: retomar una conversación que otra sesión
// tiene abierta con el bridge del móvil registrado la desconectaría (code 4090
// "no longer the active worker"); el resume debe devolver 409 y advertir, no
// tumbar la otra en silencio.
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

// TestWaitRCBridgeSettledWaitsForIdle: un resume solo sobrevive a la restauración
// del perfil cuando su proceso ha terminado de cargar (status idle). Mientras
// busy, waitRCBridgeSettled sigue esperando; al pasar a idle con el bridge, ok.
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

