package voice

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed prompts/rewrite.md
var embeddedPrompts embed.FS

// promptFile is the active meta-prompt on disk; anything matching
// promptVersionRe next to it is an archived version.
const promptFile = "rewrite.md"

var promptVersionRe = regexp.MustCompile(`^rewrite\.v(\d+)\.md$`)

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
// and one block per role.
type Prompt struct {
	Roles  []Role
	Base   string
	Blocks map[string]string
	Raw    string
}

type promptFrontMatter struct {
	Roles []Role `yaml:"roles"`
}

var (
	frontMatterRe = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n?`)
	sectionRe     = regexp.MustCompile(`(?m)^#[ \t]+(Base|Role:[ \t]*[A-Za-z0-9_-]+)[ \t]*$`)
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

// System assembles the system prompt for one rewrite.
//
// A forced role gets Base plus its own block, and nothing about the other
// roles. AutoRole gets Base plus every block, because the model cannot choose
// between roles it has not been shown — which is exactly why splitting the
// meta-prompt into one file per role would have saved nothing in the mode that
// is used by default.
func (p *Prompt) System(role string, maxQuestions int) string {
	var b strings.Builder
	b.WriteString(strings.ReplaceAll(p.Base, "MAX_QUESTIONS", strconv.Itoa(maxQuestions)))

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
		return b.String()
	}
	writeBlock(role)
	return b.String()
}

// Original returns the meta-prompt shipped inside the binary. It is what
// "restore original" restores, and it can never be edited away: an override
// lives beside it on disk rather than replacing it.
func Original() string {
	data, err := embeddedPrompts.ReadFile("prompts/" + promptFile)
	if err != nil {
		// Unreachable: go:embed fails the build if the file is missing.
		panic("voice: embedded prompt missing: " + err.Error())
	}
	return string(data)
}

// PromptStore reads and writes the meta-prompt override plus its version
// history. An empty dir means "no override": Load falls back to the embedded
// original, so a fresh install works with no files at all.
type PromptStore struct{ Dir string }

// Load returns the active meta-prompt text and whether it came from an
// override rather than the embedded original.
func (s PromptStore) Load() (string, bool) {
	if s.Dir == "" {
		return Original(), false
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, promptFile))
	if err != nil || len(data) == 0 {
		return Original(), false
	}
	return string(data), true
}

// Active parses whatever Load returns, falling back to the original if the
// override on disk is somehow unparseable.
//
// Save refuses to write anything invalid, so this should not happen — but a
// file edited by hand outside CCSM must degrade to a working default rather
// than take the dictation button down with it.
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

// Versions lists archived versions, newest first.
func (s PromptStore) Versions() []int {
	if s.Dir == "" {
		return nil
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range entries {
		if m := promptVersionRe.FindStringSubmatch(e.Name()); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				out = append(out, n)
			}
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

// Version returns the text of one archived version.
func (s PromptStore) Version(n int) (string, error) {
	if s.Dir == "" {
		return "", fmt.Errorf("no prompts directory configured")
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, fmt.Sprintf("rewrite.v%d.md", n)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Save validates the new text, archives the current one, then writes.
//
// Order matters: parsing first means a broken edit leaves the previous
// meta-prompt in place and untouched, rather than archiving it and then
// failing with nothing active.
func (s PromptStore) Save(raw string) error {
	if s.Dir == "" {
		return fmt.Errorf("no prompts directory configured")
	}
	if _, err := ParsePrompt(raw); err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create prompts dir: %w", err)
	}

	current := filepath.Join(s.Dir, promptFile)
	if existing, err := os.ReadFile(current); err == nil && len(existing) > 0 {
		next := 1
		if vs := s.Versions(); len(vs) > 0 {
			next = vs[0] + 1
		}
		archive := filepath.Join(s.Dir, fmt.Sprintf("rewrite.v%d.md", next))
		if err := os.WriteFile(archive, existing, 0o644); err != nil {
			return fmt.Errorf("archive current version: %w", err)
		}
	}

	// temp + rename, the same atomic write the config uses, so a crash
	// mid-write cannot leave a half-written prompt as the active one.
	tmp := current + ".tmp"
	if err := os.WriteFile(tmp, []byte(raw), 0o644); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}
	if err := os.Rename(tmp, current); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace prompt: %w", err)
	}
	return nil
}

// Reset removes the override so the embedded original applies again. The
// archived versions stay: restoring the default should not throw away the
// history of what was tried.
func (s PromptStore) Reset() error {
	if s.Dir == "" {
		return fmt.Errorf("no prompts directory configured")
	}
	err := os.Remove(filepath.Join(s.Dir, promptFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
