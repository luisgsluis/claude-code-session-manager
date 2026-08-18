package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecInjectionRejected proves that every user-input command rejects hostile
// payloads (shell metacharacters, command substitution, quotes, spaces, path
// traversal, newlines) with a 400 BEFORE any command can run. A hostile value
// must never reach exec.Command.
func TestExecInjectionRejected(t *testing.T) {
	h := fakeHost(t, nil)

	hostile := []string{
		"x; rm -rf /",
		"$(id)",
		"`id`",
		"x & id",
		"x|id",
		"x'id",
		`x"id`,
		"x y",
		"../etc/passwd",
		"a/../b",
		"x\nrm -rf /",
		"x$PATH",
		">/tmp/pwned",
	}

	// Titles allow punctuation by design (\p{L}\p{N}\p{P} and space), so only
	// characters OUTSIDE that set — and newlines — are rejectable.
	hostileTitle := []string{
		"x$(id)",
		"x`id`",
		"x|id",
		"x>y",
		"x<y",
		"x=y",
		"x+y",
		"x~y",
		"x^y",
		"x\nrm -rf /",
	}

	cases := []struct {
		cmd      string
		args     func(p string) map[string]string
		msg      string
		payloads []string
	}{
		{"tmux-kill", func(p string) map[string]string { return map[string]string{"name": p} }, "invalid session name", nil},
		{"tmux-rename", func(p string) map[string]string { return map[string]string{"name": p, "new_name": "safe1"} }, "invalid session name", nil},
		{"claude-rename", func(p string) map[string]string { return map[string]string{"session": p, "title": "ok"} }, "invalid session name", nil},
		{"claude-rename", func(p string) map[string]string { return map[string]string{"session": "x", "title": p} }, "invalid title", hostileTitle},
		{"claude-nueva", func(p string) map[string]string { return map[string]string{"profile": p} }, "invalid profile name", nil},
		{"claude-resume", func(p string) map[string]string { return map[string]string{"id": p} }, "invalid conversation id", nil},
		{"claude-perfil", func(p string) map[string]string { return map[string]string{"profile": p} }, "invalid profile name", nil},
		{"profile-content", func(p string) map[string]string { return map[string]string{"name": p} }, "invalid profile name", nil},
		{"conversation-get", func(p string) map[string]string { return map[string]string{"id": p} }, "invalid conversation id", nil},
	}

	for _, c := range cases {
		pl := c.payloads
		if pl == nil {
			pl = hostile
		}
		for _, p := range pl {
			_, err := h.Exec(c.cmd, c.args(p))
			if status := errStatus(err); status != 400 {
				t.Errorf("%s payload %q: status=%d want 400 (err=%v)", c.cmd, p, status, err)
			}
			if err != nil && !strings.Contains(err.Error(), c.msg) {
				t.Errorf("%s payload %q: message %q not in %q", c.cmd, p, c.msg, err.Error())
			}
		}
	}
}

// TestSessionNameNeutralized proves that session names (creation and rename) are
// normalized to a safe [A-Za-z0-9_-] form before any command runs, and that a
// name with nothing valid left after normalization is rejected with a 400. The
// raw hostile string must never reach exec.Command.
func TestSessionNameNeutralized(t *testing.T) {
	h := fakeHost(t, nil)

	// A hostile name normalizes to a harmless one and proceeds — it must not
	// be rejected as "invalid session name", and the normalized form must be
	// the one the Host acts on (here, the already-in-use check).
	_, err := h.Exec("claude-nueva", map[string]string{"name": "x; rm -rf /"})
	if err == nil {
		t.Fatal("expected an error from claude-nueva, got nil")
	}
	if !strings.Contains(err.Error(), "x-rm-rf") {
		t.Errorf("normalized name not used; error was: %v", err)
	}
	if strings.Contains(err.Error(), "invalid session name") {
		t.Errorf("hostile name was rejected instead of normalized: %v", err)
	}

	// Names with nothing left after normalization are a clean 400.
	for _, name := range []string{"...", "!!!", "   ", "$()", "\n\t"} {
		_, err := h.Exec("claude-nueva", map[string]string{"name": name})
		if status := errStatus(err); status != 400 {
			t.Errorf("name %q: status=%d want 400 (err=%v)", name, status, err)
		}
		if err != nil && !strings.Contains(err.Error(), "invalid session name") {
			t.Errorf("name %q: message %q not in %q", name, "invalid session name", err.Error())
		}
	}
}

