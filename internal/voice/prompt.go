package voice

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed prompts/rewrite.md
var embeddedPrompts embed.FS

// promptFile and promptVersionRe are the on-disk layout of the OLD version
// scheme (a single active "rewrite.md" plus auto-numbered "rewrite.vN.md"
// archives with no names and no explicit active pointer). They are kept only
// so migrateLegacy can read a directory written by that scheme; nothing
// current writes promptFile any more.
const promptFile = "rewrite.md"

var promptVersionRe = regexp.MustCompile(`^rewrite\.v(\d+)\.md$`)

// metaFile is where the current scheme's version list and active pointer
// live; the versions themselves are still "rewrite.v<id>.md" beside it, so a
// migrated directory does not need its content files renamed.
const metaFile = "versions.json"

// AutoRole is the role id that asks the model to classify the request itself.
// It is the only role with no "# Role:" block of its own to validate against —
// its block exists, but it describes how to choose, not what to do.
const AutoRole = "auto"

// Role is one selectable rewriting persona, as declared in the meta-prompt's
// front matter. Labels are per language because they are what the UI paints in
// the dropdown; the id is what travels over the API.
type Role struct {
	ID string `yaml:"id" json:"id"`
	ES string `yaml:"es" json:"es"`
	EN string `yaml:"en" json:"en"`
}

// Prompt is a parsed meta-prompt: the declared roles, the shared Base section,
// one block per role, and an optional Closing section.
//
// Closing exists because of where System() puts things. Base comes first and
// the role blocks after it, so anything written at the end of Base is not
// what the model reads last — several thousand characters of role
// instructions are. The output contract and the pre-flight check belong at
// the end, and belong in the meta-prompt file rather than hardcoded here:
// the file is user-editable and versioned, so a custom version that changes
// the output format has to be able to change its own closing reminder too.
// It is optional: a prompt with no "# Closing" section assembles exactly as
// it did before the section existed.
type Prompt struct {
	Roles   []Role
	Base    string
	Blocks  map[string]string
	Closing string
	Raw     string
}

type promptFrontMatter struct {
	Roles []Role `yaml:"roles"`
}

var (
	frontMatterRe = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n?`)
	sectionRe     = regexp.MustCompile(`(?m)^#[ \t]+(Base|Closing|Role:[ \t]*[A-Za-z0-9_-]+)[ \t]*$`)
)

// ParsePrompt reads a meta-prompt and checks it is internally consistent.
//
// The consistency check is the point: a role declared in the front matter with
// no block behind it would appear in the dropdown and then reach the model with
// no instructions at all, producing quietly worse rewrites with nothing to
// point at. Both directions are errors.
func ParsePrompt(raw string) (*Prompt, error) {
	m := frontMatterRe.FindStringSubmatch(raw)
	if m == nil {
		return nil, fmt.Errorf("missing YAML front matter delimited by ---")
	}
	var fm promptFrontMatter
	if err := yaml.Unmarshal([]byte(m[1]), &fm); err != nil {
		return nil, fmt.Errorf("invalid front matter: %w", err)
	}
	if len(fm.Roles) == 0 {
		return nil, fmt.Errorf("front matter declares no roles")
	}

	seen := map[string]bool{}
	validID := regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	for _, r := range fm.Roles {
		if !validID.MatchString(r.ID) {
			return nil, fmt.Errorf("invalid role id %q", r.ID)
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("duplicate role id %q", r.ID)
		}
		seen[r.ID] = true
		if r.ES == "" || r.EN == "" {
			return nil, fmt.Errorf("role %q needs both es and en labels", r.ID)
		}
	}

	body := raw[len(m[0]):]
	p := &Prompt{Roles: fm.Roles, Blocks: map[string]string{}, Raw: raw}

	idx := sectionRe.FindAllStringSubmatchIndex(body, -1)
	if len(idx) == 0 {
		return nil, fmt.Errorf("no '# Base' section found")
	}
	for i, loc := range idx {
		heading := body[loc[2]:loc[3]]
		end := len(body)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		content := strings.TrimSpace(body[loc[1]:end])
		if strings.EqualFold(heading, "Base") {
			if p.Base != "" {
				return nil, fmt.Errorf("duplicate '# Base' section")
			}
			p.Base = content
			continue
		}
		if strings.EqualFold(heading, "Closing") {
			if p.Closing != "" {
				return nil, fmt.Errorf("duplicate '# Closing' section")
			}
			if content == "" {
				return nil, fmt.Errorf("the '# Closing' section is empty")
			}
			p.Closing = content
			continue
		}
		id := strings.TrimSpace(strings.TrimPrefix(heading, "Role:"))
		if _, dup := p.Blocks[id]; dup {
			return nil, fmt.Errorf("duplicate '# Role: %s' section", id)
		}
		p.Blocks[id] = content
	}

	if p.Base == "" {
		return nil, fmt.Errorf("the '# Base' section is empty")
	}
	for _, r := range fm.Roles {
		if _, ok := p.Blocks[r.ID]; !ok {
			return nil, fmt.Errorf("role %q is declared but has no '# Role: %s' section", r.ID, r.ID)
		}
	}
	for id := range p.Blocks {
		if !seen[id] {
			return nil, fmt.Errorf("section '# Role: %s' is not declared in the front matter", id)
		}
	}
	return p, nil
}

