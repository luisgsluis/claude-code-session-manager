package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sendProbe wires a Host whose fake tmux records every transport a send can
// take: send-keys arguments, the bytes handed to load-buffer on stdin, the
// paste-buffer arguments, and any delete-buffer cleanup.
type sendProbe struct {
	h        *Host
	sendkeys string
	buffer   string
	paste    string
	delete   string
}

func newSendProbe(t *testing.T, extra map[string]string) *sendProbe {
	t.Helper()
	dir := t.TempDir()
	p := &sendProbe{
		sendkeys: filepath.Join(dir, "sendkeys.txt"),
		buffer:   filepath.Join(dir, "buffer.txt"),
		paste:    filepath.Join(dir, "paste.txt"),
		delete:   filepath.Join(dir, "delete.txt"),
	}
	env := map[string]string{
		"FAKE_TMUX_SENDKEYS": p.sendkeys,
		"FAKE_TMUX_BUFFER":   p.buffer,
		"FAKE_TMUX_PASTE":    p.paste,
		"FAKE_TMUX_DELETE":   p.delete,
	}
	for k, v := range extra {
		env[k] = v
	}
	p.h = fakeHost(t, env)
	return p
}

func (p *sendProbe) read(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}

func (p *sendProbe) sentKeys() string { return p.read(p.sendkeys) }
func (p *sendProbe) pasted() string   { return p.read(p.buffer) }
func (p *sendProbe) pasteArgs() string {
	return p.read(p.paste)
}
func (p *sendProbe) deleted() string { return p.read(p.delete) }

// TestSendTextShortUsesSendKeys: everything that fit under the OLD 2000-char
// cap must keep travelling the exact path it always did. This matters beyond
// nostalgia: a slash command a user types as a chat message (/model sonnet) is
// only recognised by the TUI when it arrives as keystrokes, so silently
// promoting short messages to a paste would break them.
func TestSendTextShortUsesSendKeys(t *testing.T) {
	p := newSendProbe(t, nil)

	if _, err := p.h.sessionSend("3", "hola qué tal", ""); err != nil {
		t.Fatalf("sessionSend: %v", err)
	}
	if got := p.sentKeys(); !strings.Contains(got, "hola qué tal") {
		t.Errorf("short text did not go through send-keys: %q", got)
	}
	if got := p.pasteArgs(); got != "" {
		t.Errorf("short text must not use paste-buffer, got: %q", got)
	}
}

// TestSendTextShortMultilineUsesPasteBuffer: a short message (well under
// pasteThreshold) with embedded newlines — a paragraph typed with Shift+Enter
// in the chat box — must still go through the paste transport, not send-keys.
// send-keys -l ships embedded LF bytes completely untranslated (measured on
// tmux 3.5a), and the TUI's keypress parser treats a bare LF as Enter just
// like CR, so a short multi-line message sent as keystrokes is submitted once
// per line instead of once as a whole. Only length used to gate the
// transport; this is the regression that slipped through that gap.
func TestSendTextShortMultilineUsesPasteBuffer(t *testing.T) {
	p := newSendProbe(t, nil)
	short := "línea 1\nlínea 2\nlínea 3" // well under pasteThreshold

	if _, err := p.h.sessionSend("3", short, ""); err != nil {
		t.Fatalf("sessionSend: %v", err)
	}
	if got := p.pasted(); got != short {
		t.Errorf("short multi-line text did not go through paste-buffer: got %q, want %q", got, short)
	}
	args := p.pasteArgs()
	if !strings.Contains(args, "-r") {
		t.Errorf("paste-buffer must pass -r or every newline submits the message: %q", args)
	}
	if got := p.sentKeys(); strings.Contains(got, "línea 1") {
		t.Errorf("short multi-line text leaked into send-keys instead of paste: %q", got)
	}
}

