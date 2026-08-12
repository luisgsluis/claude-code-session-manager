package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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
