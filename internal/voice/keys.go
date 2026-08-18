package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

// helperTimeout bounds the apiKeyHelper subprocess. It only has to print a
// string; anything slower is a hung script, and hanging here would stall the
// request behind it.
const helperTimeout = 5 * time.Second

// ProfileReader fetches a Claude Code profile by name. The server satisfies it
// with the agent's existing "profile-content" command, which is why reusing a
// profile needs no new agent command and no agent redeploy.
type ProfileReader interface {
	ProfileContent(name string) (string, error)
}

// resolved is a provider with its credential filled in.
type resolved struct {
	baseURL string
	apiKey  string
}

// claudeProfile is the subset of a Claude Code settings.json this needs. The
// same shape CCSM's profile viewer already reads.
type claudeProfile struct {
	APIKeyHelper string            `json:"apiKeyHelper"`
	Env          map[string]string `json:"env"`
}

// resolveProvider turns a configured provider into a usable base URL and key.
//
// Exactly one key source is set (config.Validate enforces it at startup), so
// this is a straight dispatch rather than a precedence chain — no silent
// "which one won?" ambiguity.
//
// Nothing here is cached. Running the helper costs a fork of a few
// milliseconds, and paying it per request means rotating the key is just
// editing the script: no restart, no stale credential.
func resolveProvider(p config.VoiceProvider, profiles ProfileReader) (resolved, error) {
	r := resolved{baseURL: strings.TrimRight(p.BaseURL, "/")}

	switch {
	case p.APIKey != "":
		r.apiKey = p.APIKey

	case p.APIKeyEnv != "":
		r.apiKey = os.Getenv(p.APIKeyEnv)
		if r.apiKey == "" {
			return resolved{}, fmt.Errorf("environment variable %s is empty", p.APIKeyEnv)
		}

	case p.APIKeyHelper != "":
		key, err := runKeyHelper(p.APIKeyHelper)
		if err != nil {
			return resolved{}, err
		}
		r.apiKey = key

	case p.FromProfile != "":
		if profiles == nil {
			return resolved{}, fmt.Errorf("cannot read profiles in this deployment mode")
		}
		raw, err := profiles.ProfileContent(p.FromProfile)
		if err != nil {
			return resolved{}, fmt.Errorf("read profile %s: %w", p.FromProfile, err)
		}
		var prof claudeProfile
		if err := json.Unmarshal([]byte(raw), &prof); err != nil {
			return resolved{}, fmt.Errorf("profile %s is not valid JSON", p.FromProfile)
		}
		if prof.APIKeyHelper == "" {
			return resolved{}, fmt.Errorf("profile %s declares no apiKeyHelper", p.FromProfile)
		}
		key, err := runKeyHelper(os.ExpandEnv(prof.APIKeyHelper))
		if err != nil {
			return resolved{}, err
		}
		r.apiKey = key
		// An explicit base_url in the provider wins; otherwise take the one the
		// profile already points Claude Code at.
		if r.baseURL == "" {
			r.baseURL = strings.TrimRight(prof.Env["ANTHROPIC_BASE_URL"], "/")
		}

	default:
		return resolved{}, fmt.Errorf("no key source configured")
	}

	if r.baseURL == "" {
		return resolved{}, fmt.Errorf("no base_url configured")
	}
	return r, nil
}

// runKeyHelper executes a script and takes its stdout as the key — the same
// contract as Claude Code's own apiKeyHelper, so an existing helper can be
// pointed at directly instead of copying the key into CCSM's config.
//
// The helper's output is a secret: it is never logged, never wrapped into an
// error message, and never returned to a client. Errors here say only that the
// helper failed.
func runKeyHelper(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("api_key_helper is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, path).Output()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("api_key_helper timed out")
	}
	if err != nil {
		// Deliberately not %w and not including stderr: a helper that echoes
		// the key in a failure message must not leak it into the API response.
		return "", fmt.Errorf("api_key_helper failed")
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		return "", fmt.Errorf("api_key_helper produced no key")
	}
	return key, nil
}
