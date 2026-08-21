package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// eventHub fans out audit events to the browser EventSource subscribers.
// Slow or dead clients are dropped without blocking the writer.
type eventHub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[chan []byte]struct{})}
}

func (h *eventHub) broadcast(action, user, detail string) {
	data, err := json.Marshal(map[string]any{
		"time":   time.Now().Format(time.RFC3339),
		"action": action,
		"user":   user,
		"detail": detail,
	})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- data:
		default: // buffer full: drop, keep the rest moving
		}
	}
}

func (h *eventHub) subscribe() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *eventHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *eventHub) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// handleEvents streams audit events as server-sent events. The browser
// subscribes with EventSource and shows a web notification when the tab is
// hidden, so an action on another device surfaces even without polling.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, _ := w.(http.Flusher)
	rc := http.NewResponseController(w)
	ch := s.events.subscribe()
	defer func() {
		s.events.unsubscribe(ch)
		log.Printf("sse: subscriber disconnected (%d active)", s.events.count())
	}()
	log.Printf("sse: subscriber connected (%d active)", s.events.count())

	// A short deadline around each write unblocks the loop when the client
	// vanishes; it's cleared right after so idle periods aren't bounded.
	write := func(s string) bool {
		rc.SetWriteDeadline(time.Now().Add(5 * time.Second))
		fmt.Fprint(w, s)
		if flusher != nil {
			flusher.Flush()
		}
		rc.SetWriteDeadline(time.Time{})
		return true
	}

	write("event: open\ndata: {}\n\n")

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-ch:
			write(fmt.Sprintf("data: %s\n\n", data))
		case <-ticker.C:
			write(": ping\n\n")
		case <-r.Context().Done():
			return
		}
	}
}
