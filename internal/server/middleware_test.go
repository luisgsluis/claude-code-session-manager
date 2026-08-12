package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeFlusher is a ResponseWriter that also supports Flush and SetWriteDeadline,
// mirroring the real http response writer. It lets us assert the logging wrapper
// promotes both to handlers.
type fakeFlusher struct {
	httptest.ResponseRecorder
	flushed   bool
	deadlined bool
}

func (f *fakeFlusher) Flush() { f.flushed = true }

func (f *fakeFlusher) SetWriteDeadline(time.Time) error {
	f.deadlined = true
	return nil
}

// TestMiddlewarePreservesFlusher guards the SSE live view: the logging wrapper
// must not hide http.Flusher from handlers (regression: stream returned 500
// "streaming unsupported" in production because w.(http.Flusher) failed).
func TestMiddlewarePreservesFlusher(t *testing.T) {
	h := withLogging(withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Errorf("handler sees a writer that is not http.Flusher")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: hola\n\n"))
		f.Flush()
		// NewResponseController must reach deadline controls on the real writer
		// through Unwrap.
		rc := http.NewResponseController(w)
		if err := rc.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Errorf("SetWriteDeadline: %v", err)
		}
	})))

	fw := &fakeFlusher{ResponseRecorder: *httptest.NewRecorder()}
	h.ServeHTTP(fw, httptest.NewRequest("GET", "/api/sessions/0/stream", nil))

	if fw.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", fw.Code)
	}
	if ct := fw.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}
	if !strings.Contains(fw.Body.String(), "data: hola") {
		t.Errorf("body = %q, want SSE data line", fw.Body.String())
	}
	if !fw.flushed {
		t.Error("Flush() did not reach the underlying writer")
	}
	if !fw.deadlined {
		t.Error("SetWriteDeadline did not reach the underlying writer")
	}
}
