package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const convUUID = "20000000-0000-0000-0000-0000000000aa"

func writeConv(t *testing.T, h *Host, content string) string {
	t.Helper()
	path := h.convPath + "/" + convUUID + ".jsonl"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewUUID(t *testing.T) {
	a, b := newUUID(), newUUID()
	if len(a) != 36 || !uuidPattern.MatchString(a) {
		t.Errorf("newUUID not a valid UUID: %q", a)
	}
	if a == b {
		t.Error("two newUUID calls collided")
	}
}

func TestSessionConvFallback(t *testing.T) {
	h := fakeHost(t, nil) // no FAKE_TMUX_PANE_PID -> panePID 0 -> newestActiveConv
	writeConv(t, h, `{"type":"user","message":{"content":"hola"}}
`)
	data, err := h.Exec("session-conv", map[string]string{"name": "x"})
	if err != nil {
		t.Fatalf("session-conv: %v", err)
	}
	got := data.(map[string]any)
	if got["id"] != convUUID || got["ready"] != true {
		t.Errorf("session-conv fallback: %v", got)
	}
}

func TestSessionConvNotFound(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_DEAD": "x"})
	if _, err := h.Exec("session-conv", map[string]string{"name": "x"}); err == nil {
		t.Fatal("expected 404 for dead session")
	}
	if _, err := h.Exec("session-conv", map[string]string{"name": "bad/name"}); err == nil {
		t.Fatal("expected 400 for invalid name")
	}
}

func TestSessionChat(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_LINE": "insert /rc connected",
	})
	// Long assistant text (>400 runes) proves no truncation.
	long := strings.Repeat("palabra ", 100)
	content := `{"type":"user","timestamp":"2026-08-11T10:00:00Z","cwd":"/home/admin/x","message":{"content":"pregunta"}}
{"type":"assistant","timestamp":"2026-08-11T10:00:05Z","message":{"model":"sonnet","content":[{"type":"text","text":"` + long + `"}]}}
{"type":"user","isMeta":true,"timestamp":"2026-08-11T10:00:06Z","message":{"content":"meta"}}
{"type":"tool_result","timestamp":"2026-08-11T10:00:07Z","message":{"content":"ignored"}}
{"type":"assistant","timestamp":"2026-08-11T10:00:10Z","message":{"model":"opus","content":"segunda"}}
`
	writeConv(t, h, content)

	data, err := h.Exec("session-chat", map[string]string{"name": "x"})
	if err != nil {
		t.Fatalf("session-chat: %v", err)
	}
	got := data.(map[string]any)
	if got["id"] != convUUID || got["ready"] != true {
		t.Errorf("session-chat meta: %v", got)
	}
	wantCreated := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC).Unix()
	if got["created"] != int64(wantCreated) {
		t.Errorf("created from first timestamp: %v, want %d", got["created"], wantCreated)
	}
	if got["status"] != "rc_connected" {
		t.Errorf("status: %v", got["status"])
	}
	if got["mode"] != "insert" {
		t.Errorf("mode from pane line: %v", got["mode"])
	}
	msgs := got["messages"].([]map[string]any)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (meta/tool skipped), got %d", len(msgs))
	}
	if msgs[0]["role"] != "user" || msgs[0]["content"] != "pregunta" {
		t.Errorf("first message: %v", msgs[0])
	}
	assistant := msgs[1]["content"].(string)
	if len(assistant) <= 400 {
		t.Errorf("assistant content truncated: %d runes", len(assistant))
	}
	if msgs[2]["role"] != "assistant" || msgs[2]["content"] != "segunda" {
		t.Errorf("second assistant: %v", msgs[2])
	}
	// Last assistant message's model wins.
	if got["model"] != "opus" {
		t.Errorf("model from last assistant: %v, want opus", got["model"])
	}
}

