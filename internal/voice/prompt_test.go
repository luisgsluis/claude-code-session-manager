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

Shared rules.

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
	// The whole point of writing the meta-prompt in English is that it still
	// answers in whatever language was dictated.
	if !strings.Contains(strings.ToLower(p.Base), "language") {
		t.Error("Base gives the model no instruction about the output language")
	}
	if !strings.Contains(strings.ToLower(p.Base), "genuinely unclear") {
		t.Error("Base does not tell the model to disambiguate rather than gather more information")
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

	forced := p.System("devops")
	if !strings.Contains(forced, "Operate running systems") {
		t.Error("forced role is missing its own block")
	}
	if strings.Contains(forced, "Write documentation") {
		t.Error("forced role leaked another role's block")
	}

	auto := p.System(AutoRole)
	for _, want := range []string{"Pick a role", "Operate running systems", "Write documentation"} {
		if !strings.Contains(auto, want) {
			t.Errorf("auto mode is missing %q", want)
		}
	}

	// An unknown role degrades to showing everything rather than sending a
	// system prompt with no role guidance at all.
	if got := p.System("nonexistent"); !strings.Contains(got, "Operate running systems") {
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
	versions := s.List()
	if len(versions) != 1 || !versions[0].Original || !versions[0].Active {
		t.Errorf("an empty directory has only the original, active: %+v", versions)
	}
}

// TestPromptStoreSaveNewNeverActivates is the core of the new version model:
// saving and applying are two separate calls, so writing a new version must
// not change what dictation is actually using.
func TestPromptStoreSaveNewNeverActivates(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}

	v1 := strings.Replace(validPrompt, "Shared rules.", "Shared rules v1.", 1)
	id, err := s.SaveNew(v1, "Mi versión 1")
	if err != nil {
		t.Fatalf("save new: %v", err)
	}
	if id == 0 {
		t.Fatal("a saved version must never be assigned id 0 (reserved for the original)")
	}

	if raw, custom := s.Load(); custom || raw != Original() {
		t.Error("saving a new version must not activate it")
	}

	content, err := s.VersionContent(id)
	if err != nil || !strings.Contains(content, "v1") {
		t.Errorf("version content mismatch: %v / %q", err, content)
	}

	versions := s.List()
	if len(versions) != 2 {
		t.Fatalf("expected original + 1 saved version, got %+v", versions)
	}
	if !versions[0].Original || !versions[0].Active {
		t.Errorf("original should still be active: %+v", versions[0])
	}
	if versions[1].ID != id || versions[1].Name != "Mi versión 1" || versions[1].Active {
		t.Errorf("saved version should carry its name and not be active: %+v", versions[1])
	}
}

func TestPromptStoreSaveNewDefaultName(t *testing.T) {
	s := PromptStore{Dir: t.TempDir()}
	id, err := s.SaveNew(validPrompt, "   ")
	if err != nil {
		t.Fatal(err)
	}
	versions := s.List()
	if versions[1].Name == "" {
		t.Error("a blank name must fall back to a default, not an empty label")
	}
	if !strings.Contains(versions[1].Name, "1") && versions[1].ID != id {
		t.Errorf("unexpected default name: %q", versions[1].Name)
	}
}

func TestPromptStoreSetActive(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	v1 := strings.Replace(validPrompt, "Shared rules.", "Shared rules v1.", 1)
	id, err := s.SaveNew(v1, "v1")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SetActive(id); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if raw, custom := s.Load(); !custom || !strings.Contains(raw, "v1") {
		t.Errorf("v1 should now be active: custom=%v raw=%q", custom, raw)
	}
	for _, v := range s.List() {
		if v.ID == id && !v.Active {
			t.Error("the version list disagrees with Load about which is active")
		}
		if v.Original && v.Active {
			t.Error("the original must not also be marked active once another version is")
		}
	}

	// Applying is non-destructive: switching back to the original must not
	// have discarded the saved version.
	if err := s.SetActive(0); err != nil {
		t.Fatalf("set active back to original: %v", err)
	}
	if raw, custom := s.Load(); custom || raw != Original() {
		t.Error("activating 0 must restore the original")
	}
	if content, err := s.VersionContent(id); err != nil || !strings.Contains(content, "v1") {
		t.Errorf("the saved version must still exist after switching away from it: %v / %q", err, content)
	}

	if err := s.SetActive(9999); err == nil {
		t.Error("activating a version that does not exist must fail")
	}
}

