package voice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

// fakeProfiles satisfies ProfileReader without an agent.
type fakeProfiles struct {
	content map[string]string
	err     error
}

func (f fakeProfiles) ProfileContent(name string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	c, ok := f.content[name]
	if !ok {
		return "", fmt.Errorf("no such profile")
	}
	return c, nil
}

// writeHelper drops an executable script that prints what a real apiKeyHelper
// would print.
func writeHelper(t *testing.T, script string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveProviderSources(t *testing.T) {
	t.Run("api_key", func(t *testing.T) {
		r, err := resolveProvider(config.VoiceProvider{
			BaseURL: "https://api.example/v1/",
			APIKey:  "sk-literal",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if r.apiKey != "sk-literal" {
			t.Errorf("key = %q", r.apiKey)
		}
		// The trailing slash must go, or every URL becomes /v1//chat/completions.
		if r.baseURL != "https://api.example/v1" {
			t.Errorf("base URL not normalised: %q", r.baseURL)
		}
	})

	t.Run("api_key_env", func(t *testing.T) {
		t.Setenv("CCSM_TEST_KEY", "sk-from-env")
		r, err := resolveProvider(config.VoiceProvider{
			BaseURL: "https://api.example", APIKeyEnv: "CCSM_TEST_KEY",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if r.apiKey != "sk-from-env" {
			t.Errorf("key = %q", r.apiKey)
		}
	})

	t.Run("api_key_env empty", func(t *testing.T) {
		t.Setenv("CCSM_TEST_KEY", "")
		_, err := resolveProvider(config.VoiceProvider{
			BaseURL: "https://api.example", APIKeyEnv: "CCSM_TEST_KEY",
		}, nil)
		if err == nil {
			t.Fatal("an unset variable must be an error, not an empty key")
		}
	})

	t.Run("api_key_helper", func(t *testing.T) {
		// Trailing newline included on purpose: that is what a real helper
		// script emits, and it must be trimmed rather than sent in the header.
		h := writeHelper(t, `printf '%s\n' 'sk-from-helper'`)
		r, err := resolveProvider(config.VoiceProvider{
			BaseURL: "https://api.example", APIKeyHelper: h,
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if r.apiKey != "sk-from-helper" {
			t.Errorf("key = %q (newline not trimmed?)", r.apiKey)
		}
	})

	t.Run("from_profile", func(t *testing.T) {
		h := writeHelper(t, `printf '%s' 'sk-profile'`)
		profile := fmt.Sprintf(`{"apiKeyHelper": %q, "env": {"ANTHROPIC_BASE_URL": "https://deepseek.example/anthropic"}}`, h)
		r, err := resolveProvider(config.VoiceProvider{
			FromProfile: "deepseek",
		}, fakeProfiles{content: map[string]string{"deepseek": profile}})
		if err != nil {
			t.Fatal(err)
		}
		if r.apiKey != "sk-profile" {
			t.Errorf("key = %q", r.apiKey)
		}
		if r.baseURL != "https://deepseek.example/anthropic" {
			t.Errorf("base URL should come from the profile, got %q", r.baseURL)
		}
	})

	t.Run("explicit base_url beats the profile", func(t *testing.T) {
		h := writeHelper(t, `printf '%s' 'sk-profile'`)
		profile := fmt.Sprintf(`{"apiKeyHelper": %q, "env": {"ANTHROPIC_BASE_URL": "https://from-profile"}}`, h)
		r, err := resolveProvider(config.VoiceProvider{
			BaseURL: "https://explicit", FromProfile: "deepseek",
		}, fakeProfiles{content: map[string]string{"deepseek": profile}})
		if err != nil {
			t.Fatal(err)
		}
		if r.baseURL != "https://explicit" {
			t.Errorf("explicit base_url should win, got %q", r.baseURL)
		}
	})
}

func TestResolveProviderFailures(t *testing.T) {
	cases := []struct {
		name     string
		provider config.VoiceProvider
		profiles ProfileReader
		want     string
	}{
		{"no source", config.VoiceProvider{BaseURL: "https://x"}, nil, "no key source"},
		{"no base url", config.VoiceProvider{APIKey: "k"}, nil, "no base_url"},
		{
			"helper that fails",
			config.VoiceProvider{BaseURL: "https://x", APIKeyHelper: "/nonexistent/helper"},
			nil, "api_key_helper failed",
		},
		{
			"profile without a reader",
			config.VoiceProvider{FromProfile: "deepseek"}, nil, "cannot read profiles",
		},
		{
			"profile that does not exist",
			config.VoiceProvider{FromProfile: "ghost"},
			fakeProfiles{content: map[string]string{}}, "read profile ghost",
		},
		{
			"profile that is not JSON",
			config.VoiceProvider{FromProfile: "bad"},
			fakeProfiles{content: map[string]string{"bad": "not json"}}, "not valid JSON",
		},
		{
			"profile with no helper",
			config.VoiceProvider{BaseURL: "https://x", FromProfile: "plain"},
			fakeProfiles{content: map[string]string{"plain": `{"model":"sonnet"}`}}, "no apiKeyHelper",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := resolveProvider(c.provider, c.profiles)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not mention %q", err, c.want)
			}
		})
	}
}

// TestKeyHelperEmptyOutput: a helper that succeeds but prints nothing would
// otherwise authenticate with an empty bearer token and fail confusingly at
// the provider.
func TestKeyHelperEmptyOutput(t *testing.T) {
	h := writeHelper(t, "exit 0")
	if _, err := runKeyHelper(h); err == nil || !strings.Contains(err.Error(), "no key") {
		t.Errorf("expected an empty-output error, got %v", err)
	}
}

// TestKeyHelperTimesOut: a hung helper must not hold the request open.
func TestKeyHelperTimesOut(t *testing.T) {
	if testing.Short() {
		t.Skip("takes helperTimeout seconds")
	}
	h := writeHelper(t, "sleep 30")
	_, err := runKeyHelper(h)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected a timeout, got %v", err)
	}
}

// TestKeyHelperNeverLeaksTheKey: helpers do print secrets, and a failing one
// may print one to stderr or on stdout before exiting non-zero. None of that
// may reach an error string, because error strings reach the HTTP response.
func TestKeyHelperNeverLeaksTheKey(t *testing.T) {
	const secret = "sk-super-secret-value"
	h := writeHelper(t, "printf '%s' '"+secret+"'; echo '"+secret+"' >&2; exit 3")

	_, err := runKeyHelper(h)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("the key leaked into the error message: %v", err)
	}
}