func TestSessionChatNotReady(t *testing.T) {
	h := fakeHost(t, nil)
	// No jsonl at all -> newestActiveConv returns "" -> ready:false.
	data, err := h.Exec("session-chat", map[string]string{"name": "x"})
	if err != nil {
		t.Fatalf("session-chat: %v", err)
	}
	got := data.(map[string]any)
	if got["ready"] != false {
		t.Errorf("expected ready=false with no transcript, got %v", got)
	}
}

func TestSessionSend(t *testing.T) {
	sendPath := filepath.Join(t.TempDir(), "send.log")
	h := fakeHost(t, map[string]string{"FAKE_TMUX_SENDKEYS": sendPath, "FAKE_TMUX_PANE_PID": "1234"})

	t.Run("text sends literal then Enter", func(t *testing.T) {
		os.Remove(sendPath)
		_, err := h.Exec("session-send", map[string]string{"name": "x", "text": "hola mundo"})
		if err != nil {
			t.Fatalf("session-send text: %v", err)
		}
		b, _ := os.ReadFile(sendPath)
		got := string(b)
		if !strings.Contains(got, "send-keys -t %0 -l hola mundo") {
			t.Errorf("literal not sent: %q", got)
		}
		if !strings.Contains(got, "send-keys -t %0 Enter") {
			t.Errorf("Enter not sent: %q", got)
		}
	})

	t.Run("special key goes to whitelist", func(t *testing.T) {
		os.Remove(sendPath)
		_, err := h.Exec("session-send", map[string]string{"name": "x", "keys": "ctrl-c"})
		if err != nil {
			t.Fatalf("session-send key: %v", err)
		}
		b, _ := os.ReadFile(sendPath)
		if !strings.Contains(string(b), "send-keys -t %0 C-c") {
			t.Errorf("ctrl-c not sent: %q", string(b))
		}
	})

	t.Run("validations", func(t *testing.T) {
		os.Remove(sendPath)
		if _, err := h.Exec("session-send", map[string]string{"name": "x"}); err == nil {
			t.Error("expected 400 with no text/keys")
		}
		if _, err := h.Exec("session-send", map[string]string{"name": "x", "keys": "rm -rf"}); err == nil {
			t.Error("expected 400 for non-whitelisted key")
		}
		if _, err := h.Exec("session-send", map[string]string{"name": "x", "text": strings.Repeat("a", maxSendLen+1)}); err == nil {
			t.Error("expected 400 for overlong text")
		}
		if _, err := h.Exec("session-send", map[string]string{"name": "x", "text": "x"}); err != nil {
			t.Errorf("plain send failed: %v", err)
		}
	})

	t.Run("dead session", func(t *testing.T) {
		d := fakeHost(t, map[string]string{"FAKE_TMUX_DEAD": "y"})
		if _, err := d.Exec("session-send", map[string]string{"name": "y", "text": "x"}); err == nil {
			t.Error("expected 404 for dead session")
		}
	})
}

