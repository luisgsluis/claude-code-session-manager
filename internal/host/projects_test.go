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
		{"projects", "CLAUDE.md"}, // container with its own doc: its children are still listed
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

	// A symlinked project dir (used to share projects between machines via
	// Syncthing) must still be discovered: DirEntry.IsDir() reflects Lstat on
	// the entry itself and reports false for a symlink, so the walk must
	// resolve it before deciding to skip it.
	realTarget := filepath.Join(t.TempDir(), "piano-sheet-generator")
	if err := os.MkdirAll(realTarget, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realTarget, "CLAUDE.md"), []byte("# x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realTarget, filepath.Join(h.home, "projects", "linked")); err != nil {
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
	want := []string{"principal", "deep/y/z", "nested/x", "projects", "projects/a", "projects/b", "projects/linked"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
}

// TestClaudeNewWithProject checks a project-pinned session launches with that
// project's directory as cwd. The sessions list showing the project is a
// separate concern now (TestTmuxListProject): it reads the pane's live
// pane_current_path via projectNameForCwd rather than a tag set at launch, so
// there's nothing launch-time left to assert about tagging.
func TestClaudeNewWithProject(t *testing.T) {
	newArgs := filepath.Join(t.TempDir(), "new_args")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_NEW_ARGS": newArgs})

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
	args, err := os.ReadFile(newArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "-c "+proj) {
		t.Errorf("expected launch with -c %s, got %q", proj, args)
	}
}

func TestClaudeNewPrincipalLaunchesAtHome(t *testing.T) {
	newArgs := filepath.Join(t.TempDir(), "new_args")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_NEW_ARGS": newArgs})

	if _, err := h.Exec("claude-nueva", map[string]string{"project": "principal"}); err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}
	args, err := os.ReadFile(newArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "-c "+h.home) {
		t.Errorf("expected launch at home %s, got %q", h.home, args)
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