// HasRole reports whether id is one of the declared roles.
func (p *Prompt) HasRole(id string) bool {
	for _, r := range p.Roles {
		if r.ID == id {
			return true
		}
	}
	return false
}

// RoleIDs returns the declared role ids in declaration order.
func (p *Prompt) RoleIDs() []string {
	ids := make([]string, 0, len(p.Roles))
	for _, r := range p.Roles {
		ids = append(ids, r.ID)
	}
	return ids
}

// System assembles the system prompt for one rewrite: Base, then the role
// block(s), then Closing.
//
// A forced role gets Base plus its own block, and nothing about the other
// roles. AutoRole gets Base plus every block, because the model cannot choose
// between roles it has not been shown — which is exactly why splitting the
// meta-prompt into one file per role would have saved nothing in the mode that
// is used by default.
//
// extras are the per-call sections Rewrite adds on top of the file: the
// vocabulary list, and the clarification-round notes. They go after the role
// blocks but BEFORE Closing, because Closing is the output contract and the
// point of having it is that it is read last — appending anything after it,
// least of all a vocabulary list that is user-configured and can be long,
// would put the contract back in the middle where it started. Each extra is
// expected to be a self-contained "# Heading\n\nbody" chunk; empty ones are
// skipped.
//
// Closing therefore goes last of all, so the output contract and the
// pre-flight check are the final thing the model reads rather than being
// buried mid-prompt behind the role instructions.
func (p *Prompt) System(role string, extras ...string) string {
	var b strings.Builder
	b.WriteString(p.Base)

	writeBlock := func(id string) {
		block, ok := p.Blocks[id]
		if !ok {
			return
		}
		b.WriteString("\n\n# Role: ")
		b.WriteString(id)
		b.WriteString("\n\n")
		b.WriteString(block)
	}

	if role == AutoRole || !p.HasRole(role) {
		for _, r := range p.Roles {
			writeBlock(r.ID)
		}
	} else {
		writeBlock(role)
	}

	for _, e := range extras {
		if e = strings.TrimSpace(e); e != "" {
			b.WriteString("\n\n")
			b.WriteString(e)
		}
	}

	if p.Closing != "" {
		b.WriteString("\n\n# Closing\n\n")
		b.WriteString(p.Closing)
	}
	return b.String()
}

// Original returns the meta-prompt shipped inside the binary — version id 0,
// always available to view or apply, and it can never be edited away: a
// saved version lives beside it on disk rather than replacing it.
func Original() string {
	data, err := embeddedPrompts.ReadFile("prompts/" + promptFile)
	if err != nil {
		// Unreachable: go:embed fails the build if the file is missing.
		panic("voice: embedded prompt missing: " + err.Error())
	}
	return string(data)
}

// PromptStore reads and writes the meta-prompt's named versions plus which
// one is active. An empty dir means "nothing saved yet": Load falls back to
// the embedded original, so a fresh install works with no files at all.
//
// Version id 0 always means the embedded original: it is never stored on
// disk, never appears in versionRec, and SaveOver refuses it — "the original
// cannot be modified" is enforced here, not just in the UI.
// Everything else is a versionRec{ID, Name} plus its content in
// "rewrite.v<id>.md", with the active id recorded in versions.json so that
// which version is in effect is independent of which one is being edited.
type PromptStore struct{ Dir string }

type versionRec struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type storeMeta struct {
	Active   int          `json:"active"` // 0 = original
	NextID   int          `json:"next_id"`
	Versions []versionRec `json:"versions"`
}

// VersionInfo is one entry in the version list the editor's dropdown renders:
// enough to label the option and check it as active, without exposing the
// content (fetched separately, only when that version is opened for viewing).
type VersionInfo struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Original  bool      `json:"original"`
	Active    bool      `json:"active"`
}

func (s PromptStore) versionPath(id int) string {
	return filepath.Join(s.Dir, fmt.Sprintf("rewrite.v%d.md", id))
}

func (s PromptStore) metaPath() string {
	return filepath.Join(s.Dir, metaFile)
}

// writeFileAtomic is the temp+rename write every mutation below uses, so a
// crash mid-write can never leave a half-written version or meta file active.
func writeFileAtomic(path, content string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return nil
}

