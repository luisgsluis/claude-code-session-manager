package agent

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"testing"
)

func TestClientError(t *testing.T) {
	e := &ClientError{Status: 404, Msg: "conversation not found"}
	if e.Error() != "conversation not found" {
		t.Errorf("Error(): %q", e.Error())
	}
	// The handler maps the status via errors.As; assert it's an error type.
	var err error = e
	if err == nil {
		t.Fatal("ClientError must satisfy error")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("/tmp/test.sock", "secret")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.socket != "/tmp/test.sock" {
		t.Errorf("socket: %s", c.socket)
	}
	if c.secret != "secret" {
		t.Errorf("secret: %s", c.secret)
	}
}

func TestExecNoSocket(t *testing.T) {
	c := NewClient("/nonexistent/path/test.sock", "secret")
	_, err := c.Exec("tmux-ls", nil)
	if err == nil {
		t.Error("expected error for nonexistent socket")
	}
}

func TestExecWithMockAgent(t *testing.T) {
	sockPath := t.TempDir() + "/agent.sock"
	secret := "test-secret-123"

	// Start a mock agent
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{OK: false, Error: "bad json"})
			return
		}
		if req.Secret != secret {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{OK: false, Error: "unauthorized"})
			return
		}
		// Return data based on command
		data := json.RawMessage(`[{"name":"0","created":"now","task":"testing"}]`)
		json.NewEncoder(w).Encode(Response{OK: true, Data: data})
	})

	go http.Serve(listener, mux)

	c := NewClient(sockPath, secret)

	// Test successful call
	resp, err := c.Exec("tmux-ls", nil)
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK")
	}
	var sessions []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &sessions); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(sessions) != 1 || sessions[0]["name"] != "0" {
		t.Errorf("unexpected data: %v", sessions)
	}
}

func TestExecWithWrongSecret(t *testing.T) {
	sockPath := t.TempDir() + "/agent.sock"
	secret := "correct-secret"

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		if req.Secret != secret {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{OK: false, Error: "unauthorized"})
			return
		}
	})

	go http.Serve(listener, mux)

	c := NewClient(sockPath, "wrong-secret")
	_, err = c.Exec("tmux-ls", nil)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestExecWithArgs(t *testing.T) {
	sockPath := t.TempDir() + "/agent.sock"
	secret := "s3cr3t"

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	var receivedCmd string
	var receivedArgs map[string]string

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		receivedCmd = req.Cmd
		receivedArgs = req.Args
		if req.Secret != secret {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(Response{OK: false, Error: "unauthorized"})
			return
		}
		json.NewEncoder(w).Encode(Response{OK: true, Data: json.RawMessage(`"ok"`)})
	})

	go http.Serve(listener, mux)

	c := NewClient(sockPath, secret)
	args := map[string]string{"name": "test-session", "profile": "deepseek"}
	_, err = c.Exec("tmux-kill", args)
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}

	if receivedCmd != "tmux-kill" {
		t.Errorf("cmd: %s", receivedCmd)
	}
	if receivedArgs["name"] != "test-session" {
		t.Errorf("args['name']: %s", receivedArgs["name"])
	}
	if receivedArgs["profile"] != "deepseek" {
		t.Errorf("args['profile']: %s", receivedArgs["profile"])
	}
}

func TestExecNon200Response(t *testing.T) {
	sockPath := t.TempDir() + "/agent.sock"

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})

	go http.Serve(listener, mux)

	c := NewClient(sockPath, "secret")
	_, err = c.Exec("tmux-ls", nil)
	if err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestExecBadJSON(t *testing.T) {
	sockPath := t.TempDir() + "/agent.sock"

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	})

	go http.Serve(listener, mux)

	c := NewClient(sockPath, "secret")
	_, err = c.Exec("tmux-ls", nil)
	if err == nil {
		t.Error("expected parse error for bad JSON")
	}
}

func TestExecAgentError(t *testing.T) {
	sockPath := t.TempDir() + "/agent.sock"

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Response{OK: false, Error: "something failed"})
	})

	go http.Serve(listener, mux)

	c := NewClient(sockPath, "secret")
	_, err = c.Exec("tmux-ls", nil)
	if err == nil {
		t.Error("expected error for agent error response")
	}
}

func TestExecBodyReadError(t *testing.T) {
	sockPath := t.TempDir() + "/agent.sock"

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	// Declare a Content-Length larger than the body actually written; the
	// client's io.ReadAll then fails with unexpected EOF.
	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short"))
	})
	go http.Serve(listener, mux)

	c := NewClient(sockPath, "secret")
	_, err = c.Exec("tmux-ls", nil)
	if err == nil {
		t.Error("expected read error for truncated body")
	}
}

func TestCleanupOldSocket(t *testing.T) {
	// Just verify that the Client struct is properly initialized
	sockPath := t.TempDir() + "/cleanup.sock"
	// Create a stale file
	os.WriteFile(sockPath, []byte("stale"), 0644)

	// NewClient should not touch the filesystem
	c := NewClient(sockPath, "secret")
	if c.socket != sockPath {
		t.Errorf("socket mismatch")
	}
}
