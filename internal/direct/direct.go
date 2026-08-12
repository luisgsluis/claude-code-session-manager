// Package direct lets the ccsm web server run tmux/claude commands in-process,
// without a ccsm-agent or Unix socket. This is the package deployment mode:
// one binary on the host, no separate agent daemon.
package direct

import (
	"encoding/json"

	"github.com/luisgsluis/claude-code-session-manager/internal/agent"
	"github.com/luisgsluis/claude-code-session-manager/internal/host"
)

// Executor implements the handlers' Agent interface by executing commands in
// this process. It mirrors the agent's HTTP response shape so handlers are
// identical in both modes.
type Executor struct {
	h *host.Host
}

// New returns an Executor backed by the given Host.
func New(h *host.Host) *Executor {
	return &Executor{h: h}
}

// Exec validates and runs a command in-process. On failure it returns the host
// error, a *host.Error carrying the HTTP status the handler should return.
func (e *Executor) Exec(cmd string, args map[string]string) (*agent.Response, error) {
	data, err := e.h.Exec(cmd, args)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &agent.Response{OK: true, Data: raw}, nil
}