// TestSendTextLongUsesPasteBuffer covers the new transport end to end: the
// message reaches tmux through a buffer, and an Enter still follows so it is
// actually submitted.
func TestSendTextLongUsesPasteBuffer(t *testing.T) {
	p := newSendProbe(t, nil)
	long := strings.Repeat("a", pasteThreshold+1)

	if _, err := p.h.sessionSend("3", long, ""); err != nil {
		t.Fatalf("sessionSend: %v", err)
	}
	if got := p.pasted(); got != long {
		t.Errorf("buffer content differs: got %d bytes, want %d", len(got), len(long))
	}
	if got := p.pasteArgs(); !strings.Contains(got, "-b ccsm-3") {
		t.Errorf("paste-buffer did not target the session's own buffer: %q", got)
	}
	// The literal \r is the submit; without it the message sits in the
	// composer unsent.
	if got := p.sentKeys(); !strings.Contains(got, "\r") {
		t.Errorf("no Enter followed the paste: %q", got)
	}
}

// TestPasteBufferKeepsNewlines is the regression that matters most. tmux's
// paste-buffer translates LF to CR by default, and CR is exactly what submits
// a message — so without -r a five-line prompt is sent as five separate
// messages, the first of which is a fragment. Measured on tmux 3.5a: default
// and -p alone both yield 0 LF / 6 CR for a 6-newline payload; -r yields 6 LF
// / 0 CR.
func TestPasteBufferKeepsNewlines(t *testing.T) {
	p := newSendProbe(t, nil)
	multiline := strings.Repeat("línea de texto\n", 200) // > pasteThreshold

	if _, err := p.h.sessionSend("3", multiline, ""); err != nil {
		t.Fatalf("sessionSend: %v", err)
	}
	args := p.pasteArgs()
	if !strings.Contains(args, "-r") {
		t.Fatalf("paste-buffer must pass -r or every newline submits the message: %q", args)
	}
	if !strings.Contains(args, "-p") {
		t.Errorf("paste-buffer should ask for bracketed paste (-p): %q", args)
	}
	if !strings.Contains(args, "-d") {
		t.Errorf("paste-buffer should drop the buffer afterwards (-d): %q", args)
	}
	if got := strings.Count(p.pasted(), "\n"); got != 200 {
		t.Errorf("newlines mangled on the way to the buffer: got %d, want 200", got)
	}
}

// TestSendTextLimits pins both ends of the new cap.
func TestSendTextLimits(t *testing.T) {
	p := newSendProbe(t, nil)

	if _, err := p.h.sessionSend("3", strings.Repeat("a", MaxSendLen), ""); err != nil {
		t.Errorf("a message of exactly MaxSendLen must be accepted: %v", err)
	}
	_, err := p.h.sessionSend("3", strings.Repeat("a", MaxSendLen+1), "")
	if err == nil {
		t.Fatal("expected MaxSendLen+1 to be rejected")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("unexpected rejection message: %v", err)
	}
}

// TestSendTextCapIsMeasuredInRunes: the cap counts runes, not bytes, so an
// accented Spanish prompt is not rejected at roughly half the advertised
// length.
func TestSendTextCapIsMeasuredInRunes(t *testing.T) {
	p := newSendProbe(t, nil)
	// Every rune is 2 bytes in UTF-8: well over MaxSendLen bytes, exactly
	// MaxSendLen runes.
	if _, err := p.h.sessionSend("3", strings.Repeat("ñ", MaxSendLen), ""); err != nil {
		t.Errorf("MaxSendLen runes of 2-byte characters must be accepted: %v", err)
	}
}

