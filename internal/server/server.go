package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/luisgsluis/claude-code-session-manager/internal/agent"
	"github.com/luisgsluis/claude-code-session-manager/internal/audit"
	"github.com/luisgsluis/claude-code-session-manager/internal/auth"
	"github.com/luisgsluis/claude-code-session-manager/internal/config"
	"github.com/luisgsluis/claude-code-session-manager/internal/direct"
	"github.com/luisgsluis/claude-code-session-manager/internal/handlers"
	"github.com/luisgsluis/claude-code-session-manager/internal/host"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// Server is the CCSM HTTP server.
type Server struct {
	cfg        *config.Config
	cfgMu      sync.RWMutex // guards the mutable subset of cfg: Users, LANSubnets, HostAttachAddr, Rc
	configPath string
	http       *http.Server
	mux        *http.ServeMux
	uptime     time.Time
	sessions   *auth.Store
	agentCli   *agent.Client
	staticPath string
	exec       handlers.Agent
	audit      *audit.Logger
	auditPath  string
	events     *eventHub

	// Authentication extras
	loginRL    *auth.RateLimiter // shared by the password and the TOTP step
	totpReplay *auth.ReplayGuard
	enrollMu   sync.Mutex
	enroll     map[string]pendingEnrollment // secrets generated but not confirmed yet

	// Handlers
	sessionHdlr      *handlers.SessionHandler
	profileHdlr      *handlers.ProfileHandler
	conversationHdlr *handlers.ConversationHandler
}

