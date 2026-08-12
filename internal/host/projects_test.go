package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectsList(t *testing.T) {
	h := fakeHost(t, nil)

	// A CLAUDE.md is found at various depths (1..3); dot-dirs are skipped and
	// anything deeper than 3 levels is ignored.
	dirs := []struct{ dir, file string }{
		{"projects", "CLAUDE.md"}, // contenedor con doc propio: sus hijos se listan igualmente
		{"projects/a", "CLAUDE.md"},
		{"projects/b", "claude.md"},
		{"nested/x", "CLAUDE.md"},
		{"deep/y/z", "CLAUDE.md"},
		{".hidden/x", "CLAUDE.md"},
	}
	for _, d := range dirs {
		p := filepath.Join(h.home, filepath.FromSlash(d.dir))
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, d.file), []byte("# x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p4 := filepath.Join(h.home, "too", "deep", "for", "this")
	if err := os.MkdirAll(p4, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p4, "CLAUDE.md"), []byte("# x"), 0600); err != nil {
		t.Fatal(err)
	}

	projects, err := h.projectsList()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, pr := range projects {
		names = append(names, pr["name"].(string))
	}
	want := []string{"principal", "deep/y/z", "nested/x", "projects", "projects/a", "projects/b"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
}

func TestClaudeNewWithProject(t *testing.T) {
	optsFile := filepath.Join(t.TempDir(), "opts")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_OPTS": optsFile})

	proj := filepath.Join(h.home, "projects", "ccsm")
	if err := os.MkdirAll(proj, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "CLAUDE.md"), []byte("# x"), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := h.Exec("claude-nueva", map[string]string{"project": "projects/ccsm"})
	if err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	out := data.(map[string]string)
	if out["session"] != "3" {
		t.Fatalf("unexpected session %v", out["session"])
	}
	b, err := os.ReadFile(optsFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "projects/ccsm" {
		t.Fatalf("session not tagged: %q", string(b))
	}
}

func TestClaudeNewPrincipalNotTagged(t *testing.T) {
	optsFile := filepath.Join(t.TempDir(), "opts")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_OPTS": optsFile})

	if _, err := h.Exec("claude-nueva", map[string]string{"project": "principal"}); err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	if b, err := os.ReadFile(optsFile); err == nil && strings.TrimSpace(string(b)) != "" {
		t.Fatalf("principal must not be tagged: %q", string(b))
	}
}

func TestClaudeNewUnknownProject(t *testing.T) {
	h := fakeHost(t, nil)
	if _, err := h.Exec("claude-nueva", map[string]string{"project": "does/not/exist"}); err == nil {
		t.Fatal("expected error for unknown project")
	}
	if _, err := h.Exec("claude-nueva", map[string]string{"project": "../escape"}); err == nil {
		t.Fatal("expected error for traversal project")
	}
}