// readMeta loads versions.json, migrating an old-scheme directory into it the
// first time it is read. It fails open (an empty, "original is active" store)
// rather than error, matching Load's existing degrade-gracefully contract —
// a directory CCSM cannot write to should still let dictation work with the
// original prompt.
func (s PromptStore) readMeta() storeMeta {
	if s.Dir == "" {
		return storeMeta{NextID: 1}
	}
	data, err := os.ReadFile(s.metaPath())
	if err != nil {
		m := s.migrateLegacy()
		_ = s.writeMeta(m)
		return m
	}
	var m storeMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return storeMeta{NextID: 1}
	}
	if m.NextID == 0 {
		m.NextID = 1
	}
	return m
}

func (s PromptStore) writeMeta(m storeMeta) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create prompts dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.metaPath(), string(data))
}

// migrateLegacy reads a directory left by the old scheme (see promptFile,
// promptVersionRe above) and turns it into a storeMeta, without deleting or
// renumbering anything a previous CCSM wrote:
//   - each old "rewrite.vN.md" archive becomes an inactive named version,
//     numbered in the new scheme starting at 1 in the old files' own order —
//     which happens to keep every id the same, since the old scheme was
//     already gap-free starting at 1;
//   - an old "rewrite.md" (the old scheme's one-and-only active override, if
//     any edit had actually been saved) becomes the newest version AND the
//     active one, then is removed — content now lives only in its
//     "rewrite.v<id>.md" copy, as the new scheme expects.
//
// A file that fails to parse is skipped rather than aborting the whole
// migration: better to lose one broken archive than to make the editor
// unusable because of it.
func (s PromptStore) migrateLegacy() storeMeta {
	m := storeMeta{NextID: 1}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return m
	}

	type legacy struct {
		n    int
		path string
	}
	var legacies []legacy
	for _, e := range entries {
		if mm := promptVersionRe.FindStringSubmatch(e.Name()); mm != nil {
			if n, err := strconv.Atoi(mm[1]); err == nil {
				legacies = append(legacies, legacy{n, filepath.Join(s.Dir, e.Name())})
			}
		}
	}
	sort.Slice(legacies, func(i, j int) bool { return legacies[i].n < legacies[j].n })

	for _, lg := range legacies {
		data, err := os.ReadFile(lg.path)
		if err != nil || len(data) == 0 {
			continue
		}
		if _, err := ParsePrompt(string(data)); err != nil {
			continue
		}
		id := m.NextID
		m.NextID++
		newPath := s.versionPath(id)
		if newPath != lg.path {
			if err := os.Rename(lg.path, newPath); err != nil {
				continue
			}
		}
		m.Versions = append(m.Versions, versionRec{
			ID: id, Name: fmt.Sprintf("Versión %d (migrada)", lg.n), CreatedAt: time.Now(),
		})
	}

	activePath := filepath.Join(s.Dir, promptFile)
	if data, err := os.ReadFile(activePath); err == nil && len(data) > 0 {
		if _, err := ParsePrompt(string(data)); err == nil {
			id := m.NextID
			if err := writeFileAtomic(s.versionPath(id), string(data)); err == nil {
				m.NextID++
				m.Versions = append(m.Versions, versionRec{
					ID: id, Name: "Migrada del editor anterior", CreatedAt: time.Now(),
				})
				m.Active = id
			}
		}
		_ = os.Remove(activePath)
	}
	return m
}

// Load returns the active meta-prompt text and whether it came from a saved
// version rather than the embedded original.
func (s PromptStore) Load() (string, bool) {
	if s.Dir == "" {
		return Original(), false
	}
	m := s.readMeta()
	if m.Active == 0 {
		return Original(), false
	}
	data, err := os.ReadFile(s.versionPath(m.Active))
	if err != nil || len(data) == 0 {
		return Original(), false
	}
	return string(data), true
}

// Active parses whatever Load returns, falling back to the original if the
// active version on disk is somehow unparseable.
//
// SaveOver and SaveNew refuse to write anything invalid, so this should not
// happen — but a file edited by hand outside CCSM must degrade to a working
// default rather than take the dictation button down with it.
func (s PromptStore) Active() (*Prompt, error) {
	raw, custom := s.Load()
	p, err := ParsePrompt(raw)
	if err == nil {
		return p, nil
	}
	if !custom {
		return nil, err
	}
	return ParsePrompt(Original())
}

// List returns every version the editor can show, the embedded original
// first, in creation order after that — with Active telling the UI which one
// a check mark belongs on.
func (s PromptStore) List() []VersionInfo {
	m := s.readMeta()
	out := []VersionInfo{{Original: true, Active: m.Active == 0}}
	for _, v := range m.Versions {
		out = append(out, VersionInfo{
			ID: v.ID, Name: v.Name, CreatedAt: v.CreatedAt, Active: v.ID == m.Active,
		})
	}
	return out
}