// New creates a new Server. staticPath is the path to the static/ directory;
// configPath is the path to config.yaml (used for write-back on PATCH /api/config).
// The executor is chosen from the config: an empty agent_socket selects direct
// mode (package deployment, commands run in-process), a socket path selects
// agent mode (container deployment, commands go to ccsm-agent over the socket).
func New(cfg *config.Config, staticPath, configPath string) *Server {
	s := &Server{
		cfg:        cfg,
		configPath: configPath,
		uptime:     time.Now(),
		events:     newEventHub(),
		sessions:   auth.NewStore(cfg.SessionSecret),
		staticPath: staticPath,
		loginRL:    auth.NewLoginRateLimiter(),
		totpReplay: auth.NewReplayGuard(),
		enroll:     make(map[string]pendingEnrollment),
	}

	auditPath := cfg.AuditPath
	if auditPath == "" || auditPath == "auto" {
		auditPath = os.Getenv("HOME") + "/.ccsm/audit.jsonl"
	}
	s.auditPath = auditPath
	if l, err := audit.Open(auditPath); err == nil {
		s.audit = l
	} else {
		log.Printf("audit log disabled: %v", err)
	}
	auditLog := s.auditLog

	var executor handlers.Agent
	if cfg.AgentSocket == "" {
		s.agentCli = nil
		h := host.New(host.Options{
			ProfilesPath:    cfg.ProfilesPath,
			SettingsPath:    cfg.SettingsPath,
			ConvPath:        cfg.ConversationsPath,
			ClaudeBinary:    cfg.ClaudeBinary,
			TmuxBinary:      cfg.TmuxBinary,
			BashBinary:      cfg.BashBinary,
			RcBootstrap:     cfg.Rc.BootstrapProfile,
			RcWaitSeconds:   cfg.Rc.WaitSeconds,
			RcPollSeconds:   cfg.Rc.PollSeconds,
			RcSettleSeconds: cfg.Rc.SettleSeconds,
		})
		executor = direct.New(h)
	} else {
		s.agentCli = agent.NewClient(cfg.AgentSocket, cfg.AgentSecret)
		executor = s.agentCli
	}
	// Always keep the executor: the turn watcher (turn_complete / approval /
	// choice events) uses it in both deployment modes.
	s.exec = executor

	s.sessionHdlr = &handlers.SessionHandler{
		Agent:      executor,
		AttachAddr: cfg.HostAttachAddr,
		Audit:      auditLog,
	}
	s.profileHdlr = &handlers.ProfileHandler{
		Agent: executor,
		Audit: auditLog,
	}
	s.conversationHdlr = &handlers.ConversationHandler{
		Agent: executor,
	}

	s.mux = http.NewServeMux()
	s.registerRoutes()
	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      withLogging(withSecurityHeaders(s.mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second, // session-rc can run two RC bootstraps (staging + auto-recover resume)
		IdleTimeout:  120 * time.Second,
	}
	return s
}

func (s *Server) registerRoutes() {
	// Health (no auth)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// Auth
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/totp", s.handleTOTP)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/auth/status", s.handleAuthStatus)

	// Static files and SPA (Phase 5)
	s.mux.HandleFunc("GET /", s.handleSPA)
	if s.staticPath != "" {
		fs := http.FileServer(http.Dir(s.staticPath))
		// Revalidate on every load (304 via Last-Modified) so the browser never
		// keeps a stale app.js after a deploy.
		s.mux.Handle("GET /static/", http.StripPrefix("/static/", noCache(fs)))
	}

	// Sessions (protected)
	s.mux.HandleFunc("GET /api/sessions", s.auth(s.sessionHdlr.ListSessions))
	s.mux.HandleFunc("DELETE /api/sessions/{name}", s.auth(s.sessionHdlr.KillSession))
	s.mux.HandleFunc("POST /api/sessions/new", s.auth(s.sessionHdlr.NewSession))
	s.mux.HandleFunc("POST /api/sessions/resume", s.auth(s.sessionHdlr.ResumeSession))
	s.mux.HandleFunc("POST /api/sessions/{name}/rename", s.auth(s.sessionHdlr.RenameSession))
	s.mux.HandleFunc("POST /api/sessions/{name}/claude-name", s.auth(s.sessionHdlr.SetClaudeName))
	s.mux.HandleFunc("GET /api/sessions/{name}/stream", s.auth(s.sessionHdlr.LiveStream))
	s.mux.HandleFunc("GET /api/sessions/{name}/chat", s.auth(s.sessionHdlr.Chat))
	s.mux.HandleFunc("GET /api/sessions/{name}/chat/stream", s.auth(s.sessionHdlr.ChatStream))
	s.mux.HandleFunc("POST /api/sessions/{name}/send", s.auth(s.sessionHdlr.Send))
	s.mux.HandleFunc("POST /api/sessions/{name}/rc", s.auth(s.sessionHdlr.ReconnectRC))

	// Projects (protected)
	s.mux.HandleFunc("GET /api/projects", s.auth(s.sessionHdlr.ListProjects))

	// Profiles (protected)
	s.mux.HandleFunc("GET /api/profiles", s.auth(s.profileHdlr.ListProfiles))
	s.mux.HandleFunc("POST /api/profiles/apply", s.auth(s.profileHdlr.ApplyProfile))
	s.mux.HandleFunc("GET /api/profiles/{name}", s.auth(s.profileHdlr.GetProfileContent))

	// Conversations (protected)
	s.mux.HandleFunc("GET /api/conversations", s.auth(s.conversationHdlr.ListConversations))
	s.mux.HandleFunc("GET /api/conversations/{id}", s.auth(s.conversationHdlr.GetConversation))
	s.mux.HandleFunc("GET /api/conversations/{id}/export", s.auth(s.conversationHdlr.ExportConversation))
	s.mux.HandleFunc("GET /api/conversations/{id}/meta", s.auth(s.conversationHdlr.GetConversationMeta))
	s.mux.HandleFunc("PUT /api/conversations/{id}/meta", s.auth(s.conversationHdlr.SetConversationMeta))

	// Audit log (protected)
	s.mux.HandleFunc("GET /api/audit", s.auth(s.handleAudit))
	s.mux.HandleFunc("GET /api/metrics", s.auth(s.handleMetrics))
	s.mux.HandleFunc("GET /api/events", s.auth(s.handleEvents))

	// Config (protected): non-secret deployment info for the settings panel.
	// Secret material (session_secret, agent_secret, password_hash) is never
	// exposed. PATCH allows editing hot-reload fields; user management is
	// handled by separate endpoints.
	s.mux.HandleFunc("GET /api/config", s.auth(s.handleConfig))
	s.mux.HandleFunc("PATCH /api/config", s.auth(s.handlePatchConfig))
	s.mux.HandleFunc("GET /api/config/users", s.auth(s.handleListUsers))
	s.mux.HandleFunc("POST /api/config/users", s.auth(s.handleAddUser))
	s.mux.HandleFunc("DELETE /api/config/users/{username}", s.auth(s.handleDeleteUser))
	s.mux.HandleFunc("POST /api/config/users/{username}/password", s.auth(s.handleChangePassword))
	s.mux.HandleFunc("POST /api/config/users/{username}/totp", s.auth(s.handleTOTPEnrollStart))
	s.mux.HandleFunc("PUT /api/config/users/{username}/totp", s.auth(s.handleTOTPEnrollConfirm))
	s.mux.HandleFunc("DELETE /api/config/users/{username}/totp", s.auth(s.handleTOTPDisable))

	// Settings content (protected): raw settings.json for profile viewer
	s.mux.HandleFunc("GET /api/settings", s.auth(s.handleGetSettings))
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// API 404
	if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	// Serve index.html for SPA routes
	w.Header().Set("Cache-Control", "no-cache")
	if s.staticPath != "" {
		http.ServeFile(w, r, s.staticPath+"/index.html")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(minimalHTML))
}

