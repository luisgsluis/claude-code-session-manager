package handlers

import (
	"errors"
	"net/http"

	"github.com/luisgsluis/claude-code-session-manager/internal/agent"
)

// Agent executes commands either over the agent socket (container deployment)
// or in-process (package deployment). Both ccsm-agent and direct mode return
// the same response shape, so handlers are identical in either deployment.
type Agent interface {
	Exec(cmd string, args map[string]string) (*agent.Response, error)
}

// writeAgentError maps an executor error to a meaningful HTTP response. Errors
// that carry a status (invalid input, missing resource, execution failure) keep
// their status and clean message; transport errors (agent unreachable) become a
// 502 with a hint about the backend being unavailable.
func writeAgentError(w http.ResponseWriter, err error) {
	var ce *agent.ClientError
	if errors.As(err, &ce) && ce.Status > 0 {
		writeError(w, ce.Status, ce.Msg)
		return
	}
	writeError(w, http.StatusBadGateway, "backend unavailable: "+err.Error())
}
