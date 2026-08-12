// ccsm-agent listens on a Unix socket and executes tmux/claude commands on the
// host. It is meant to be run directly on the host, managed by systemd or
// similar. It refuses to start a second instance on the same socket.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/host"
)

// Request is a command from the web server.
type Request struct {
	Cmd    string            `json:"cmd"`
	Args   map[string]string `json:"args,omitempty"`
	Secret string            `json:"secret"`
}

// Response is the agent's reply.
type Response struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func main() {
	socket := flag.String("socket", "/run/ccsm/agent.sock", "Unix socket path")
	secret := flag.String("secret", "", "shared secret for authentication (only when --secret-file is not used)")
	secretFile := flag.String("secret-file", "", "path to a file containing the shared secret (0600 recommended); keeps the secret out of process args and unit files")
	profiles := flag.String("profiles", os.Getenv("HOME")+"/claude-shared/claude-perfiles", "profiles directory")
	settings := flag.String("settings", os.Getenv("HOME")+"/.claude/settings.json", "settings.json path")
	conversations := flag.String("conversations", os.Getenv("HOME")+"/.claude/projects/-home-admin", "conversations directory")
	claudeBin := flag.String("claude", os.Getenv("HOME")+"/.local/bin/claude", "claude binary")
	tmuxBin := flag.String("tmux", "/usr/bin/tmux", "tmux binary")
	bashBin := flag.String("bash", "/usr/bin/bash", "bash binary")
	rcProfile := flag.String("rc-profile", "estandar", "bootstrap profile for RC")
	rcWait := flag.Int("rc-wait", 25, "seconds to wait for RC connect")
	rcPoll := flag.Int("rc-poll", 2, "seconds between RC polls")
	rcSettle := flag.Int("rc-settle", 1, "confirmation margin (s) after the resume process is idle+bridge before restoring the target profile")
	flag.Parse()

	secretValue, err := resolveSecret(*secretFile, *secret)
	if err != nil {
		log.Fatal(err)
	}

	h := host.New(host.Options{
		ProfilesPath:  *profiles,
		SettingsPath:  *settings,
		ConvPath:      *conversations,
		ClaudeBinary:  *claudeBin,
		TmuxBinary:    *tmuxBin,
		BashBinary:    *bashBin,
		RcBootstrap:   *rcProfile,
		RcWaitSeconds:   *rcWait,
		RcPollSeconds:   *rcPoll,
		RcSettleSeconds: *rcSettle,
	})

	if err := ensureSingle(*socket); err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("unix", *socket)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	// Restrict access to socket owner
	if err := os.Chmod(*socket, 0600); err != nil {
		log.Printf("chmod socket: %v", err)
	}

	log.Printf("ccsm-agent listening on %s", *socket)

	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		handleExec(w, r, secretValue, h)
	})
	mux.HandleFunc("/health", handleHealth)

	if err := http.Serve(listener, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// resolveSecret returns the shared secret to use. If --secret-file is given it
// reads the file (whitespace-trimmed) — this keeps the secret out of `ps aux`
// and out of the systemd unit. Otherwise it falls back to --secret. Both empty
// is an error. The file takes precedence so deployments never leak the secret
// in process arguments.
func resolveSecret(secretFile, secret string) (string, error) {
	if secretFile != "" {
		b, err := os.ReadFile(secretFile)
		if err != nil {
			return "", fmt.Errorf("read secret file: %w", err)
		}
		s := strings.TrimSpace(string(b))
		if s == "" {
			return "", errors.New("secret file is empty")
		}
		return s, nil
	}
	if secret == "" {
		return "", errors.New("--secret (or --secret-file) is required")
	}
	return secret, nil
}

// ensureSingle guarantees the agent is only deployed once: if a live listener
// already owns the socket, refuse to start. A stale socket file (no listener)
// is removed so a fresh instance can bind.
func ensureSingle(socket string) error {
	if conn, err := net.DialTimeout("unix", socket, time.Second); err == nil {
		conn.Close()
		return &ErrAlreadyRunning{Socket: socket}
	}
	// Socket exists but nothing is listening → stale, remove it.
	os.Remove(socket)
	return nil
}

// ErrAlreadyRunning reports a duplicate agent deployment.
type ErrAlreadyRunning struct{ Socket string }

func (e *ErrAlreadyRunning) Error() string {
	return "another ccsm-agent is already listening on " + e.Socket + "; refusing to run a second instance"
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, Response{OK: true, Data: "agent ok"})
}

func handleExec(w http.ResponseWriter, r *http.Request, secret string, h *host.Host) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST required"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "read body"})
		return
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "invalid JSON"})
		return
	}

	if req.Secret != secret {
		writeJSON(w, http.StatusForbidden, Response{OK: false, Error: "unauthorized"})
		return
	}

	data, err := h.Exec(req.Cmd, req.Args)
	if err != nil {
		status := http.StatusInternalServerError
		var he *host.Error
		if errors.As(err, &he) && he.Status > 0 {
			status = he.Status
		}
		writeJSON(w, status, Response{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{OK: true, Data: data})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