// --- Auth handlers ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.isLANRequest(r) {
		token, _ := s.sessions.CreateSession("[lan]", true)
		auth.SetCookie(w, token, isHTTPS(r))
		s.auditLogIP("login", "[lan]", "lan bypass", s.clientIP(r))
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "lan_bypass": true})
		return
	}

	ip := s.clientIP(r)
	if s.rateLimited(w, ip) {
		return
	}

	var req loginRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	s.cfgMu.RLock()
	ok := auth.CheckPassword(s.cfg.Users, req.Username, req.Password)
	secret := auth.TOTPSecretFor(s.cfg.Users, req.Username)
	s.cfgMu.RUnlock()
	if !ok {
		// The UI probes for the LAN bypass by POSTing empty credentials on
		// every page load (checkAuth in app.js). That is not an attempt, and
		// counting it would let a reloading browser lock itself out.
		if req.Username != "" {
			s.recordAuthFailure(ip, req.Username)
		}
		s.auditLogIP("login_failed", req.Username, "invalid credentials", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	// Second factor: hand out a pending session instead of a real one. The
	// auth middleware rejects it, so nothing is reachable until the code is in.
	if secret != "" {
		token, _ := s.sessions.CreatePendingSession(req.Username)
		auth.SetCookie(w, token, isHTTPS(r))
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": false, "totp_required": true})
		return
	}

	s.loginRL.Reset(ip)
	token, _ := s.sessions.CreateSession(req.Username, false)
	auth.SetCookie(w, token, isHTTPS(r))
	s.auditLogIP("login", req.Username, "", ip)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

type totpRequest struct {
	Code string `json:"code"`
}

// handleTOTP is the second step of a two-factor login: it upgrades the pending
// session issued by handleLogin into a full one. Public like /login — the
// pending cookie is what identifies the caller.
func (s *Server) handleTOTP(w http.ResponseWriter, r *http.Request) {
	ip := s.clientIP(r)

	sess := s.sessions.GetSession(auth.GetToken(r))
	if sess == nil || !sess.TOTPPending {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no pending login"})
		return
	}
	// Checked after the pending-session lookup so a blocked IP still gets a
	// clear 429 rather than a misleading "no pending login".
	if s.rateLimited(w, ip) {
		return
	}

	var req totpRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	s.cfgMu.RLock()
	secret := auth.TOTPSecretFor(s.cfg.Users, sess.Username)
	s.cfgMu.RUnlock()

	step, ok := auth.VerifyTOTP(secret, req.Code, time.Now())
	// A code is valid for its whole 30 s window, so accepting it twice would
	// make a captured code replayable.
	if ok && !s.totpReplay.Use(sess.Username, step) {
		ok = false
	}
	if !ok {
		s.recordAuthFailure(ip, sess.Username)
		s.auditLogIP("login_totp_failed", sess.Username, "invalid code", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid code"})
		return
	}

	s.sessions.DeleteSession(auth.GetToken(r))
	s.loginRL.Reset(ip)
	token, _ := s.sessions.CreateSession(sess.Username, false)
	auth.SetCookie(w, token, isHTTPS(r))
	s.auditLogIP("login", sess.Username, "2fa", ip)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// clientIP is the real client address without its port: the peer, or
// X-Forwarded-For when the peer is a trusted proxy. Same rule the LAN bypass
// uses, so a proxied client cannot forge it. The port must go — it is the
// identity of a connection, not of a client, and keeping it would give every
// attempt its own rate-limit bucket.
func (s *Server) clientIP(r *http.Request) string {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return auth.HostOnly(auth.ClientIP(r, s.cfg.TrustedProxies))
}

// rateLimited answers a blocked client with 429 and reports whether it did.
func (s *Server) rateLimited(w http.ResponseWriter, ip string) bool {
	if !s.loginRL.Blocked(ip) {
		return false
	}
	writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many failed attempts, try again later"})
	return true
}

// recordAuthFailure counts one failed attempt and audits the block exactly
// once, on the transition — not once per attempt, which would flood the log
// with the very traffic it is meant to make readable.
func (s *Server) recordAuthFailure(ip, user string) {
	if s.loginRL.Fail(ip) {
		s.auditLogIP("login_blocked", user, "too many failed attempts", ip)
	}
}

// isHTTPS reports whether the request reached CCSM over TLS, either directly
// or via a reverse proxy that sets X-Forwarded-Proto (Caddy/Nginx do this by
// default). Used to decide the session cookie's Secure attribute: CCSM is
// commonly deployed behind a proxy that terminates TLS, so r.TLS alone (only
// set when this process itself terminates TLS) would under-report it.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (s *Server) auditLog(action, user, detail string) {
	s.auditLogIP(action, user, detail, "")
}

// auditLogIP is auditLog for authentication events, which record the client
// address so an external blocker (the homelab ipban job) can act on them.
func (s *Server) auditLogIP(action, user, detail, ip string) {
	if s.audit != nil {
		s.audit.LogWithIP(action, user, detail, ip)
	}
	s.events.broadcast(action, user, detail)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	user := ""
	if token := auth.GetToken(r); token != "" {
		if sess := s.sessions.GetSession(token); sess != nil {
			user = sess.Username
		}
		s.sessions.DeleteSession(token)
	}
	auth.ClearCookie(w)
	s.auditLogIP("logout", user, "", s.clientIP(r))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	token := auth.GetToken(r)
	if token == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": false})
		return
	}

	sess := s.sessions.GetSession(token)
	if sess == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": false})
		return
	}
	// A reload mid-2FA must land back on the code step, not on the password
	// form — the password has already been accepted.
	if sess.TOTPPending {
		writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": false, "totp_required": true})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"username":      sess.Username,
		"lan_bypass":    sess.LANBypass,
	})
}

