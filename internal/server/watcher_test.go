package server

import (
	"encoding/json"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/agent"
)

// fakeStatusAgent answers "session-status" with a canned turnStatus, one per
// call in order (repeating the last one once exhausted). Anything else
// (tmux-ls, used by turnWatchOnce but not by watchSession directly) errors,
// since these tests call watchSession itself.
type fakeStatusAgent struct {
	statuses []turnStatus
	calls    int
}

func (f *fakeStatusAgent) Exec(cmd string, args map[string]string) (*agent.Response, error) {
	if cmd != "session-status" {
		return nil, errFakeUnsupported
	}
	st := f.statuses[min(f.calls, len(f.statuses)-1)]
	f.calls++
	data, _ := json.Marshal(st)
	return &agent.Response{OK: true, Data: data}, nil
}

var errFakeUnsupported = &fakeErr{"unsupported op in fakeStatusAgent"}

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }

// drainEvents reads every broadcast eventHub.broadcast has already queued on
// ch without blocking, returning their "action" fields in order.
func drainEvents(t *testing.T, ch chan []byte) []string {
	t.Helper()
	var actions []string
	for {
		select {
		case data := <-ch:
			var ev struct {
				Action string `json:"action"`
			}
			if err := json.Unmarshal(data, &ev); err != nil {
				t.Fatalf("bad event payload: %v", err)
			}
			actions = append(actions, ev.Action)
		default:
			return actions
		}
	}
}

// TestWatchSessionChoiceIdentity reproduces the bug reported in production:
// an AskUserQuestion picker (A) resolves — by a typed answer or by being
// interrupted with an unrelated message — and Claude immediately asks a
// different question (B) before the next turnWatchPoll sample. The old code
// tracked only "is some choice pending", so with A already flagged as
// notified, B's mere presence never re-armed it: no session_choice event
// fired, and an open grid tile kept showing A's now-stale panel forever,
// with the "Aprobar" click confirming whatever the real (different) pane was
// actually on. Comparing the question's own identity must announce B too.
func TestWatchSessionChoiceIdentity(t *testing.T) {
	s := &Server{events: newEventHub()}
	ch := s.events.subscribe()
	defer s.events.unsubscribe(ch)

	agent := &fakeStatusAgent{statuses: []turnStatus{
		{Waiting: "choice", WaitingID: "¿A?"},
		{Waiting: "choice", WaitingID: "¿A?"}, // same question polled again: no repeat
		{Waiting: "choice", WaitingID: "¿B?"}, // different question, same poll cadence
	}}
	s.exec = agent
	state := map[string]*turnWatchState{}

	s.watchSession(state, "7")
	s.watchSession(state, "7")
	s.watchSession(state, "7")

	got := drainEvents(t, ch)
	want := []string{"session_choice", "session_choice"} // A once, then B
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestWatchSessionChoiceResolvedThenRepeats: once a choice clears (no more
// pending), asking the *same* question text again later is a genuinely new
// instance and must re-announce — the identity guard only suppresses a
// question that is still the one currently on screen.
func TestWatchSessionChoiceResolvedThenRepeats(t *testing.T) {
	s := &Server{events: newEventHub()}
	ch := s.events.subscribe()
	defer s.events.unsubscribe(ch)

	agent := &fakeStatusAgent{statuses: []turnStatus{
		{Waiting: "choice", WaitingID: "¿Otra vez?"},
		{}, // resolved: no choice pending
		{Waiting: "choice", WaitingID: "¿Otra vez?"}, // asked again from scratch
	}}
	s.exec = agent
	state := map[string]*turnWatchState{}

	s.watchSession(state, "7")
	s.watchSession(state, "7")
	s.watchSession(state, "7")

	got := drainEvents(t, ch)
	if len(got) != 2 || got[0] != "session_choice" || got[1] != "session_choice" {
		t.Errorf("events = %v, want two session_choice broadcasts", got)
	}
}

// TestWatchSessionIdleFirstSampleDoesNotSilenceLaterChoices is the direct
// regression test for the empty-vs-pending distinction: a session's first
// poll lands idle far more often than not (most sessions don't open with a
// pending question), so this is the common case, not an edge case — get it
// wrong (e.g. by reading an empty WaitingID as "notified") and a session's
// very first sample eats the one-shot "already notified" state, and no
// AskUserQuestion for that session ever announces again for as long as the
// process runs.
func TestWatchSessionIdleFirstSampleDoesNotSilenceLaterChoices(t *testing.T) {
	s := &Server{events: newEventHub()}
	ch := s.events.subscribe()
	defer s.events.unsubscribe(ch)

	agent := &fakeStatusAgent{statuses: []turnStatus{
		{}, // idle: the common first sample
		{}, // still idle
		{Waiting: "choice", WaitingID: "¿A?"},
		{}, // resolved
		{Waiting: "choice", WaitingID: "¿B?"},
	}}
	s.exec = agent
	state := map[string]*turnWatchState{}

	for i := 0; i < 5; i++ {
		s.watchSession(state, "7")
	}

	got := drainEvents(t, ch)
	if len(got) != 2 || got[0] != "session_choice" || got[1] != "session_choice" {
		t.Errorf("events = %v, want session_choice for A and for B, none for the idle samples", got)
	}
}

// TestWatchSessionApprovalIdentity is the structural counterpart of
// TestWatchSessionChoiceIdentity for approval-type dialogs (command/file
// approval, plan approval): they render through the exact same numbered
// picker as AskUserQuestion (see host.paneWaitingDetail), so they share the
// identical gap — a second, different approval landing right after the
// first resolves, within the same poll, must still announce as
// session_waiting, not be swallowed because "approval" alone hasn't changed.
func TestWatchSessionApprovalIdentity(t *testing.T) {
	s := &Server{events: newEventHub()}
	ch := s.events.subscribe()
	defer s.events.unsubscribe(ch)

	agent := &fakeStatusAgent{statuses: []turnStatus{
		{Waiting: "approval", WaitingID: "Do you want to create foo.txt?"},
		{Waiting: "approval", WaitingID: "Do you want to proceed?"}, // different approval, same reason
	}}
	s.exec = agent
	state := map[string]*turnWatchState{}

	s.watchSession(state, "7")
	s.watchSession(state, "7")

	got := drainEvents(t, ch)
	if len(got) != 2 || got[0] != "session_waiting" || got[1] != "session_waiting" {
		t.Errorf("events = %v, want two session_waiting broadcasts (different approvals)", got)
	}
}

// TestWatchSessionSetupWizardAnnounces: before this, "setup" (the auto-mode
// environment wizard) had no watcher coverage at all — an open grid tile
// already renders a "skip" control for waiting === 'setup' (index.html), but
// nothing ever told it the wizard had appeared, so it stayed on whatever it
// last fetched until some unrelated event (or the tile being reopened)
// happened to refresh it.
func TestWatchSessionSetupWizardAnnounces(t *testing.T) {
	s := &Server{events: newEventHub()}
	ch := s.events.subscribe()
	defer s.events.unsubscribe(ch)

	agent := &fakeStatusAgent{statuses: []turnStatus{
		{Waiting: "setup", WaitingID: "setup"},
	}}
	s.exec = agent
	state := map[string]*turnWatchState{}

	s.watchSession(state, "7")

	got := drainEvents(t, ch)
	if len(got) != 1 || got[0] != "session_waiting" {
		t.Errorf("events = %v, want one session_waiting broadcast for the setup wizard", got)
	}
}