// TestSessionSendConcurrent reproduces the bug reported in production: a
// model switch (typed as "/model sonnet" through the same text path a chat
// message uses) racing the user's next message, neither awaited by the
// other client-side, interleaved their literal send-keys calls into the
// pane's input line before either Enter landed — submitting one line,
// "/model sonnetQuiero que verifiques...", which Claude Code rejected as an
// invalid model name. sessionSendLock (host.go) serializes sessionSend calls
// per session; this asserts the fake tmux log never shows a literal call
// from one goroutine followed by the other's literal instead of its own
// Enter, with the delay (FAKE_TMUX_SENDKEYS_DELAY) making that window wide
// enough to hit reliably if the lock were ever removed.
func TestSessionSendConcurrent(t *testing.T) {
	sendPath := filepath.Join(t.TempDir(), "send.log")
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_SENDKEYS":       sendPath,
		"FAKE_TMUX_SENDKEYS_DELAY": "0.05",
	})

	var wg sync.WaitGroup
	texts := []string{"/model sonnet", "Quiero que verifiques hyperbackup"}
	for _, text := range texts {
		wg.Add(1)
		go func(text string) {
			defer wg.Done()
			if _, err := h.Exec("session-send", map[string]string{"name": "x", "text": text}); err != nil {
				t.Errorf("session-send %q: %v", text, err)
			}
		}(text)
	}
	wg.Wait()

	b, err := os.ReadFile(sendPath)
	if err != nil {
		t.Fatalf("read send log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 send-keys calls (2 × literal+Enter), got %d: %q", len(lines), lines)
	}
	// Every literal line must be immediately followed by ITS OWN Enter (same
	// -t target, "send-keys -t <target> Enter") — never by the other
	// goroutine's literal, which is exactly what the reported bug looked
	// like ("/model sonnet" + the next message merged before either Enter
	// fired).
	for i := 0; i < len(lines); i += 2 {
		fields := strings.Fields(lines[i])
		if len(fields) < 4 || fields[3] != "-l" {
			t.Fatalf("line %d: expected a literal send-keys, got %q", i, lines[i])
		}
		target := fields[2]
		wantEnter := "send-keys -t " + target + " Enter"
		if lines[i+1] != wantEnter {
			t.Errorf("line %d: expected %q right after its own literal (%q), got %q — calls interleaved", i, wantEnter, lines[i], lines[i+1])
		}
	}
}

func TestPanePIDAndProcRead(t *testing.T) {
	h := fakeHost(t, map[string]string{"FAKE_TMUX_PANE_PID": "1234"})
	if pid := h.panePID("x"); pid != 1234 {
		t.Errorf("panePID: %d", pid)
	}
	if id, pid := h.paneInfo("x"); id != "%0" || pid != 1234 {
		t.Errorf("paneInfo(x) = %q,%d", id, pid)
	}
	if tgt := h.paneTarget("x"); tgt != "%0" {
		t.Errorf("paneTarget(x) = %q", tgt)
	}
	if procCmdline(os.Getpid()) == "" {
		t.Error("procCmdline of self should be non-empty")
	}
	if id := uuidRe.FindString(procCmdline(os.Getpid())); id != "" {
		t.Errorf("unexpected uuid in own argv: %q", id)
	}
}

// TestPaneInfoMatchesSessionName is the regression for a session literally
// named "0": pane commands must address it by pane id matched on the session
// NAME from the full pane table, never by tmux target resolution.
func TestPaneInfoMatchesSessionName(t *testing.T) {
	h := fakeHost(t, map[string]string{
		"FAKE_TMUX_PANE_SESSION": "0",
		"FAKE_TMUX_PANE_ID":      "%7",
		"FAKE_TMUX_PANE_PID":     "4321",
	})
	if id, pid := h.paneInfo("0"); id != "%7" || pid != 4321 {
		t.Errorf("paneInfo(0) = %q,%d", id, pid)
	}
	if tgt := h.paneTarget("0"); tgt != "%7" {
		t.Errorf("paneTarget(0) = %q", tgt)
	}
	if id, _ := h.paneInfo("otra"); id != "" {
		t.Errorf("paneInfo(otra) matched %q", id)
	}
}

func TestSessionCreated(t *testing.T) {
	t.Run("epoch", func(t *testing.T) {
		h := fakeHost(t, map[string]string{"FAKE_TMUX_CREATED": "1786442261"})
		if got := h.sessionCreated("x"); got != 1786442261 {
			t.Errorf("sessionCreated = %d", got)
		}
	})
	t.Run("no stub", func(t *testing.T) {
		h := fakeHost(t, nil)
		if got := h.sessionCreated("x"); got != 0 {
			t.Errorf("sessionCreated without stub = %d", got)
		}
	})
	t.Run("bad value", func(t *testing.T) {
		h := fakeHost(t, map[string]string{"FAKE_TMUX_CREATED": "abc"})
		if got := h.sessionCreated("x"); got != 0 {
			t.Errorf("sessionCreated bad value = %d", got)
		}
	})
}

