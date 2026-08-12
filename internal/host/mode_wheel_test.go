package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newWheelHost builds a fakeHost whose tmux stub simulates a real Shift+Tab
// mode wheel: capture-pane returns the footer for the mode currently stored in
// the state file, and a raw shift-tab (\e[Z) advances to the next mode, wrapping
// around. The session name is "3" and always alive.
func newWheelHost(t *testing.T, wheel string) (*Host, string) {
	t.Helper()
	state := filepath.Join(t.TempDir(), "wheel")
	if err := os.WriteFile(state, []byte("0"), 0600); err != nil {
		t.Fatal(err)
	}
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_MODE_WHEEL":   wheel,
		"FAKE_TMUX_MODE_STATE":   state,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	return h, state
}

func modeWheelEq(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("wheel = %v, want %v", got, want)
	}
}

func TestDiscoverModeWheelStandard(t *testing.T) {
	h, state := newWheelHost(t, "manual,accept-edits,plan,auto")
	w, err := h.discoverModeWheel("3")
	if err != nil {
		t.Fatalf("discoverModeWheel: %v", err)
	}
	modeWheelEq(t, w, []string{"manual", "accept-edits", "plan", "auto"})
	// A full cycle leaves the session where it started (state back at index 0).
	if idx, _ := os.ReadFile(state); string(idx) != "0" {
		t.Errorf("session not restored after probe: state index = %q, want 0", idx)
	}
}

func TestDiscoverModeWheelWithBypass(t *testing.T) {
	h, _ := newWheelHost(t, "manual,accept-edits,plan,auto,bypassPermissions")
	w, err := h.discoverModeWheel("3")
	if err != nil {
		t.Fatalf("discoverModeWheel: %v", err)
	}
	modeWheelEq(t, w, []string{"manual", "accept-edits", "plan", "auto", "bypassPermissions"})
}

func TestDiscoverModeWheelNoAuto(t *testing.T) {
	h, _ := newWheelHost(t, "manual,accept-edits,plan")
	w, err := h.discoverModeWheel("3")
	if err != nil {
		t.Fatalf("discoverModeWheel: %v", err)
	}
	modeWheelEq(t, w, []string{"manual", "accept-edits", "plan"})
}

func TestSessionModeUsesDiscoveredWheel(t *testing.T) {
	h, _ := newWheelHost(t, "manual,accept-edits,plan,auto")
	resp, err := h.sessionMode("3", "auto")
	if err != nil {
		t.Fatalf("sessionMode: %v", err)
	}
	if resp["mode"] != "auto" {
		t.Errorf("mode = %v, want auto", resp["mode"])
	}
	if n, _ := resp["pressed"].(int); n != 3 {
		t.Errorf("pressed = %v, want 3 (manual→accept-edits→plan→auto)", resp["pressed"])
	}
	// The discovered wheel is now cached for the chat payload.
	if w := h.cachedModeWheel(); strings.Join(w, ",") != "manual,accept-edits,plan,auto" {
		t.Errorf("cached wheel = %v, want the discovered one", w)
	}
}

func TestSessionModeUnsupportedOnRealWheel(t *testing.T) {
	// Wheel without auto: requesting auto must fail clearly listing the real
	// modes, not silently land on the wrong one.
	h, _ := newWheelHost(t, "manual,accept-edits,plan")
	_, err := h.sessionMode("3", "auto")
	if status := errStatus(err); status != 400 {
		t.Fatalf("sessionMode(auto) status = %d, want 400 (err=%v)", status, err)
	}
	if err != nil && !strings.Contains(err.Error(), "available: manual, accept-edits, plan") {
		t.Errorf("error %q does not list the real wheel", err.Error())
	}
}

func TestDiscoverModeWheelRefusesBusy(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE":        "… ⏵⏵ auto mode on (shift+tab to cycle) · esc to interrupt · ctrl+t to hide tasks",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	if _, err := h.discoverModeWheel("3"); err == nil {
		t.Error("discoverModeWheel on a working session: want error, got nil")
	}
}

func TestWheelForBusyFallsBack(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE":        "… ⏵⏵ auto mode on (shift+tab to cycle) · esc to interrupt · ctrl+t to hide tasks",
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	w := h.wheelFor("3")
	modeWheelEq(t, w, modeWheel) // standard wheel, previous behaviour
}

func TestCachedModeWheelEmptyBeforeCalibration(t *testing.T) {
	h, _ := newWheelHost(t, "manual,accept-edits,plan,auto")
	if w := h.cachedModeWheel(); w != nil {
		t.Errorf("cached wheel before any probe = %v, want nil", w)
	}
}
