package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Client communicates with the ccsm-agent over a Unix socket.
type Client struct {
	socket string
	secret string
	client *http.Client
}

// ClientError is an agent-side error carrying the HTTP status the agent chose
// (400 invalid input, 404 missing resource, 500 execution failure). The web
// server maps it to the same status so callers see a meaningful code and a
// clean message instead of a blanket 502.
type ClientError struct {
	Status int
	Msg    string
}

func (e *ClientError) Error() string { return e.Msg }

// NewClient creates a new agent client.
func NewClient(socket, secret string) *Client {
	return &Client{
		socket: socket,
		secret: secret,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.DialTimeout("unix", socket, 2*time.Second)
				},
			},
			Timeout: 35 * time.Second, // RC bootstrap can take up to 25s
		},
	}
}

// Request is a command to the agent.
type Request struct {
	Cmd    string            `json:"cmd"`
	Args   map[string]string `json:"args,omitempty"`
	Secret string            `json:"secret"`
}

// Response is the agent's reply.
type Response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Exec sends a command to the agent and returns the parsed response.
func (c *Client) Exec(cmd string, args map[string]string) (*Response, error) {
	reqBody := Request{
		Cmd:    cmd,
		Args:   args,
		Secret: c.secret,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.client.Post(
		"http://unix/exec",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("agent call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Pull the clean `error` field out of {"ok":false,"error":"…"} so the
		// caller sees the message, not raw JSON.
		var r Response
		msg := strings.TrimSpace(string(respBody))
		if json.Unmarshal(respBody, &r) == nil && r.Error != "" {
			msg = r.Error
		}
		return nil, &ClientError{Status: resp.StatusCode, Msg: msg}
	}

	var r Response
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if !r.OK {
		return nil, &ClientError{Status: http.StatusInternalServerError, Msg: r.Error}
	}

	return &r, nil
}
