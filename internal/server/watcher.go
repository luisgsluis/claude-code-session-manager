package server

import (
	"encoding/json"
	"strings"
	"time"
)

// Turn watcher: polls the active sessions and broadcasts SSE events when a
// Claude turn completes ("turn_complete"), an approval is requested
// ("session_waiting") or an AskUserQuestion picker appears ("session_choice").
// These never reach the transcript in a way the chat view surfaces, so a
// hidden panel tab gets a browser notification instead of silence.

// turnWatchPoll is how often the watcher samples each active session.
const turnWatchPoll = 4 * time.Second

// turnWatchSettle is how many consecutive idle samples must pass before a
// completed turn is announced — guards mid-turn pauses and footer flicker.
const turnWatchSettle = 2

// turnWatchState tracks what a session has already announced, per session.
type turnWatchState struct {
	idleStreak     int
	lastNotifiedID string
	notifiedWait   string
	notifiedChoice bool
}

// turnStatus mirrors host.sessionStatus.
type turnStatus struct {
	Working           bool            `json:"working"`
	Waiting           string          `json:"waiting"`
	Choice            json.RawMessage `json:"choice"`
	LastAssistant     string          `json:"last_assistant_id"`
	LastAssistantText string          `json:"last_assistant_text"`
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

	// Approval requested (command permission, file edit, …).
	if st.Waiting == "approval" && st.Waiting != ws.notifiedWait {
		ws.notifiedWait = "approval"
		s.events.broadcast("session_waiting", name, "session "+name+": approval requested")
	} else if st.Waiting != "approval" {
		ws.notifiedWait = ""
	}
	// AskUserQuestion picker — the user has to choose.
	if len(st.Choice) > 0 && !ws.notifiedChoice {
		ws.notifiedChoice = true
		s.events.broadcast("session_choice", name, "session "+name+": "+choiceQuestion(st.Choice))
	} else if len(st.Choice) == 0 {
		ws.notifiedChoice = false
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

func choiceQuestion(raw json.RawMessage) string {
	var c struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal(raw, &c); err != nil || c.Question == "" {
		return "choose an option"
	}
	return c.Question
}
