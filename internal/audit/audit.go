// Package audit records CCSM actions (session created/killed, profile applied,
// login, user management) as JSONL so operations have a history. The file is
// written with O_APPEND, so the ccsm server and ccsm-agent can log to the same
// file concurrently without interleaving.
package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry is one recorded action.
type Entry struct {
	Time   time.Time `json:"time"`
	Action string    `json:"action"`
	User   string    `json:"user,omitempty"`
	Detail string    `json:"detail,omitempty"`
}

// Logger appends JSONL entries to a file.
type Logger struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

// Open appends to path, creating the file (and its directory) if needed.
func Open(path string) (*Logger, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	return &Logger{path: path, f: f}, nil
}

// Log appends one entry. A nil Logger is a no-op.
func (l *Logger) Log(action, user, detail string) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	line, err := json.Marshal(Entry{Time: time.Now(), Action: action, User: user, Detail: detail})
	if err != nil {
		return err
	}
	if _, err := l.f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// Read returns up to n entries from the file, most recent first. A nil Logger
// or a missing file yields an empty slice. Takes the same lock as Log so a
// concurrent write can never be observed as a partially-written trailing line.
func (l *Logger) Read(n int) ([]Entry, error) {
	if l == nil {
		return []Entry{}, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return ReadFile(l.path, n)
}

// ReadFile reads up to n entries from a JSONL file, most recent first.
func ReadFile(path string, n int) ([]Entry, error) {
	if n < 1 {
		n = 1
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	entries := make([]Entry, 0, n)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Entry
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		entries = append(entries, e)
	}
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// Close closes the underlying file.
func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}
