package main

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/host"
)

func TestResolveSecret(t *testing.T) {
	t.Run("flag value", func(t *testing.T) {
		got, err := resolveSecret("", "abc")
		if err != nil || got != "abc" {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("file overrides flag", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "secret")
		if err := os.WriteFile(p, []byte("  from-file\n"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := resolveSecret(p, "flag-value")
		if err != nil || got != "from-file" {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("both empty is an error", func(t *testing.T) {
		if _, err := resolveSecret("", ""); err == nil {
			t.Fatal("expected error for empty secret")
		}
	})
	t.Run("missing file is an error", func(t *testing.T) {
		if _, err := resolveSecret(filepath.Join(t.TempDir(), "nope"), ""); err == nil {
			t.Fatal("expected error for missing file")
		}
	})
	t.Run("empty file is an error", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "empty")
		os.WriteFile(p, []byte("  \n"), 0600)
		if _, err := resolveSecret(p, ""); err == nil {
			t.Fatal("expected error for empty file")
		}
	})
}

func TestEnsureSingleFresh(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "agent.sock")
	if err := ensureSingle(sock); err != nil {
		t.Fatalf("fresh socket should succeed: %v", err)
	}
}

func TestEnsureSingleStaleRemoved(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "agent.sock")
	// A socket file with no listener behind it.
	if err := os.WriteFile(sock, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureSingle(sock); err != nil {
		t.Fatalf("stale socket should be removed: %v", err)
	}
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Errorf("stale socket not removed")
	}
}

func TestEnsureSingleAlreadyRunning(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "agent.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	err = ensureSingle(sock)
	if err == nil {
		t.Fatal("expected refusal on live socket")
	}
	var re *ErrAlreadyRunning
	if !errors.As(err, &re) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
	if !strings.Contains(err.Error(), "already listening") {
		t.Errorf("unhelpful message: %s", err)
	}
}

func newTestHostForAgent(t *testing.T) (*host.Host, string) {
	t.Helper()
	base := t.TempDir()
	profiles := filepath.Join(base, "perfiles")
	os.MkdirAll(profiles, 0700)
	return host.New(host.Options{
		ProfilesPath: profiles,
		SettingsPath: filepath.Join(base, "settings.json"),
		ConvPath:     filepath.Join(base, "conv"),
		Home:         base,
	}), profiles
}

func TestHandleExecMethod(t *testing.T) {
	h, _ := newTestHostForAgent(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/exec", nil)
	handleExec(w, req, "secret", h)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

func TestHandleExecInvalidJSON(t *testing.T) {
	h, _ := newTestHostForAgent(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/exec", strings.NewReader("not json"))
	handleExec(w, req, "secret", h)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestHandleExecWrongSecret(t *testing.T) {
	h, _ := newTestHostForAgent(t)
	w := httptest.NewRecorder()
	body := `{"cmd":"profiles-ls","secret":"wrong"}`
	req := httptest.NewRequest("POST", "/exec", strings.NewReader(body))
	handleExec(w, req, "correct", h)
	if w.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", w.Code)
	}
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(resp.Error, "unauthorized") {
		t.Errorf("error: %s", resp.Error)
	}
}

func TestHandleExecHostErrorStatus(t *testing.T) {
	h, _ := newTestHostForAgent(t)
	// claude-perfil with a missing profile → 404 from the host.
	body := `{"cmd":"claude-perfil","args":{"profile":"nohay"},"secret":"s"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/exec", strings.NewReader(body))
	handleExec(w, req, "s", h)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.OK || !strings.Contains(resp.Error, "profile") {
		t.Errorf("resp: %+v", resp)
	}
}

func TestHandleExecUnknownCommand(t *testing.T) {
	h, _ := newTestHostForAgent(t)
	body := `{"cmd":"nonesuch","secret":"s"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/exec", strings.NewReader(body))
	handleExec(w, req, "s", h)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestHandleExecSuccess(t *testing.T) {
	h, profiles := newTestHostForAgent(t)
	if err := os.WriteFile(profiles+"/estandar.json", []byte(`{"model":"sonnet"}`), 0600); err != nil {
		t.Fatal(err)
	}
	body := `{"cmd":"profiles-ls","secret":"s"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/exec", strings.NewReader(body))
	handleExec(w, req, "s", h)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK {
		t.Errorf("OK false: %+v", resp)
	}
	if !strings.Contains(w.Body.String(), "estandar") {
		t.Errorf("missing profile in body: %s", w.Body.String())
	}
}

func TestHandleHealth(t *testing.T) {
	w := httptest.NewRecorder()
	handleHealth(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != http.StatusOK {
		t.Errorf("got %d", w.Code)
	}
}
