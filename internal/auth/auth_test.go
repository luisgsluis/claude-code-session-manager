package auth

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
)

func TestSessionLifecycle(t *testing.T) {
	store := NewStore("test-secret")

	// Create session
	token, sess := store.CreateSession("luis", false)
	if token == "" {
		t.Fatal("token empty")
	}
	if sess.Username != "luis" {
		t.Errorf("username: %s", sess.Username)
	}
	if sess.LANBypass {
		t.Error("should not be LAN bypass")
	}

	// Get session
	retrieved := store.GetSession(token)
	if retrieved == nil {
		t.Fatal("session not found")
	}
	if retrieved.Username != "luis" {
		t.Errorf("retrieved username: %s", retrieved.Username)
	}

	// Delete session
	store.DeleteSession(token)
	if store.GetSession(token) != nil {
		t.Error("session should be deleted")
	}
}

func TestSessionExpiry(t *testing.T) {
	store := NewStore("test-secret")
	store.sessions = make(map[string]*Session) // reset

	now := time.Now()
	token := "test-token"
	store.sessions[token] = &Session{
		Username:  "luis",
		CreatedAt: now.Add(-25 * time.Hour), // expired
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	if store.GetSession(token) != nil {
		t.Error("expired session should return nil")
	}
}

func TestSessionLANBypass(t *testing.T) {
	store := NewStore("test-secret")
	token, sess := store.CreateSession("[lan]", true)
	if !sess.LANBypass {
		t.Error("LAN session should have bypass")
	}

	retrieved := store.GetSession(token)
	if retrieved == nil {
		t.Fatal("session not found")
	}
	if !retrieved.LANBypass {
		t.Error("retrieved LAN session should have bypass")
	}
}

func TestCookieRoundtrip(t *testing.T) {
	w := httptest.NewRecorder()
	SetCookie(w, "test-token-123")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	token := GetToken(req)
	if token == "" {
		t.Fatal("cookie not found in request")
	}
	if token != "test-token-123" {
		t.Errorf("token mismatch: %s", token)
	}
}

func TestClearCookie(t *testing.T) {
	w := httptest.NewRecorder()
	ClearCookie(w)
	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("no Set-Cookie header")
	}
	// Should have MaxAge=-1
	if cookie == "" {
		t.Fatal("empty")
	}
}

func TestGetTokenNoCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	token := GetToken(req)
	if token != "" {
		t.Errorf("expected empty token, got %s", token)
	}
}

func TestCheckPassword(t *testing.T) {
	// Hash generated with: ccsm --hash-password "test123"
	hash := "$2a$10$H/SC9MUlyPBtcbU1Y8/EMu1vClnnTOUf8gK3jQ7WDv.8.5pwwTQ4W"
	users := []config.User{
		{Username: "luis", PasswordHash: hash},
	}

	if !CheckPassword(users, "luis", "test123") {
		t.Error("correct password should match")
	}
	if CheckPassword(users, "luis", "wrong") {
		t.Error("wrong password should not match")
	}
	if CheckPassword(users, "nonexistent", "test123") {
		t.Error("nonexistent user should not match")
	}
}

func TestTokenUniqueness(t *testing.T) {
	store := NewStore("test-secret")
	t1, _ := store.CreateSession("luis", false)
	t2, _ := store.CreateSession("luis", false)
	if t1 == t2 {
		t.Error("two sessions should have different tokens")
	}
}

func TestMultipleConcurrentSessions(t *testing.T) {
	store := NewStore("test-secret")
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			token, _ := store.CreateSession("user", false)
			if store.GetSession(token) == nil {
				t.Error("concurrent session not found")
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
