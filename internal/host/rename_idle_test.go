package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Footer as Claude Code renders it today in bypass mode: before modeFromBadge
// knew the spaced spelling, this pane never looked text-safe.
const bypassIdleFooter = "────\n❯ \n────\n  ⏵⏵ bypass permissions on (shift+tab to cycle) · ← for agents\n"

// TestRenameWaitsForIdle: /rename must not be typed while the first turn is
// generating (it would be queued and merged with the next message), and must
// be typed once the session is idle.
func TestRenameWaitsForIdle(t *testing.T) {
	dir := t.TempDir()
	pane := filepath.Join(dir, "pane")
	sends := filepath.Join(dir, "sends")
	os.WriteFile(sends, nil, 0600)
	working := "✻ Cooking… (3s · ↓ 1.2k tokens)\n" + bypassIdleFooter
	if err := os.WriteFile(pane, []byte(working), 0600); err != nil {
		t.Fatal(err)
	}
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_PANE_FILE":    pane,
		"FAKE_TMUX_SENDKEYS":     sends,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	done := make(chan struct{})
	go func() { h.renameWhenIdle("3", "mi titulo"); close(done) }()

	time.Sleep(1500 * time.Millisecond)
	if b, _ := os.ReadFile(sends); strings.Contains(string(b), "/rename") {
		t.Fatalf("/rename typed while generating: %q", b)
	}
	if err := os.WriteFile(pane, []byte(bypassIdleFooter), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("renameWhenIdle did not finish once idle")
	}
	if b, _ := os.ReadFile(sends); !strings.Contains(string(b), "/rename mi titulo") {
		t.Fatalf("/rename not typed once idle: %q", b)
	}
}

// TestBypassFooterIsTextSafe: the spaced bypass badge counts as a ready pane,
// so ensureTextReady returns at once instead of waiting out its timeout.
func TestBypassFooterIsTextSafe(t *testing.T) {
	pane := filepath.Join(t.TempDir(), "pane")
	os.WriteFile(pane, []byte(bypassIdleFooter), 0600)
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_PANE_FILE":    pane,
		"FAKE_TMUX_PANE_SESSION": "3",
		"FAKE_TMUX_PANE_PID":     "12345",
	})
	if !h.paneTextSafe("3") {
		t.Fatal("bypass permissions footer not text-safe")
	}
	if got := h.paneMode("3"); got != "bypassPermissions" {
		t.Fatalf("paneMode = %q", got)
	}
}