func (s PromptStore) has(m storeMeta, id int) bool {
	for _, v := range m.Versions {
		if v.ID == id {
			return true
		}
	}
	return false
}

// VersionContent returns the text of one version; id 0 is always the
// embedded original.
func (s PromptStore) VersionContent(id int) (string, error) {
	if id == 0 {
		return Original(), nil
	}
	if s.Dir == "" || !s.has(s.readMeta(), id) {
		return "", fmt.Errorf("version not found")
	}
	data, err := os.ReadFile(s.versionPath(id))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveNew validates content and stores it as a brand-new version, leaving
// whichever version is currently active untouched — applying it is a
// separate, explicit step (SetActive), not a side effect of saving.
func (s PromptStore) SaveNew(content, name string) (int, error) {
	if s.Dir == "" {
		return 0, fmt.Errorf("no prompts directory configured")
	}
	if _, err := ParsePrompt(content); err != nil {
		return 0, err
	}
	m := s.readMeta()
	id := m.NextID
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("Versión %d", id)
	}
	if err := writeFileAtomic(s.versionPath(id), content); err != nil {
		return 0, err
	}
	m.Versions = append(m.Versions, versionRec{ID: id, Name: name, CreatedAt: time.Now()})
	m.NextID = id + 1
	if err := s.writeMeta(m); err != nil {
		return 0, err
	}
	return id, nil
}

// SaveOver overwrites an existing version's content in place, keeping its
// name and id. The embedded original (id 0) is never a valid target: editing
// it always goes through SaveNew instead.
func (s PromptStore) SaveOver(id int, content string) error {
	if id == 0 {
		return fmt.Errorf("the original prompt cannot be modified; save it as a new version instead")
	}
	if s.Dir == "" {
		return fmt.Errorf("no prompts directory configured")
	}
	if _, err := ParsePrompt(content); err != nil {
		return err
	}
	if !s.has(s.readMeta(), id) {
		return fmt.Errorf("version not found")
	}
	return writeFileAtomic(s.versionPath(id), content)
}

// Rename changes an existing version's label without touching its content or
// id. The embedded original (id 0) has no stored name to change — it is
// always labelled from the UI's own translation string, not versionRec.Name.
func (s PromptStore) Rename(id int, name string) error {
	if id == 0 {
		return fmt.Errorf("the original prompt has no name to change")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name must not be empty")
	}
	if s.Dir == "" {
		return fmt.Errorf("no prompts directory configured")
	}
	m := s.readMeta()
	found := false
	for i := range m.Versions {
		if m.Versions[i].ID == id {
			m.Versions[i].Name = name
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version not found")
	}
	return s.writeMeta(m)
}

// Delete removes a saved version permanently, content and all. The embedded
// original (id 0) can never be deleted — there being nothing left to fall
// back on would take dictation down with it. Deleting the active version
// falls back to the original (id 0), the same safe default SetActive(0)
// produces, rather than leaving versions.json pointing at a version that no
// longer exists.
func (s PromptStore) Delete(id int) error {
	if id == 0 {
		return fmt.Errorf("the original prompt cannot be deleted")
	}
	if s.Dir == "" {
		return fmt.Errorf("no prompts directory configured")
	}
	m := s.readMeta()
	idx := -1
	for i, v := range m.Versions {
		if v.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("version not found")
	}
	m.Versions = append(m.Versions[:idx], m.Versions[idx+1:]...)
	if m.Active == id {
		m.Active = 0
	}
	if err := s.writeMeta(m); err != nil {
		return err
	}
	// Best-effort: versions.json (just written) is the source of truth for
	// List/VersionContent, so a leftover file here is inert, not a leak that
	// resurfaces — same trade-off migrateLegacy makes removing the old
	// active-override file.
	_ = os.Remove(s.versionPath(id))
	return nil
}

// SetActive makes version id the one Load and Active return; 0 applies the
// embedded original. This never touches any version's content — it only
// moves the pointer, so applying a version is always non-destructive and
// every version stays available afterward.
func (s PromptStore) SetActive(id int) error {
	if id == 0 {
		if s.Dir == "" {
			return nil
		}
		m := s.readMeta()
		if m.Active == 0 {
			return nil
		}
		m.Active = 0
		return s.writeMeta(m)
	}
	if s.Dir == "" {
		return fmt.Errorf("no prompts directory configured")
	}
	m := s.readMeta()
	if !s.has(m, id) {
		return fmt.Errorf("version not found")
	}
	m.Active = id
	return s.writeMeta(m)
}
