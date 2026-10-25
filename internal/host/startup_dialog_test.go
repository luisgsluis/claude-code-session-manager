package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const trustPane = ` Accessing workspace:
 /home/admin/projects/never-opened
 Quick safety check: Is this a project you created or one you trust?
 ❯ No, exit
   Yes, I trust this folder
 Enter to confirm · Esc to cancel
`

func startupHost(t *testing.T, pane string) (*Host, string) {
	t.Helper()
	dir := t.TempDir()
	paneFile := filepath.Join(dir, "pane")
	sends := filepath.Join(dir, "sends")
	if err := os.WriteFile(paneFile, []byte(pane), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sends, nil, 0600); err != nil {
		t.Fatal(err)
	}
	h := fakeHost(t, map[string]string{"FAKE_TMUX_PANE_FILE": paneFile, "FAKE_TMUX_SENDKEYS": sends})
	h.paneReadyTimeout = 600 * time.Millisecond
	return h, sends
}

// A project dir Claude has never opened parks the new session on the
// folder-trust picker: newSession must answer it (Down to "Yes, I trust this
// folder", then Enter) instead of leaving the session hung.
func TestNewSessionAcceptsFolderTrustOfNeverOpenedDir(t *testing.T) {
	h, sends := startupHost(t, trustPane)
	if _, err := h.newSession("--session-id x", "t1", h.home); err != nil {
		t.Fatalf("newSession: %v", err)
	}
	got, _ := os.ReadFile(sends)
	if !strings.Contains(string(got), "Down") || !strings.Contains(string(got), "Enter") {
		t.Errorf("send-keys log %q: want Down then Enter to accept the trust dialog", got)
	}
}

// Anything else that blocks the boot and must not be auto-answered (here an
// expired login) fails the creation with the dialog text.
func TestNewSessionReportsBlockingStartupDialog(t *testing.T) {
	h, sends := startupHost(t, "OAuth token has expired\nSelect login method:\n")
	_, err := h.newSession("--session-id x", "t1", h.home)
	if err == nil || !strings.Contains(err.Error(), "Select login method") {
		t.Fatalf("err = %v, want one carrying the dialog text", err)
	}
	if got, _ := os.ReadFile(sends); len(got) != 0 {
		t.Errorf("keys were sent into a dialog that must not be answered: %q", got)
	}
}
