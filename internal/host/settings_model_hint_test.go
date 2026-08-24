package host

import (
	"os"
	"testing"
)

func tmpSettings(t *testing.T, h *Host, content string) {
	t.Helper()
	if err := os.WriteFile(h.settingsPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsModelHint(t *testing.T) {
	deepseek := `{
	  "model": "sonnet",
	  "env": {
	    "ANTHROPIC_BASE_URL": "https://api.deepseek.com/anthropic",
	    "ANTHROPIC_DEFAULT_OPUS_MODEL": "deepseek-v4-pro[1m]",
	    "ANTHROPIC_DEFAULT_SONNET_MODEL": "deepseek-v4-flash[1m]",
	    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "deepseek-v4-flash[1m]",
	    "ANTHROPIC_SMALL_FAST_MODEL": "deepseek-v4-flash[1m]",
	    "CLAUDE_CODE_SUBAGENT_MODEL": "deepseek-v4-flash[1m]"
	  }
	}`
	estandar := `{"model":"sonnet","tui":"fullscreen"}`
	direct := `{"model":"deepseek-v4-flash[1m]"}`
	noTier := `{"env":{"ANTHROPIC_DEFAULT_OPUS_MODEL":"custom-opus","ANTHROPIC_DEFAULT_SONNET_MODEL":"custom-sonnet"}}`

	cases := []struct {
		name string
		json string
		want string
	}{
		{"deepseek-tier-sonnet", deepseek, "deepseek-v4-flash[1m]"},
		{"estandar-sonnet", estandar, "sonnet"},
		{"direct-model-id", direct, "deepseek-v4-flash[1m]"},
		{"no-tier-prefers-sonnet", noTier, "custom-sonnet"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, _, _ := newTestHost(t)
			tmpSettings(t, h, c.json)
			if got := h.settingsModelHint(); got != c.want {
				t.Fatalf("settingsModelHint() = %q, want %q", got, c.want)
			}
		})
	}

	t.Run("missing-file", func(t *testing.T) {
		h, _, _ := newTestHost(t) // settings.json doesn't exist
		if got := h.settingsModelHint(); got != "" {
			t.Fatalf("settingsModelHint() = %q, want \"\"", got)
		}
	})

	t.Run("unparseable", func(t *testing.T) {
		h, _, _ := newTestHost(t)
		tmpSettings(t, h, "{not json")
		if got := h.settingsModelHint(); got != "" {
			t.Fatalf("settingsModelHint() = %q, want \"\"", got)
		}
	})

}

func TestPaneModelHint(t *testing.T) {
	// Without a real tmux pane, paneModelHint should still fall back to the
	// settings when there is no capture output (capture fails → returns "").
	h, _, _ := newTestHost(t)
	tmpSettings(t, h, `{"model":"sonnet","env":{"ANTHROPIC_DEFAULT_SONNET_MODEL":"deepseek-v4-flash[1m]"}}`)
	// h.tmuxBinary is empty in the test host → capture-pane fails → hint = settings.
	if got := h.paneModelHint("any"); got != "deepseek-v4-flash[1m]" {
		t.Fatalf("paneModelHint = %q, want deepseek-v4-flash[1m]", got)
	}
}