// --- Auth middleware ---

// isLANRequest decides the LAN-bypass under a read lock, since LANSubnets is
// mutable via PATCH /api/config while requests are being served concurrently.
func (s *Server) isLANRequest(r *http.Request) bool {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return auth.IsLAN(auth.ClientIP(r, s.cfg.TrustedProxies), s.cfg.LANSubnets)
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.isLANRequest(r) {
			next(w, handlers.WithUser(r, "[lan]"))
			return
		}

		token := auth.GetToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		sess := s.sessions.GetSession(token)
		if sess == nil {
			auth.ClearCookie(w)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "session expired"})
			return
		}
		// Half-way through a 2FA login: password accepted, code still owed.
		// This single guard covers every protected route.
		if sess.TOTPPending {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "totp required"})
			return
		}

		next(w, handlers.WithUser(r, sess.Username))
	}
}

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"uptime":  time.Since(s.uptime).String(),
		"version": Version,
	})
}

// configInfo is the non-secret subset of the deployment config exposed to the
// UI settings panel. Secrets (session_secret, agent_secret, password hashes)
// are deliberately excluded.
type configInfo struct {
	Port           int               `json:"port"`
	Mode           string            `json:"mode"` // "direct" or "agent"
	AgentSocket    string            `json:"agent_socket"`
	HostAttachAddr string            `json:"host_attach_addr"`
	LANSubnets     []string          `json:"lan_subnets"`
	Users          []userInfo        `json:"users"`
	Paths          map[string]string `json:"paths"`
	RC             map[string]any    `json:"rc"`
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	mode := "agent"
	if s.cfg.AgentSocket == "" {
		mode = "direct"
	}

	s.cfgMu.RLock()
	users := make([]userInfo, 0, len(s.cfg.Users))
	for _, u := range s.cfg.Users {
		users = append(users, userInfo{Username: u.Username, TOTP: u.TOTPSecret != ""})
	}
	info := configInfo{
		Port:           s.cfg.Port,
		Mode:           mode,
		AgentSocket:    s.cfg.AgentSocket,
		HostAttachAddr: s.cfg.HostAttachAddr,
		LANSubnets:     s.cfg.LANSubnets,
		Users:          users,
		Paths: map[string]string{
			"conversations": s.cfg.ConversationsPath,
			"profiles":      s.cfg.ProfilesPath,
			"settings":      s.cfg.SettingsPath,
			"claude_binary": s.cfg.ClaudeBinary,
			"tmux_binary":   s.cfg.TmuxBinary,
			"bash_binary":   s.cfg.BashBinary,
		},
		RC: map[string]any{
			"bootstrap_profile": s.cfg.Rc.BootstrapProfile,
			"wait_seconds":      s.cfg.Rc.WaitSeconds,
			"poll_seconds":      s.cfg.Rc.PollSeconds,
		},
	}
	s.cfgMu.RUnlock()
	writeJSON(w, http.StatusOK, info)
}

