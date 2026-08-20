package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSessionResize covers the fix for Claude Code wrapping its TUI output to
// tmux's default 80x24 (no real client ever attaches to these detached
// sessions): resizing must set window-size to manual (or an explicit -x/-y
// would be overridden the moment tmux reconsiders client geometry) before
// forcing the exact size.
func TestSessionResize(t *testing.T) {
	optsFile := filepath.Join(t.TempDir(), "opts")
	resizeArgs := filepath.Join(t.TempDir(), "resize_args")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_OPTS": optsFile, "FAKE_TMUX_RESIZE_ARGS": resizeArgs})

	data, err := h.Exec("session-resize", map[string]string{"name": "3", "cols": "137", "rows": "42"})
	if err != nil {
		t.Fatalf("session-resize: %v", err)
	}
	out := data.(map[string]any)
	if out["cols"] != 137 || out["rows"] != 42 || out["session"] != "3" {
		t.Errorf("unexpected result: %+v", out)
	}

	opts, err := os.ReadFile(optsFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(opts)) != "manual" {
		t.Errorf("expected window-size set to manual, got %q", opts)
	}

	args, err := os.ReadFile(resizeArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "-t =3") || !strings.Contains(string(args), "-x 137") || !strings.Contains(string(args), "-y 42") {
		t.Errorf("resize-window args: %q", args)
	}
}

func TestSessionResizeSessionNotFound(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_DEAD": "3"})
	if _, err := h.Exec("session-resize", map[string]string{"name": "3", "cols": "100", "rows": "30"}); errStatus(err) != 404 {
		t.Errorf("expected 404, got %v (err=%v)", errStatus(err), err)
	}
}

func TestSessionResizeInvalidName(t *testing.T) {
	h := fakeHost(t, nil)
	if _, err := h.Exec("session-resize", map[string]string{"name": "a!b", "cols": "100", "rows": "30"}); errStatus(err) != 400 {
		t.Errorf("expected 400, got %v (err=%v)", errStatus(err), err)
	}
}

func TestSessionResizeInvalidNumbers(t *testing.T) {
	h := fakeHost(t, nil)
	if _, err := h.Exec("session-resize", map[string]string{"name": "3", "cols": "not-a-number", "rows": "30"}); errStatus(err) != 400 {
		t.Errorf("cols=not-a-number: expected 400, got %v (err=%v)", errStatus(err), err)
	}
	if _, err := h.Exec("session-resize", map[string]string{"name": "3", "cols": "100", "rows": "not-a-number"}); errStatus(err) != 400 {
		t.Errorf("rows=not-a-number: expected 400, got %v (err=%v)", errStatus(err), err)
	}
}

// TestSessionResizeOutOfRange guards against a bogus/hostile client value
// (e.g. a stale layout measurement or a manipulated request) turning the pane
// unusable — tmux itself enforces no sane limits of its own.
func TestSessionResizeOutOfRange(t *testing.T) {
	h := fakeHost(t, nil)
	cases := []map[string]string{
		{"name": "3", "cols": "1", "rows": "30"},
		{"name": "3", "cols": "5000", "rows": "30"},
		{"name": "3", "cols": "100", "rows": "1"},
		{"name": "3", "cols": "100", "rows": "5000"},
	}
	for _, args := range cases {
		if _, err := h.Exec("session-resize", args); errStatus(err) != 400 {
			t.Errorf("%v: expected 400, got %v (err=%v)", args, errStatus(err), err)
		}
	}
}

func TestSessionResizeTmuxFailure(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_RESIZE_FAIL": "1"})
	if _, err := h.Exec("session-resize", map[string]string{"name": "3", "cols": "100", "rows": "30"}); errStatus(err) != 500 {
		t.Errorf("expected 500, got %v (err=%v)", errStatus(err), err)
	}
}