// TestSanitizePaneText: control characters must not reach the terminal. The
// escape case is not theoretical — a literal bracketed-paste terminator in the
// payload would end the paste early and let the rest be read as keystrokes.
func TestSanitizePaneText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"keeps newlines and tabs", "a\nb\tc", "a\nb\tc"},
		{"strips ESC", "a\x1bb", "ab"},
		{"strips bracketed paste end", "a\x1b[201~rm -rf /", "a[201~rm -rf /"},
		{"strips NUL and BEL", "a\x00b\x07c", "abc"},
		{"strips DEL", "a\x7fb", "ab"},
		{"strips carriage return", "a\rb", "ab"},
		{"keeps accents and emoji", "ñ é 🎤", "ñ é 🎤"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sanitizePaneText(c.in); got != c.want {
				t.Errorf("sanitizePaneText(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestSendTextStripsControlCharsOnBothPaths: sanitising in sendText rather
// than in one transport means a hostile payload is cleaned whether it is short
// enough for send-keys or long enough to be pasted.
func TestSendTextStripsControlCharsOnBothPaths(t *testing.T) {
	t.Run("short", func(t *testing.T) {
		p := newSendProbe(t, nil)
		if _, err := p.h.sessionSend("3", "hola\x1b[201~adios", ""); err != nil {
			t.Fatalf("sessionSend: %v", err)
		}
		if strings.Contains(p.sentKeys(), "\x1b") {
			t.Error("ESC reached the pane through send-keys")
		}
	})
	t.Run("long", func(t *testing.T) {
		p := newSendProbe(t, nil)
		payload := strings.Repeat("a", pasteThreshold) + "\x1b[201~tail"
		if _, err := p.h.sessionSend("3", payload, ""); err != nil {
			t.Fatalf("sessionSend: %v", err)
		}
		if strings.Contains(p.pasted(), "\x1b") {
			t.Error("ESC reached the pane through paste-buffer")
		}
	})
}

// TestPasteBufferCleansUpOnFailure: -d deletes the buffer as part of a
// successful paste, so a paste that fails would otherwise leave the message
// sitting in tmux's paste stack.
func TestPasteBufferCleansUpOnFailure(t *testing.T) {
	p := newSendProbe(t, map[string]string{"FAKE_TMUX_PASTE_FAIL": "1"})

	_, err := p.h.sessionSend("3", strings.Repeat("a", pasteThreshold+1), "")
	if err == nil {
		t.Fatal("expected a failing paste-buffer to surface as an error")
	}
	if got := p.deleted(); !strings.Contains(got, "ccsm-3") {
		t.Errorf("buffer not cleaned up after a failed paste: %q", got)
	}
}

// TestLoadBufferFailureStopsBeforePasting: if the text never made it into the
// buffer, pasting would submit whatever the buffer happened to hold.
func TestLoadBufferFailureStopsBeforePasting(t *testing.T) {
	p := newSendProbe(t, map[string]string{"FAKE_TMUX_LOAD_FAIL": "1"})

	_, err := p.h.sessionSend("3", strings.Repeat("a", pasteThreshold+1), "")
	if err == nil {
		t.Fatal("expected a failing load-buffer to surface as an error")
	}
	if got := p.pasteArgs(); got != "" {
		t.Errorf("pasted despite the buffer never being loaded: %q", got)
	}
}

// TestSendTextBufferIsPerSession: two sessions sending long messages at the
// same time must not share one buffer, or one overwrites the other's payload
// between load and paste.
func TestSendTextBufferIsPerSession(t *testing.T) {
	p := newSendProbe(t, nil)
	long := strings.Repeat("a", pasteThreshold+1)

	if _, err := p.h.sessionSend("3", long, ""); err != nil {
		t.Fatalf("sessionSend 3: %v", err)
	}
	if _, err := p.h.sessionSend("7", long, ""); err != nil {
		t.Fatalf("sessionSend 7: %v", err)
	}
	args := p.pasteArgs()
	if !strings.Contains(args, "-b ccsm-3") || !strings.Contains(args, "-b ccsm-7") {
		t.Errorf("buffers are not per-session: %q", args)
	}
}

// TestSendKeysUnchangedForSlashCommands: the control commands CCSM types
// itself must keep using send-keys. Pasting "/remote-control" risks it landing
// as literal text instead of being recognised as a command, which would
// silently break RC re-registration.
func TestSendKeysUnchangedForSlashCommands(t *testing.T) {
	p := newSendProbe(t, nil)

	if err := p.h.sendKeys("3", "/remote-control"); err != nil {
		t.Fatalf("sendKeys: %v", err)
	}
	if got := p.sentKeys(); !strings.Contains(got, "/remote-control") {
		t.Errorf("slash command did not go through send-keys: %q", got)
	}
	if got := p.pasteArgs(); got != "" {
		t.Errorf("slash command must never be pasted: %q", got)
	}
}