// --- Config write-back ---

// writeConfig serializes cfg to YAML and writes it atomically to path.
func writeConfig(path string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EBUSY) {
		os.Remove(tmp)
		return err
	}
	// EBUSY: config.yaml is bind-mounted into the container, so the rename
	// cannot replace the mounted inode. Write through the mount in place
	// (keeps the inode, updates the host file).
	f, ferr := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600)
	if ferr != nil {
		os.Remove(tmp)
		return ferr
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	os.Remove(tmp)
	if werr != nil {
		return werr
	}
	return cerr
}

// patchConfigRequest is a partial update: every field is optional.
type patchConfigRequest struct {
	LANSubnets     []string       `json:"lan_subnets"`
	HostAttachAddr *string        `json:"host_attach_addr"`
	Rc             *patchRcConfig `json:"rc"`
}

type patchRcConfig struct {
	BootstrapProfile *string `json:"bootstrap_profile"`
	WaitSeconds      *int    `json:"wait_seconds"`
	PollSeconds      *int    `json:"poll_seconds"`
}

var attachPattern = regexp.MustCompile(`^[a-zA-Z0-9@._-]{1,120}$`)
var bootstrapPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

func (s *Server) handlePatchConfig(w http.ResponseWriter, r *http.Request) {
	var req patchConfigRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	var updated []string
	needsRestart := false

	// LAN subnets: hot-reload
	if req.LANSubnets != nil {
		for _, cidr := range req.LANSubnets {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid CIDR: " + cidr})
				return
			}
		}
		s.cfg.LANSubnets = req.LANSubnets
		updated = append(updated, "lan_subnets")
	}

	// Host attach addr: hot-reload
	if req.HostAttachAddr != nil {
		if !attachPattern.MatchString(*req.HostAttachAddr) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid host_attach_addr"})
			return
		}
		s.cfg.HostAttachAddr = *req.HostAttachAddr
		s.sessionHdlr.AttachAddr = *req.HostAttachAddr
		updated = append(updated, "host_attach_addr")
	}

	// RC settings: write to cfg + host struct
	if req.Rc != nil {
		if req.Rc.BootstrapProfile != nil {
			if !bootstrapPattern.MatchString(*req.Rc.BootstrapProfile) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid bootstrap_profile"})
				return
			}
			s.cfg.Rc.BootstrapProfile = *req.Rc.BootstrapProfile
			updated = append(updated, "rc.bootstrap_profile")
		}
		if req.Rc.WaitSeconds != nil {
			if *req.Rc.WaitSeconds < 5 || *req.Rc.WaitSeconds > 120 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rc.wait_seconds must be 5-120"})
				return
			}
			s.cfg.Rc.WaitSeconds = *req.Rc.WaitSeconds
			updated = append(updated, "rc.wait_seconds")
		}
		if req.Rc.PollSeconds != nil {
			if *req.Rc.PollSeconds < 1 || *req.Rc.PollSeconds > 10 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rc.poll_seconds must be 1-10"})
				return
			}
			s.cfg.Rc.PollSeconds = *req.Rc.PollSeconds
			updated = append(updated, "rc.poll_seconds")
		}
	}

	if len(updated) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no valid fields to update"})
		return
	}

	if err := writeConfig(s.configPath, s.cfg); err != nil {
		log.Printf("write config: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist config"})
		return
	}

	s.auditLog("config_update", handlers.UserFrom(r), "fields="+strings.Join(updated, ","))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":             true,
		"updated":        updated,
		"restart_needed": needsRestart,
	})
}

