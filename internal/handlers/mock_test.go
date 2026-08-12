package handlers

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/luisgsluis/claude-code-session-manager/internal/agent"
)

// mockAgentServer starts a fake ccsm-agent on a Unix socket.
// Returns the socket path and a cleanup function.
func mockAgentServer(t *testing.T, handler http.HandlerFunc) (string, func()) {
	t.Helper()

	sockPath := t.TempDir() + "/agent.sock"
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	go http.Serve(listener, mux)

	return sockPath, func() {
		listener.Close()
	}
}

// mockAgentOK returns an agent handler that responds OK with the given JSON data.
func mockAgentOK(data string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"data":` + data + `}`))
	}
}

// mockAgentError returns an agent handler that responds with an error.
func mockAgentError(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": msg,
		})
	}
}

// mockAgentRecorder returns a handler that records the received request body
// and echoes back the args as the data response, so tests can assert what was
// forwarded to the agent.
func mockAgentRecorder() (http.HandlerFunc, func() string) {
	var mu sync.Mutex
	var lastBody string
	return func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			lastBody = string(body)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Echo the args as the data
			w.Write([]byte(`{"ok":true,"data":{"args":` + string(body) + `}}`))
		}, func() string {
			mu.Lock()
			defer mu.Unlock()
			return lastBody
		}
}

// requireAgent creates an agent client connected to the mock server.
// We use the real agent.Client but pointed at our mock socket.
func requireAgent(t *testing.T, sockPath string) *agent.Client {
	t.Helper()
	return agent.NewClient(sockPath, "")
}