func TestPromptStoreSaveOver(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	id, err := s.SaveNew(validPrompt, "mine")
	if err != nil {
		t.Fatal(err)
	}

	v2 := strings.Replace(validPrompt, "Shared rules.", "Shared rules v2.", 1)
	if err := s.SaveOver(id, v2); err != nil {
		t.Fatalf("save over: %v", err)
	}
	content, err := s.VersionContent(id)
	if err != nil || !strings.Contains(content, "v2") {
		t.Errorf("save over did not update the content: %v / %q", err, content)
	}
	versions := s.List()
	if len(versions) != 2 || versions[1].Name != "mine" {
		t.Errorf("save over must not rename or duplicate the version: %+v", versions)
	}

	if err := s.SaveOver(0, v2); err == nil {
		t.Error("the original must never be overwritable")
	}
	if err := s.SaveOver(9999, v2); err == nil {
		t.Error("overwriting a version that does not exist must fail")
	}
}

func TestPromptStoreRename(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	id, err := s.SaveNew(validPrompt, "original name")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Rename(id, "  new name  "); err != nil {
		t.Fatalf("rename: %v", err)
	}
	versions := s.List()
	if versions[1].Name != "new name" {
		t.Errorf("rename did not update the name (and should trim it): %+v", versions[1])
	}
	if content, err := s.VersionContent(id); err != nil || !strings.Contains(content, "Shared rules.") {
		t.Errorf("rename must not touch content: %v / %q", err, content)
	}

	if err := s.Rename(id, "   "); err == nil {
		t.Error("renaming to a blank name must be rejected")
	}
	if err := s.Rename(0, "x"); err == nil {
		t.Error("the original must not be renameable")
	}
	if err := s.Rename(9999, "x"); err == nil {
		t.Error("renaming a version that does not exist must fail")
	}
}

func TestPromptStoreDelete(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	id, err := s.SaveNew(validPrompt, "mine")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(0); err == nil {
		t.Error("the original must never be deletable")
	}
	if err := s.Delete(9999); err == nil {
		t.Error("deleting a version that does not exist must fail")
	}

	if err := s.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.VersionContent(id); err == nil {
		t.Error("a deleted version must no longer be readable")
	}
	if len(s.List()) != 1 {
		t.Errorf("delete must remove the version from the list, got %+v", s.List())
	}
	if _, err := os.Stat(s.versionPath(id)); !os.IsNotExist(err) {
		t.Errorf("delete must remove the version's file from disk: %v", err)
	}
}

// TestPromptStoreDeleteActiveFallsBackToOriginal is the important one: a
// dangling active pointer would send every future dictation to a version
// that no longer exists on disk (Load already degrades to the original in
// that case, but versions.json should not be left claiming otherwise).
func TestPromptStoreDeleteActiveFallsBackToOriginal(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	id, err := s.SaveNew(validPrompt, "mine")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetActive(id); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if raw, custom := s.Load(); custom || raw != Original() {
		t.Error("deleting the active version must fall back to the original")
	}
	for _, v := range s.List() {
		if v.Original && !v.Active {
			t.Error("the original should be active again after the active version was deleted")
		}
	}
}

// TestPromptStoreRejectsInvalidWithoutTouchingDisk is the important one: a bad
// edit from the UI must leave the working version in place, not overwrite it
// and then fail with nothing usable.
func TestPromptStoreRejectsInvalidWithoutTouchingDisk(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	id, err := s.SaveNew(validPrompt, "mine")
	if err != nil {
		t.Fatal(err)
	}

	broken := "---\nroles:\n  - {id: devops, es: D, en: D}\n---\n# Base\n\nno block for devops\n"
	if err := s.SaveOver(id, broken); err == nil {
		t.Fatal("expected an invalid overwrite to be rejected")
	}
	content, err := s.VersionContent(id)
	if err != nil || content != validPrompt {
		t.Error("a rejected overwrite must leave the previous content exactly as it was")
	}

	if _, err := s.SaveNew(broken, "broken"); err == nil {
		t.Fatal("expected an invalid new version to be rejected")
	}
	if versions := s.List(); len(versions) != 2 {
		t.Errorf("a rejected save must not add a version, got %+v", versions)
	}
	if _, err := os.Stat(filepath.Join(dir, "versions.json.tmp")); !os.IsNotExist(err) {
		t.Error("a temp file was left behind")
	}
}

