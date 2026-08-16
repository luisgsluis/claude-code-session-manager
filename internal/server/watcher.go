package server

import (
	"encoding/json"
	"strings"
	"time"
)

// Turn watcher: polls the active sessions and broadcasts SSE events when a
// Claude turn completes ("turn_complete"), or the pane blocks on user input —
// command/file/plan approval or the auto-mode setup wizard ("session_waiting")
// or an AskUserQuestion picker ("session_choice"). None of these reach the
// transcript in a way the chat view surfaces, so a hidden panel tab gets a
// browser notification instead of silence, and an open grid tile knows to
// refetch instead of showing a stale panel.

// turnWatchPoll is how often the watcher samples each active session.
const turnWatchPoll = 4 * time.Second

// turnWatchSettle is how many consecutive idle samples must pass before a
// completed turn is announced — guards mid-turn pauses and footer flicker.
const turnWatchSettle = 2

// turnWatchState tracks what a session has already announced, per session.
type turnWatchState struct {
	idleStreak     int
	lastNotifiedID string
	notifiedWaitID string // identity of the last-announced blocking dialog (any reason); "" while none pending
}

// turnStatus mirrors host.sessionStatus.
type turnStatus struct {
	Working           bool   `json:"working"`
	Waiting           string `json:"waiting"`
	WaitingID         string `json:"waiting_id"`
	LastAssistant     string `json:"last_assistant_id"`
	LastAssistantText string `json:"last_assistant_text"`
}

// startTurnWatcher launches the polling loop. Errors are silent on purpose: a
// dead agent or a session that vanished just means no event this round.
func (s *Server) startTurnWatcher() {
	if s.exec == nil {
		return
	}
	go func() {
		state := map[string]*turnWatchState{}
		for {
			time.Sleep(turnWatchPoll)
			s.turnWatchOnce(state)
		}
	}()
}

func (s *Server) turnWatchOnce(state map[string]*turnWatchState) {
	resp, err := s.exec.Exec("tmux-ls", nil)
	if err != nil {
		return
	}
	var list []map[string]any
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return
	}
	seen := map[string]bool{}
	for _, item := range list {
		name, _ := item["name"].(string)
		if name == "" {
			continue
		}
		seen[name] = true
		s.watchSession(state, name)
	}
	for name := range state {
		if !seen[name] {
			delete(state, name) // dropped or renamed: re-arm
		}
	}
}

func (s *Server) watchSession(state map[string]*turnWatchState, name string) {
	resp, err := s.exec.Exec("session-status", map[string]string{"name": name})
	if err != nil {
		return
	}
	var st turnStatus
	if err := json.Unmarshal(resp.Data, &st); err != nil {
		return
	}
	ws := state[name]
	if ws == nil {
		ws = &turnWatchState{}
		// Prime with the current assistant id so a session that was already idle
		// when the watcher started (or the container restarted) is not announced.
		ws.lastNotifiedID = st.LastAssistant
		state[name] = ws
	}

	// Any dialog blocking the pane needs the user to act: command/file/plan
	// approval, an AskUserQuestion picker, or the auto-mode setup wizard —
	// all three report a non-empty Waiting reason from host.paneWaitingDetail.
	// Tracked by the dialog's own identity (WaitingID — the question or
	// approval detail line, or the reason itself for setup, which has no
	// such line), not just "is something pending" or "which reason": two
	// different dialogs, even of the SAME reason, can land in the same
	// turnWatchPoll window — one resolves, a different one immediately
	// follows — and a reason-only guard sees no change on the second
	// sample, silently swallowing it. This was found and fixed for choice
	// dialogs specifically; approval and plan-approval render through the
	// exact same numbered picker (see paneWaitingDetail) and share the
	// identical structural gap, and setup previously had no watcher
	// coverage at all — a pending setup wizard never told an open grid tile
	// to refresh, even though the tile already renders a "skip" control for
	// it (index.html, waiting === 'setup').
	if st.WaitingID != "" && st.WaitingID != ws.notifiedWaitID {
		ws.notifiedWaitID = st.WaitingID
		action := "session_waiting"
		if st.Waiting == "choice" {
			action = "session_choice"
		}
		s.events.broadcast(action, name, "session "+name+": "+st.WaitingID)
	} else if st.WaitingID == "" {
		ws.notifiedWaitID = ""
	}

	// Turn completed: the pane went idle and a new assistant message settled.
	if st.Working {
		ws.idleStreak = 0
	} else {
		ws.idleStreak++
	}
	if !st.Working && ws.idleStreak >= turnWatchSettle &&
		st.LastAssistant != "" && st.LastAssistant != ws.lastNotifiedID {
		ws.lastNotifiedID = st.LastAssistant
		s.events.broadcast("turn_complete", name, "session "+name+": "+turnPreview(st.LastAssistantText))
	}
}

// turnPreview shortens the last assistant text for a notification body.
func turnPreview(text string) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return "turn completed"
	}
	if len(t) > 120 {
		t = t[:120] + "…"
	}
	return t
}