// --- User management ---

type addUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	Password string `json:"password"`
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// userInfo is what the API exposes about a user: the name and whether a second
// factor is enrolled. Never the password hash, never the TOTP secret.
type userInfo struct {
	Username string `json:"username"`
	TOTP     bool   `json:"totp"`
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	s.cfgMu.RLock()
	users := make([]userInfo, 0, len(s.cfg.Users))
	for _, u := range s.cfg.Users {
		users = append(users, userInfo{Username: u.Username, TOTP: u.TOTPSecret != ""})
	}
	s.cfgMu.RUnlock()
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleAddUser(w http.ResponseWriter, r *http.Request) {
	var req addUserRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if !usernamePattern.MatchString(req.Username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid username (a-z, 0-9, _-, 1-32 chars)"})
		return
	}
	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	for _, u := range s.cfg.Users {
		if u.Username == req.Username {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "user already exists: " + req.Username})
			return
		}
	}
	s.cfg.Users = append(s.cfg.Users, config.User{Username: req.Username, PasswordHash: string(hash)})
	if err := writeConfig(s.configPath, s.cfg); err != nil {
		log.Printf("write config: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist config"})
		return
	}
	s.auditLog("user_add", handlers.UserFrom(r), "user="+req.Username)
	writeJSON(w, http.StatusCreated, map[string]string{"ok": "user " + req.Username + " created"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing username"})
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	found := -1
	for i, u := range s.cfg.Users {
		if u.Username == username {
			found = i
			break
		}
	}
	if found < 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found: " + username})
		return
	}
	if len(s.cfg.Users) <= 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot delete the last user"})
		return
	}
	s.cfg.Users = append(s.cfg.Users[:found], s.cfg.Users[found+1:]...)
	if err := writeConfig(s.configPath, s.cfg); err != nil {
		log.Printf("write config: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist config"})
		return
	}
	s.auditLog("user_delete", handlers.UserFrom(r), "user="+username)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "user " + username + " deleted"})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing username"})
		return
	}
	var req changePasswordRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	found := -1
	for i, u := range s.cfg.Users {
		if u.Username == username {
			found = i
			break
		}
	}
	if found < 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found: " + username})
		return
	}
	s.cfg.Users[found].PasswordHash = string(hash)
	if err := writeConfig(s.configPath, s.cfg); err != nil {
		log.Printf("write config: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist config"})
		return
	}
	s.auditLog("user_password", handlers.UserFrom(r), "user="+username)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "password changed for " + username})
}

// --- Two-factor authentication (TOTP) ---

// pendingEnrollment is a secret handed to the browser but not yet written to
// config.yaml. It only becomes the user's real secret once they prove they can
// generate a code from it — persisting on generation instead would lock out
// anyone whose scan silently failed.
type pendingEnrollment struct {
	secret  string
	created time.Time
}

// enrollTTL bounds how long a generated-but-unconfirmed secret stays usable.
const enrollTTL = 10 * time.Minute

// handleTOTPEnrollStart generates a secret and returns it, once, together with
// the otpauth:// URI the UI renders as a QR.
func (s *Server) handleTOTPEnrollStart(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if !s.userExists(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found: " + username})
		return
	}

	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		log.Printf("totp secret: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate secret"})
		return
	}

	s.enrollMu.Lock()
	s.enroll[username] = pendingEnrollment{secret: secret, created: time.Now()}
	s.enrollMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"secret": secret,
		"uri":    auth.TOTPURI(username, secret),
	})
}

// handleTOTPEnrollConfirm verifies a code against the pending secret and only
// then persists it.
func (s *Server) handleTOTPEnrollConfirm(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	var req totpRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	s.enrollMu.Lock()
	pending, ok := s.enroll[username]
	if ok && time.Since(pending.created) > enrollTTL {
		delete(s.enroll, username)
		ok = false
	}
	s.enrollMu.Unlock()
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no pending enrollment, start again"})
		return
	}

	step, valid := auth.VerifyTOTP(pending.secret, req.Code, time.Now())
	if !valid {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid code"})
		return
	}

	s.cfgMu.Lock()
	found := -1
	for i, u := range s.cfg.Users {
		if u.Username == username {
			found = i
			break
		}
	}
	if found < 0 {
		s.cfgMu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found: " + username})
		return
	}
	s.cfg.Users[found].TOTPSecret = pending.secret
	err := writeConfig(s.configPath, s.cfg)
	s.cfgMu.Unlock()
	if err != nil {
		log.Printf("write config: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist config"})
		return
	}

	s.enrollMu.Lock()
	delete(s.enroll, username)
	s.enrollMu.Unlock()
	// The confirming code must not be reusable to log in with.
	s.totpReplay.Use(username, step)

	s.auditLog("totp_enable", handlers.UserFrom(r), "user="+username)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "2FA enabled for " + username})
}