// TestActiveFallsBackWhenOverrideIsCorrupt: SaveOver/SaveNew cannot write a
// broken file, but somebody editing versions.json or a version file by hand
// on the host can. That must not take the dictation button down.
func TestActiveFallsBackWhenOverrideIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	s := PromptStore{Dir: dir}
	if err := os.WriteFile(filepath.Join(dir, "rewrite.v1.md"), []byte("garbage, no front matter"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := `{"active":1,"next_id":2,"versions":[{"id":1,"name":"hand-edited","created_at":"2020-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(filepath.Join(dir, "versions.json"), []byte(meta), 0o644); err != nil {
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
	if _, err := s.SaveNew(validPrompt, "x"); err == nil {
		t.Error("an unconfigured store cannot save a new version")
	}
	if err := s.SaveOver(1, validPrompt); err == nil {
		t.Error("an unconfigured store cannot overwrite a version")
	}
	if err := s.SetActive(1); err == nil {
		t.Error("an unconfigured store cannot activate a version it does not have")
	}
	if err := s.SetActive(0); err != nil {
		t.Error("activating the original on an unconfigured store is a harmless no-op")
	}
}

// TestPromptStoreMigratesLegacyLayout covers upgrading a directory written by
// the old scheme (a single "rewrite.md" override plus auto-numbered
// "rewrite.vN.md" archives, no names, no explicit active pointer) into the
// current one, the first time it is read.
func TestPromptStoreMigratesLegacyLayout(t *testing.T) {
	dir := t.TempDir()
	archived1 := strings.Replace(validPrompt, "Shared rules.", "Shared rules archived 1.", 1)
	archived2 := strings.Replace(validPrompt, "Shared rules.", "Shared rules archived 2.", 1)
	active := strings.Replace(validPrompt, "Shared rules.", "Shared rules active override.", 1)
	for name, content := range map[string]string{
		"rewrite.v1.md": archived1,
		"rewrite.v2.md": archived2,
		"rewrite.md":    active,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := PromptStore{Dir: dir}
	raw, custom := s.Load()
	if !custom || !strings.Contains(raw, "active override") {
		t.Fatalf("migration should activate the old override: custom=%v raw=%q", custom, raw)
	}

	versions := s.List()
	if len(versions) != 4 { // original + 2 archives + the old active override
		t.Fatalf("expected 4 versions after migration, got %+v", versions)
	}
	var sawArchive1, sawArchive2, sawMigratedActive bool
	for _, v := range versions {
		switch {
		case v.Original:
		case strings.Contains(v.Name, "1 (migrada)"):
			sawArchive1 = true
			if v.Active {
				t.Error("an old archive must not come back active")
			}
		case strings.Contains(v.Name, "2 (migrada)"):
			sawArchive2 = true
		case strings.Contains(v.Name, "Migrada del editor anterior"):
			sawMigratedActive = true
			if !v.Active {
				t.Error("the old override should become the active version")
			}
		}
	}
	if !sawArchive1 || !sawArchive2 || !sawMigratedActive {
		t.Errorf("migration missed an entry: %+v", versions)
	}

	// The old scheme's files must not linger once migrated: the override is
	// folded into a version and removed, and versions.json now exists so a
	// second read does not migrate again.
	if _, err := os.Stat(filepath.Join(dir, "rewrite.md")); !os.IsNotExist(err) {
		t.Error("the old active override file should be removed after migration")
	}
	if _, err := os.Stat(filepath.Join(dir, "versions.json")); err != nil {
		t.Error("migration should persist versions.json so it only runs once")
	}
}

func TestPromptStoreMigrationSkipsUnparseableArchives(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rewrite.v1.md"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rewrite.v2.md"), []byte(validPrompt), 0o644); err != nil {
		t.Fatal(err)
	}

	s := PromptStore{Dir: dir}
	versions := s.List()
	if len(versions) != 2 { // original + the one valid archive
		t.Errorf("the broken archive should be skipped, not block migration: %+v", versions)
	}
}