// TestLifetimeConvMatchesIdleSession is the regression for the chat coming up
// empty: the old fallback only looked at transcripts written in the last 2
// minutes, so an idle session's transcript (last written 30 min ago, but still
// by this running session) was missed and the chat stayed empty until the next
// message touched the file.
func TestLifetimeConvMatchesIdleSession(t *testing.T) {
	created := time.Now().Add(-2 * time.Hour).Unix()
	h := fakeHost(t, map[string]string{"FAKE_TMUX_CREATED": strconv.FormatInt(created, 10)})
	id := "40000000-0000-0000-0000-0000000000cc"
	path := h.convPath + "/" + id + ".jsonl"
	os.WriteFile(path, []byte(`{"type":"user","message":{"content":"hola"}}`+"\n"), 0600)
	old := time.Now().Add(-30 * time.Minute)
	os.Chtimes(path, old, old)
	if got := h.convIDForSession("x"); got != id {
		t.Errorf("convIDForSession = %q, want %q (idle session missed)", got, id)
	}
}

func TestLifetimeConvSkipsOlderTranscript(t *testing.T) {
	created := time.Now().Add(-1 * time.Hour).Unix()
	h := fakeHost(t, map[string]string{"FAKE_TMUX_CREATED": strconv.FormatInt(created, 10)})
	id := "40000000-0000-0000-0000-0000000000dd"
	path := h.convPath + "/" + id + ".jsonl"
	os.WriteFile(path, []byte(`{"type":"user","message":{"content":"viejo"}}`+"\n"), 0600)
	old := time.Now().Add(-3 * time.Hour)
	os.Chtimes(path, old, old)
	if got := h.convIDForSession("x"); got != "" {
		t.Errorf("convIDForSession = %q, want empty", got)
	}
}

// TestLifetimeConvFallsBackToFresh: when the session creation can't be read,
// the fallback degrades to the freshness window (recently written transcripts).
func TestLifetimeConvFallsBackToFresh(t *testing.T) {
	h := fakeHost(t, nil) // no FAKE_TMUX_CREATED → sessionCreated 0
	id := "40000000-0000-0000-0000-0000000000ee"
	path := h.convPath + "/" + id + ".jsonl"
	os.WriteFile(path, []byte(`{"type":"user","message":{"content":"fresco"}}`+"\n"), 0600) // mtime now
	if got := h.convIDForSession("x"); got != id {
		t.Errorf("convIDForSession = %q, want %q", got, id)
	}
}

// TestFindUUIDInTree is the regression for the uuid-in-argv lookup. A live
// child carries the uuid in argv, but only a uuid directly after --session-id
// / --resume (NUL-separated, like claude's real argv) counts — a bare uuid
// somewhere in argv (e.g. tool-call shell text) must NOT be matched.
func TestFindUUIDInTree(t *testing.T) {
	uuid := "30000000-0000-0000-0000-0000000000bb"

	runChild := func(t *testing.T, args ...string) {
		t.Helper()
		// `sh -c 'sleep 30' <args...>` stays alive 30s and keeps argv intact.
		// (Plain `sleep --session-id x` exits at once on GNU coreutils and
		// /proc goes empty; the test binary rejects the unknown flag too.)
		cmd := exec.Command("sh", append([]string{"-c", "sleep 30"}, args...)...)
		if err := cmd.Start(); err != nil {
			t.Skipf("cannot spawn child: %v", err)
		}
		t.Cleanup(func() { cmd.Process.Kill() })
	}

	t.Run("session-id marker matches", func(t *testing.T) {
		runChild(t, "--session-id", uuid)
		if id := findUUIDInTree(os.Getpid(), 2); id != uuid {
			t.Errorf("findUUIDInTree = %q, want %q", id, uuid)
		}
	})

	t.Run("bare uuid in argv is not a conversation id", func(t *testing.T) {
		runChild(t, uuid) // positional arg, no flag before it
		if id := findUUIDInTree(os.Getpid(), 2); id != "" {
			t.Errorf("findUUIDInTree matched bare uuid: %q", id)
		}
	})
}
