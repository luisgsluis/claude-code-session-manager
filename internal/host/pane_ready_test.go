package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSessionSendWaitsForPaneReady reproduces the production incident: a
// model switch sent right after session creation, before Claude Code's own
// TUI has rendered its first frame. sessionSendLock already serializes
// concurrent sessionSend calls correctly (no interleaving of literal text),
// but a send whose own Enter lands before Claude Code is reading input
// doesn't submit anything — the literal just sits in the pty's raw buffer,
// to be swallowed into whatever gets typed next. ensurePaneReady must hold
// the send until the pane is confirmed interactive, so "/model sonnet" is
// only typed once Claude Code will actually consume its Enter.
func TestSessionSendWaitsForPaneReady(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "ready-counter")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_READY_AFTER":   "3", // not ready for 2 reads, ready on the 3rd
		"FAKE_TMUX_READY_COUNTER": counter,
		"FAKE_TMUX_PANE_SESSION":  "3",
		"FAKE_TMUX_PANE_PID":      "12345",
	})
	h.paneReadyTimeout = 2 * time.Second // fakeHost zeroes this; restore it for this test

	start := time.Now()
	resp, err := h.Exec("session-send", map[string]string{"name": "3", "text": "/model sonnet"})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("session-send: %v", err)
	}
	if m, _ := resp.(map[string]any); m["sent"] != "text" {
		t.Errorf("sent = %v, want text", m["sent"])
	}
	// paneReadyPoll is 200ms; 2 not-ready reads means at least one poll wait.
	if elapsed < paneReadyPoll {
		t.Errorf("session-send returned in %v, want it to have waited at least one poll (%v) for the pane to become ready", elapsed, paneReadyPoll)
	}
}

// TestEnsurePaneReadyFailsOpen confirms a pane that never becomes readable
// doesn't block sessionSend forever: past paneReadyTimeout it proceeds and
// sends anyway, matching the rest of the host's best-effort philosophy.
func TestEnsurePaneReadyFailsOpen(t *testing.T) {
	sends := filepath.Join(t.TempDir(), "sends")
	if err := os.WriteFile(sends, nil, 0600); err != nil {
		t.Fatal(err)
	}
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE":         "", // capture-pane always empty: never ready
		"FAKE_TMUX_SENDKEYS":     sends,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	h.paneReadyTimeout = 250 * time.Millisecond // short bound so the test stays fast

	if _, err := h.Exec("session-send", map[string]string{"name": "3", "text": "hola"}); err != nil {
		t.Fatalf("session-send: %v", err)
	}
	sent, err := os.ReadFile(sends)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sent), "hola") {
		t.Errorf("send log %q does not contain the text: ensurePaneReady blocked the send instead of failing open", sent)
	}
}

// TestEnsurePaneReadyDisabledInTests confirms fakeHost's zeroed
// paneReadyTimeout skips the wait entirely (the baseline every other test in
// this package relies on to stay fast against the fake tmux's default empty
// capture-pane).
func TestEnsurePaneReadyDisabledInTests(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	if h.paneReadyTimeout != 0 {
		t.Fatalf("fakeHost paneReadyTimeout = %v, want 0", h.paneReadyTimeout)
	}
	start := time.Now()
	h.ensurePaneReady("3") // capture-pane returns "" (FAKE_TMUX_LINE unset): not ready
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("ensurePaneReady took %v with paneReadyTimeout=0, want ~instant", elapsed)
	}
}

// TestSessionSendTextRejectedWhileApprovalBlocks reproduces the second half
// of the production incident (see TestSessionSendWaitsForPaneReady's
// docstring): ensurePaneReady alone treats a rendered approval dialog as
// "ready" (its own frame IS rendered), so text sent while a "trust this
// folder?"-style dialog is still up used to be typed straight into it — that
// dialog reads raw keystrokes through its own picker, not a readline, so the
// text just sat unread and merged with whatever was typed once the dialog
// finally cleared, e.g. "/model sonnetempezamos un proyecto nuevo". Once the
// dialog is still open past paneReadyTimeout, sessionSend must now refuse
// the text outright instead of sending it into the still-blocked pane.
func TestSessionSendTextRejectedWhileApprovalBlocks(t *testing.T) {
	sends := filepath.Join(t.TempDir(), "sends")
	if err := os.WriteFile(sends, nil, 0600); err != nil {
		t.Fatal(err)
	}
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE":         approvalPane(0), // stays blocked the whole poll window
		"FAKE_TMUX_SENDKEYS":     sends,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	h.paneReadyTimeout = 250 * time.Millisecond // short bound so the test stays fast

	if _, err := h.Exec("session-send", map[string]string{"name": "3", "text": "/model sonnet"}); err == nil {
		t.Fatal("expected session-send to reject text while an approval dialog blocks the pane, got no error")
	}
	sent, err := os.ReadFile(sends)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sent), "/model sonnet") {
		t.Errorf("text was sent into a still-blocked pane: %q", sent)
	}
}

// TestSessionSendKeyNotBlockedByApproval is the flip side: a special key
// aimed AT the dialog itself — the "Aprobar" button's Enter — must NOT wait
// for paneTextSafe (which by definition never becomes true while the dialog
// is up): it only needs ensurePaneReady's weaker "some frame is rendered"
// bar, or approving a dialog would hang for paneReadyTimeout every time.
func TestSessionSendKeyNotBlockedByApproval(t *testing.T) {
	sends := filepath.Join(t.TempDir(), "sends")
	if err := os.WriteFile(sends, nil, 0600); err != nil {
		t.Fatal(err)
	}
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE":         approvalPane(0),
		"FAKE_TMUX_SENDKEYS":     sends,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	h.paneReadyTimeout = 2 * time.Second // long enough that a wrongly-blocked call would fail this test's own timeout expectation

	start := time.Now()
	if _, err := h.Exec("session-send", map[string]string{"name": "3", "keys": "enter"}); err != nil {
		t.Fatalf("session-send enter: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("approving took %v, want ~instant (must not wait on paneTextSafe)", elapsed)
	}
	sent, err := os.ReadFile(sends)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sent), "\r") {
		t.Errorf("expected the approve Enter to reach tmux, got %q", sent)
	}
}
