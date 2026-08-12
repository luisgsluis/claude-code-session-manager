package direct

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/host"
)

func newTestHost(t *testing.T) (*host.Host, string) {
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

func TestExecSuccess(t *testing.T) {
	h, profiles := newTestHost(t)
	os.WriteFile(profiles+"/estandar.json", []byte(`{"model":"sonnet"}`), 0600)
	e := New(h)

	resp, err := e.Exec("profiles-ls", nil)
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !resp.OK {
		t.Error("OK false")
	}
	var list []map[string]any
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		t.Fatalf("bad data: %v", err)
	}
	if len(list) != 1 || list[0]["name"] != "estandar" {
		t.Errorf("list: %v", list)
	}
}

func TestExecHostErrorPropagates(t *testing.T) {
	h, _ := newTestHost(t)
	e := New(h)

	// Missing profile → host returns a 404 error; it must flow through
	// untouched (the handler maps it to HTTP 404, not a blanket 502).
	_, err := e.Exec("claude-perfil", map[string]string{"profile": "nohay"})
	if err == nil {
		t.Fatal("expected error")
	}
	var he *host.Error
	if !errors.As(err, &he) || he.Status != 404 {
		t.Errorf("expected 404 host error, got %v", err)
	}
}
