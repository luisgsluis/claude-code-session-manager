package voice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validPrompt is a minimal well-formed meta-prompt for tests that need one.
const validPrompt = `---
roles:
  - {id: auto, es: Automático, en: Auto}
  - {id: devops, es: DevOps, en: DevOps}
  - {id: docs, es: Documentación, en: Documentation}
---
# Base

Shared rules. Ask at most MAX_QUESTIONS questions.

# Role: auto

Pick a role.

# Role: devops

Operate running systems.

# Role: docs

Write documentation.
`

// TestOriginalIsValid guards the file that ships with the binary. Without
// this, a typo in rewrite.md would only be discovered by a user pressing the
// microphone and getting a 500.
func TestOriginalIsValid(t *testing.T) {
	p, err := ParsePrompt(Original())
	if err != nil {
		t.Fatalf("the shipped meta-prompt does not parse: %v", err)
	}
	for _, want := range []string{"auto", "software", "arch", "devops", "debug", "docs"} {
		if !p.HasRole(want) {
			t.Errorf("shipped meta-prompt is missing the %q role", want)
		}
	}
	if !strings.Contains(p.Base, "MAX_QUESTIONS") {
		t.Error("Base never mentions MAX_QUESTIONS, so the question cap is not passed to the model")
	}
	// The whole point of writing the meta-prompt in English is that it still
	// answers in whatever language was dictated.
	if !strings.Contains(strings.ToLower(p.Base), "language") {
		t.Error("Base gives the model no instruction about the output language")
	}
}

func TestParsePromptRejects(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no front matter", "# Base\n\nhi\n", "front matter"},
		{"broken yaml", "---\nroles: [oops\n---\n# Base\n\nhi\n", "invalid front matter"},
		{"no roles", "---\nroles: []\n---\n# Base\n\nhi\n", "no roles"},
		{
			"role without a block",
			"---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\nhi\n",
			"has no '# Role: devops' section",
		},
		{
			"block without a declaration",
			"---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\nhi\n\n# Role: devops\n\nx\n\n# Role: ghost\n\ny\n",
			"not declared",
		},
		{
			"duplicate role id",
			"---\nroles:\n  - {id: a, es: A, en: A}\n  - {id: a, es: A, en: A}\n---\n# Base\n\nhi\n\n# Role: a\n\nx\n",
			"duplicate role id",
		},
		{
			"invalid role id",
			"---\nroles:\n  - {id: \"Dev Ops\", es: D, en: D}\n---\n# Base\n\nhi\n",
			"invalid role id",
		},
		{
			"missing label",
			"---\nroles:\n  - {id: devops, es: D}\n---\n# Base\n\nhi\n\n# Role: devops\n\nx\n",
			"both es and en labels",
		},
		{
			"no base section",
			"---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Role: devops\n\nx\n",
			"'# Base' section",
		},
		{
			"empty base section",
			"---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\n# Role: devops\n\nx\n",
			"empty",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParsePrompt(c.in)
			if err == nil {
				t.Fatal("expected a parse error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not mention %q", err, c.want)
			}
		})
	}
}

// TestSystemAssembly: a forced role must not carry the other roles' blocks
// (that is the whole benefit of sectioning the file), while auto must carry
// all of them — the model cannot classify into roles it has not been shown.
func TestSystemAssembly(t *testing.T) {
	p, err := ParsePrompt(validPrompt)
	if err != nil {
		t.Fatal(err)
	}

	forced := p.System("devops", 3)
	if !strings.Contains(forced, "Operate running systems") {
		t.Error("forced role is missing its own block")
	}
	if strings.Contains(forced, "Write documentation") {
		t.Error("forced role leaked another role's block")
	}
	if !strings.Contains(forced, "Ask at most 3 questions") {
		t.Errorf("MAX_QUESTIONS was not substituted: %q", forced)
	}

	auto := p.System(AutoRole, 2)
	for _, want := range []string{"Pick a role", "Operate running systems", "Write documentation"} {
		if !strings.Contains(auto, want) {
			t.Errorf("auto mode is missing %q", want)
		}
	}
	if !strings.Contains(auto, "Ask at most 2 questions") {
		t.Error("MAX_QUESTIONS was not substituted in auto mode")
	}

	// An unknown role degrades to showing everything rather than sending a
	// system prompt with no role guidance at all.
	if got := p.System("nonexistent", 1); !strings.Contains(got, "Operate running systems") {
		t.Error("an unknown role should fall back to the full set")
	}
}

