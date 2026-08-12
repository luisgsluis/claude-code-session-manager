package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// Session holds authenticated user state.
type Session struct {
	Username  string    `json:"username"`
	LANBypass bool      `json:"lan_bypass"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Store manages active sessions in memory.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	secret   string
}

// NewStore creates a session store.
func NewStore(secret string) *Store {
	return &Store{
		sessions: make(map[string]*Session),
		secret:   secret,
	}
}

const (
	sessionCookie = "ccsm_session"
	sessionTTL    = 24 * time.Hour
)

// CreateSession generates a session token and stores the session.
func (s *Store) CreateSession(username string, lanBypass bool) (string, *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	sess := &Session{
		Username:  username,
		LANBypass: lanBypass,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	}

	token := s.generateToken(sess)
	s.sessions[token] = sess
	return token, sess
}

// GetSession returns the session for a token, or nil if expired/invalid.
func (s *Store) GetSession(token string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[token]
	if !ok {
		return nil
	}
	if time.Now().After(sess.ExpiresAt) {
		delete(s.sessions, token)
		return nil
	}
	return sess
}

// DeleteSession removes a session.
func (s *Store) DeleteSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *Store) generateToken(sess *Session) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(sess.Username))
	mac.Write([]byte(sess.CreatedAt.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

// SetCookie writes the session cookie to the response.
func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// ClearCookie removes the session cookie.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// GetToken extracts the session token from the request cookie.
func GetToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

// CheckPassword verifies a password against a user list using bcrypt.
func CheckPassword(users []config.User, username, password string) bool {
	for _, u := range users {
		if u.Username != username {
			continue
		}
		return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
	}
	return false
}