// TestClaudeTitlePunctuationTypedLiteral proves that although a Claude title may
// contain shell-dangerous punctuation (by design — the title pattern allows
// ";" and "/" etc.), it is delivered to the tmux pane as literal keystrokes
// (send-keys -l), never parsed by any shell — so it cannot execute anything.
func TestClaudeTitlePunctuationTypedLiteral(t *testing.T) {
	rec := filepath.Join(t.TempDir(), "keys.txt")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_SENDKEYS": rec})

	title := "; rm -rf /"
	_, err := h.Exec("claude-rename", map[string]string{"session": "x", "title": title})
	if err != nil {
		t.Fatalf("claude-rename with punctuation title: %v", err)
	}

	raw, err := os.ReadFile(rec)
	if err != nil {
		t.Fatalf("read keys: %v", err)
	}
	// Don't trim: the trailing Enter is the literal byte \r, which
	// strings.TrimSpace would strip along with the surrounding whitespace.
	got := string(raw)
	if n := strings.Count(got, "\n"); n != 2 {
		t.Fatalf("expected 2 send-keys calls (literal + Enter), got %d: %q", n, raw)
	}
	if !strings.Contains(got, "-l /rename ; rm -rf /") {
		t.Errorf("title not sent as literal -l argument: %q", got)
	}
	// Enter as a raw literal byte (\r), not tmux's named "Enter" key — see
	// pressEnter in host.go.
	if !strings.Contains(got, "-l \r") {
		t.Errorf("expected enter sent as a literal \\r byte, got: %q", got)
	}
}

// TestNewSessionShellStringNoBreakout proves the claude launch string is built
// only from validated parts (profile) and literal flags, is single-quote
// escaped, and reaches tmux as ONE argv element (so the pane shell sees a single
// argument and no second parse can happen). The session name travels as its own
// tmux argument and never enters the shell string.
func TestNewSessionShellStringNoBreakout(t *testing.T) {
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatal(err)
	}
	rec := filepath.Join(base, "argv.txt")

	writeExe := func(name, script string) string {
		p := filepath.Join(binDir, name)
		if err := os.WriteFile(p, []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// Records ONLY the new-session argv (one element per line). has-session must
	// report the session dead (exit 1) so claude-nueva doesn't reject the name;
	// capture-pane reports RC connected so the status poll resolves instantly.
	tmux := writeExe("tmux", `#!/bin/sh
case "$1" in
  has-session) exit 1 ;;
  new-session) printf '%s\n' "$@" > `+rec+`; printf 'sess1\n'; exit 0 ;;
  capture-pane) printf '/rc connected\n'; exit 0 ;;
esac
exit 1
`)
	_ = writeExe("claude", "#!/bin/sh\nexit 0")
	_ = writeExe("bash", "#!/bin/sh\nexit 0")
	systemdRun := writeExe("systemd-run", fakeSystemdRun)

	profiles := filepath.Join(base, "perfiles")
	os.MkdirAll(profiles, 0700)
	os.WriteFile(filepath.Join(profiles, "estandar.json"), []byte(`{"model":"sonnet"}`), 0600)

	h := New(Options{
		ProfilesPath:     profiles,
		SettingsPath:     filepath.Join(base, "settings.json"),
		ConvPath:         filepath.Join(base, "conv"),
		ClaudeBinary:     filepath.Join(binDir, "claude"),
		TmuxBinary:       tmux,
		SystemdRunBinary: systemdRun,
		BashBinary:       filepath.Join(binDir, "bash"),
		RcBootstrap:      "estandar",
		RcWaitSeconds:    2,
		RcPollSeconds:    0,
		Home:             base,
	})

	if _, err := h.Exec("claude-nueva", map[string]string{"profile": "estandar", "name": "abc123"}); err != nil {
		t.Fatalf("claude-nueva: %v", err)
	}

	raw, err := os.ReadFile(rec)
	if err != nil {
		t.Fatalf("read argv: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(raw)), "\n")
	t.Logf("tmux argv: %v", args)

	find := func(want string) int {
		for i, a := range args {
			if a == want {
				return i
			}
		}
		return -1
	}
	// The session name is its own tmux argument after -s.
	i := find("-s")
	if i < 0 || i+1 >= len(args) || args[i+1] != "abc123" {
		t.Errorf("session name not a separate tmux arg (-s abc123); argv=%v", args)
	}

	// The pane command is the last argv element and is single-quoted for -lc.
	pane := args[len(args)-1]
	if !strings.HasPrefix(pane, filepath.Join(binDir, "bash")+" -lc '") || !strings.HasSuffix(pane, "'") {
		t.Fatalf("pane command not 'bash -lc ...' single-quoted: %q", pane)
	}
	inner := pane[strings.Index(pane, "-lc '")+len("-lc '"):]
	inner = strings.TrimSuffix(inner, "'")

	if strings.Contains(inner, "abc123") {
		t.Errorf("session name leaked into the pane shell string: %s", inner)
	}
	if strings.ContainsAny(inner, ";$`&|") {
		t.Errorf("unexpected metacharacter in pane shell string: %s", inner)
	}
	for _, want := range []string{"--settings", "estandar.json", "--remote-control"} {
		if !strings.Contains(inner, want) {
			t.Errorf("pane shell string missing %q: %s", want, inner)
		}
	}
}