func TestPromptStoreFallsBackToOriginal(t *testing.T) {
	s := PromptStore{Dir: t.TempDir()}

	raw, custom := s.Load()
	if custom {
		t.Error("an empty directory must not report a custom prompt")
	}
	if raw != Original() {
		t.Error("an empty directory must serve the embedded original")
	}
	if len(s.Versions()) != 0 {
		t.Error("an empty directory has no versions")
	}
}

func TestPromptStoreSaveArchivesAndReset(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}

	v1 := strings.Replace(validPrompt, "Shared rules.", "Shared rules v1.", 1)
	if err := s.Save(v1); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if raw, custom := s.Load(); !custom || !strings.Contains(raw, "v1") {
		t.Fatalf("v1 not active: custom=%v", custom)
	}
	// Nothing to archive on the first save: there was no previous override.
	if got := s.Versions(); len(got) != 0 {
		t.Errorf("first save should archive nothing, got %v", got)
	}

	v2 := strings.Replace(validPrompt, "Shared rules.", "Shared rules v2.", 1)
	if err := s.Save(v2); err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if raw, _ := s.Load(); !strings.Contains(raw, "v2") {
		t.Error("v2 is not the active prompt")
	}
	versions := s.Versions()
	if len(versions) != 1 || versions[0] != 1 {
		t.Fatalf("expected exactly version 1 archived, got %v", versions)
	}
	archived, err := s.Version(1)
	if err != nil || !strings.Contains(archived, "v1") {
		t.Errorf("archived version 1 should hold the v1 text: %v / %q", err, archived)
	}

	v3 := strings.Replace(validPrompt, "Shared rules.", "Shared rules v3.", 1)
	if err := s.Save(v3); err != nil {
		t.Fatalf("save v3: %v", err)
	}
	// Newest first, so the UI lists the most recent rollback target at the top.
	if got := s.Versions(); len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Errorf("versions should be newest-first [2 1], got %v", got)
	}

	if err := s.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if raw, custom := s.Load(); custom || raw != Original() {
		t.Error("reset must restore the embedded original")
	}
	// History survives a reset: going back to the default is not a reason to
	// throw away what was tried.
	if got := s.Versions(); len(got) != 2 {
		t.Errorf("reset must keep the archived versions, got %v", got)
	}
	if err := s.Reset(); err != nil {
		t.Errorf("reset must be idempotent: %v", err)
	}
}

// TestPromptStoreRejectsInvalidWithoutTouchingDisk is the important one: a bad
// edit from the UI must leave the working prompt in place, not archive it and
// then fail with nothing active.
func TestPromptStoreRejectsInvalidWithoutTouchingDisk(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	if err := s.Save(validPrompt); err != nil {
		t.Fatal(err)
	}

	broken := "---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\nno block for devops\n"
	if err := s.Save(broken); err == nil {
		t.Fatal("expected an invalid prompt to be rejected")
	}
	raw, custom := s.Load()
	if !custom || raw != validPrompt {
		t.Error("a rejected save must leave the previous prompt exactly as it was")
	}
	if got := s.Versions(); len(got) != 0 {
		t.Errorf("a rejected save must not archive anything, got %v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "rewrite.md.tmp")); !os.IsNotExist(err) {
		t.Error("a temp file was left behind")
	}
}

// TestActiveFallsBackWhenOverrideIsCorrupt: Save cannot write a broken file,
// but somebody editing it by hand on the host can. That must not take the
// dictation button down.
func TestActiveFallsBackWhenOverrideIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	if err := os.WriteFile(filepath.Join(dir, "rewrite.md"), []byte("garbage, no front matter"), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := s.Active()
	if err != nil {
		t.Fatalf("Active should fall back rather than fail: %v", err)
	}
	if !p.HasRole("devops") {
		t.Error("the fallback should be the embedded original")
	}
}

func TestPromptStoreWithoutDir(t *testing.T) {
	s := PromptStore{}
	if raw, custom := s.Load(); custom || raw != Original() {
		t.Error("an unconfigured store serves the original")
	}
	if err := s.Save(validPrompt); err == nil {
		t.Error("an unconfigured store cannot save")
	}
	if err := s.Reset(); err == nil {
		t.Error("an unconfigured store cannot reset")
	}
}