// handleTOTPDisable removes a user's second factor.
func (s *Server) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")

	s.cfgMu.Lock()
	found := -1
	for i, u := range s.cfg.Users {
		if u.Username == username {
			found = i
			break
		}
	}
	if found < 0 {
		s.cfgMu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found: " + username})
		return
	}
	s.cfg.Users[found].TOTPSecret = ""
	err := writeConfig(s.configPath, s.cfg)
	s.cfgMu.Unlock()
	if err != nil {
		log.Printf("write config: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist config"})
		return
	}

	s.enrollMu.Lock()
	delete(s.enroll, username)
	s.enrollMu.Unlock()
	s.totpReplay.Forget(username)

	s.auditLog("totp_disable", handlers.UserFrom(r), "user="+username)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "2FA disabled for " + username})
}

func (s *Server) userExists(username string) bool {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	for _, u := range s.cfg.Users {
		if u.Username == username {
			return true
		}
	}
	return false
}

// handleAudit returns the most recent audit entries, most recent first.
func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	n := 100
	if v := r.URL.Query().Get("n"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 1000 {
			n = parsed
		}
	}
	entries, err := s.audit.Read(n)
	if err != nil {
		log.Printf("read audit: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read audit log"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries})
}

// handleMetrics merges the host-computed usage stats with server process stats
// (uptime + resident RAM) that only the server process knows.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	args := map[string]string{"audit_path": s.auditPath}
	if model := r.URL.Query().Get("model"); model != "" {
		args["model"] = model
	}
	resp, err := s.exec.Exec("metrics", args)
	if err != nil {
		log.Printf("metrics backend: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "backend unavailable"})
		return
	}
	var m map[string]any
	if err := json.Unmarshal(resp.Data, &m); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "parse metrics"})
		return
	}
	m["uptime_s"] = int(time.Since(s.uptime).Seconds())
	m["ram_mb"] = processRAMMB()
	writeJSON(w, http.StatusOK, m)
}

// processRAMMB returns the server's resident set size in MB from /proc/self/status.
func processRAMMB() int {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.Atoi(fields[1]); err == nil {
					return kb / 1024
				}
			}
		}
	}
	return 0
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	resp, err := s.exec.Exec("settings-content", nil)
	if err != nil {
		log.Printf("settings-content backend: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "backend unavailable"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp.Data)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	log.Printf("CCSM listening on %s", s.http.Addr)
	s.startTurnWatcher()
	return s.http.ListenAndServe()
}

// noCache makes the browser revalidate static assets on every load (via
// Last-Modified/304) instead of keeping a stale copy after a deploy.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// maxJSONBody caps request bodies decoded as JSON, so a client can't force
// unbounded memory allocation in json.Decode by sending an oversized body.
const maxJSONBody = 1 << 20 // 1 MiB

// decodeJSON reads and decodes a JSON body under maxJSONBody, writing a 400
// and returning false on any read/size/parse error.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return false
	}
	return true
}

const minimalHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Claude Sessions</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 640px; margin: 4rem auto; padding: 0 1rem; background: #1a1a2e; color: #e0e0e0; }
  h1 { font-size: 1.5rem; }
  code { background: #16213e; padding: 0.2em 0.4em; border-radius: 4px; }
</style>
</head>
<body>
<h1>🤖 Claude Code Session Manager</h1>
<p>CCSM v` + Version + ` — API server running.</p>
<p>Serves the UI from <code>/static/</code> when built with it embedded; without it,
this page is the response for <code>/</code>.</p>
<p>Endpoints: <code>GET /api/health</code> · <code>POST /api/auth/login</code> · <code>GET /api/auth/status</code></p>
</body>
</html>`
