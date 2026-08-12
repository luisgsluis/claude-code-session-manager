// Package host implements the execution of tmux/claude commands. It is the
// single source of truth for command logic, used both by ccsm-agent (over a
// Unix socket) and by ccsm running in direct mode (package, no agent).
package host

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Options configures a Host executor.
type Options struct {
	ProfilesPath    string
	SettingsPath    string
	ConvPath        string
	ClaudeBinary    string
	TmuxBinary      string
	BashBinary      string
	RcBootstrap     string
	RcWaitSeconds   int
	RcPollSeconds   int
	RcSettleSeconds int
	Home            string
}

// Host executes tmux/claude commands. It runs either in the ccsm-agent process
// (container deployment) or directly inside ccsm (package deployment).
type Host struct {
	profilesPath    string
	settingsPath    string
	convPath        string
	claudeBinary    string
	tmuxBinary      string
	bashBinary      string
	rcBootstrap     string
	rcWaitSeconds   int
	rcPollSeconds   int
	rcSettleSeconds int
	home            string

	// modeWheelCache holds the discovered Shift+Tab wheel order per profile
	// (the wheel is account-dependent: auto/bypassPermissions only appear when
	// enabled, so a hardcoded order can silently land on the wrong mode).
	modeWheelCache map[string][]string
	modeMu         sync.Mutex
}

// Error carries an HTTP-like status so the web server can return the right
// code (400 invalid input, 404 missing, 500 execution failure).
type Error struct {
	Status int
	Msg    string
}

func (e *Error) Error() string { return e.Msg }

// errBad builds a 400 error.
func errBad(format string, args ...any) error {
	return &Error{Status: 400, Msg: fmt.Sprintf(format, args...)}
}

// errNotFound builds a 404 error.
func errNotFound(format string, args ...any) error {
	return &Error{Status: 404, Msg: fmt.Sprintf(format, args...)}
}

// errServer builds a 500 error.
func errServer(format string, args ...any) error {
	return &Error{Status: 500, Msg: fmt.Sprintf(format, args...)}
}

// errConflict builds a 409 error.
func errConflict(format string, args ...any) error {
	return &Error{Status: 409, Msg: fmt.Sprintf(format, args...)}
}

// New resolves "auto"/empty paths against $HOME and returns a Host.
func New(o Options) *Host {
	if o.Home == "" {
		o.Home = os.Getenv("HOME")
	}
	def := func(v, fallback string) string {
		if v == "" || v == "auto" {
			return fallback
		}
		return v
	}
	return &Host{
		profilesPath:    def(o.ProfilesPath, o.Home+"/claude-shared/claude-perfiles"),
		settingsPath:    def(o.SettingsPath, o.Home+"/.claude/settings.json"),
		convPath:        def(o.ConvPath, o.Home+"/.claude/projects/-home-admin"),
		claudeBinary:    def(o.ClaudeBinary, o.Home+"/.local/bin/claude"),
		tmuxBinary:      def(o.TmuxBinary, "/usr/bin/tmux"),
		bashBinary:      def(o.BashBinary, "/usr/bin/bash"),
		rcBootstrap:     def(o.RcBootstrap, "estandar"),
		rcWaitSeconds:   o.RcWaitSeconds,
		rcPollSeconds:   o.RcPollSeconds,
		rcSettleSeconds: o.RcSettleSeconds,
		home:            o.Home,
		modeWheelCache:  map[string][]string{},
	}
}

// Exec validates and dispatches a command. It returns the command's data and,
// on failure, an error that may be a *Error carrying an HTTP status.
func (h *Host) Exec(cmd string, args map[string]string) (any, error) {
	switch cmd {
	case "tmux-ls":
		return h.tmuxList()
	case "tmux-kill":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		return nil, h.tmuxKill(name)
	case "tmux-rename":
		name := args["name"]
		newName := args["new_name"]
		if !safeName(name) || !safeName(newName) {
			return nil, errBad("invalid session name")
		}
		return h.tmuxRename(name, newName)
	case "session-pane":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		return h.sessionPane(name)
	case "session-conv":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		return h.sessionConv(name)
	case "session-chat":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		return h.sessionChat(name)
	case "session-status":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		return h.sessionStatus(name)
	case "session-send":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		return h.sessionSend(name, args["text"], args["keys"])
	case "session-mode":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		if args["mode"] == "" {
			return nil, errBad("missing mode")
		}
		return h.sessionMode(name, args["mode"])
	case "session-rc":
		name := args["name"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		return h.sessionRc(name)
	case "claude-rename":
		name := args["session"]
		title := args["title"]
		if !safeName(name) {
			return nil, errBad("invalid session name")
		}
		if !safeTitle(title) {
			return nil, errBad("invalid title")
		}
		return h.claudeRename(name, title)
	case "claude-nueva":
		if p := args["profile"]; p != "" && !safeProfileName(p) {
			return nil, errBad("invalid profile name")
		}
		if n := strings.TrimSpace(args["name"]); n != "" && !safeName(n) {
			return nil, errBad("invalid session name")
		}
		return h.claudeNew(args)
	case "claude-resume":
		id := args["id"]
		if !safeUUID(id) {
			return nil, errBad("invalid conversation id")
		}
		return h.claudeResume(id)
	case "claude-perfil":
		name := args["profile"]
		if !safeProfileName(name) {
			return nil, errBad("invalid profile name")
		}
		return nil, h.claudeProfile(name)
	case "profiles-ls":
		return h.profilesList()
	case "projects-ls":
		return h.projectsList()
	case "profile-content":
		name := args["name"]
		if !safeProfileName(name) {
			return nil, errBad("invalid profile name")
		}
		return h.profileContent(name)
	case "settings-content":
		return h.settingsContent()
	case "conversations-ls":
		return h.conversationsList(args)
	case "conversation-get":
		id := args["id"]
		if !safeUUID(id) {
			return nil, errBad("invalid conversation id")
		}
		return h.conversationGet(id, args["lines"])
	case "conversation-export":
		id := args["id"]
		if !safeUUID(id) {
			return nil, errBad("invalid conversation id")
		}
		return h.conversationExport(id, args["format"])
	case "conversation-meta-get":
		id := args["id"]
		if !safeUUID(id) {
			return nil, errBad("invalid conversation id")
		}
		return h.conversationMetaGet(id)
	case "conversation-meta-set":
		id := args["id"]
		if !safeUUID(id) {
			return nil, errBad("invalid conversation id")
		}
		return h.conversationMetaSet(id, args)
	case "metrics":
		return h.metrics(args["audit_path"], args["model"])
	default:
		return nil, errBad("unknown command: %s", cmd)
	}
}

// Validation patterns (same as olivetin-cmd). Enforced here for every entry
// point: ccsm-agent over the socket and ccsm in direct mode.
var (
	namePattern    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	uuidRe         = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	uuidPattern    = regexp.MustCompile(`^` + uuidRe.String() + `$`)
	profilePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	titlePattern   = regexp.MustCompile(`^[\p{L}\p{N}\p{P} ]{1,80}$`)
	// projectPattern matches the relative name of a project dir under home
	// (as returned by projectsList). No path traversal: dots are allowed only
	// as part of names, never as "..".
	projectPattern = regexp.MustCompile(`^[\p{L}\p{N}._-]+(/[\p{L}\p{N}._-]+)*$`)
)

func safeProjectName(s string) bool {
	return projectPattern.MatchString(s) && !strings.Contains(s, "..")
}

func safeName(s string) bool        { return namePattern.MatchString(s) }
func safeUUID(s string) bool        { return uuidPattern.MatchString(s) }
func safeProfileName(s string) bool { return profilePattern.MatchString(s) }
func safeTitle(s string) bool       { return titlePattern.MatchString(s) }

var leadingNoise = regexp.MustCompile(`^[^\p{L}\p{N}]*[\s]*`)

func (h *Host) tmuxList() ([]map[string]any, error) {
	out, err := exec.Command(h.tmuxBinary, "list-sessions", "-F", "#{session_name}\t#{session_created_string}\t#{pane_title}\t#{@ccsm_project}").Output()
	if err != nil {
		// No sessions is not an error — tmux exits 1
		return []map[string]any{}, nil
	}

	hostname := os.Getenv("HOSTNAME")
	var sessions []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		s := map[string]any{"name": parts[0]}
		if len(parts) > 1 {
			s["created"] = parts[1]
		}
		if len(parts) > 2 {
			task := leadingNoise.ReplaceAllString(strings.TrimSpace(parts[2]), "")
			if task == "" || task == hostname {
				task = "(sin tarea)"
			}
			s["task"] = task
		}
		if len(parts) > 3 {
			s["project"] = strings.TrimSpace(parts[3])
		}
		s["status"] = h.rcStatus(parts[0])
		sessions = append(sessions, s)
	}

	// tmux numbers sessions; sort numerically so "10" comes after "9".
	sort.Slice(sessions, func(i, j int) bool {
		ni, e1 := strconv.Atoi(sessions[i]["name"].(string))
		nj, e2 := strconv.Atoi(sessions[j]["name"].(string))
		if e1 == nil && e2 == nil {
			return ni < nj
		}
		return sessions[i]["name"].(string) < sessions[j]["name"].(string)
	})

	return sessions, nil
}

// rcStatus reports the Remote Control bridge state of a tmux session: the pane
// status bar and scrollback are the primary signal (the /rc badge), and a live
// claude carrying --remote-control in its argv also counts as connected. The
// badge is hidden whenever the status bar shows the mode hint ("plan mode on",
// "auto mode on", ...), so relying on the badge alone yields false "pending"
// for sessions that are actually registered in the app.
func (h *Host) rcStatus(session string) string {
	if st := h.rcStatusLive(session); st != "rc_pending" {
		return st
	}
	if h.paneRCActive(session) {
		return "rc_connected"
	}
	return "rc_pending"
}

// rcStatusLive reports the bridge state from the pane alone (badge +
// scrollback), without the process-argv fallback. Wait loops use it: an alive
// --remote-control process is RC *intent*, not a confirmed bridge.
//
// Since 2.1.228 the status bar no longer paints the /rc badge or the
// "/remote-control is active" footer flag, so the pane is no longer a reliable
// indicator on its own. The authoritative signal is the per-process session
// file Claude writes at ~/.claude/sessions/<pid>.json: bridgeSessionId is
// present exactly when the mobile-app bridge registered. Check it first; keep
// the badge/scrollback checks as a fallback for older versions.
func (h *Host) rcStatusLive(session string) string {
	if h.sessionBridgeID(session) != "" {
		return "rc_connected"
	}
	out, err := exec.Command(h.tmuxBinary, "capture-pane", "-p", "-t", session).Output()
	if err != nil {
		return "rc_pending"
	}
	lastLine := ""
	for _, l := range strings.Split(string(out), "\n") {
		if t := strings.TrimSpace(l); t != "" {
			lastLine = t
		}
	}
	if strings.Contains(lastLine, "/rc failed") {
		return "rc_failed"
	}
	if strings.Contains(lastLine, "/rc") {
		return "rc_connected"
	}
	if hist, err := exec.Command(h.tmuxBinary, "capture-pane", "-p", "-t", session, "-S", "-").Output(); err == nil {
		for _, l := range strings.Split(string(hist), "\n") {
			if strings.Contains(l, "/remote-control is active") {
				return "rc_connected"
			}
		}
	}
	return "rc_pending"
}

func (h *Host) tmuxKill(name string) error {
	out, err := exec.Command(h.tmuxBinary, "kill-session", "-t", "="+name).CombinedOutput()
	if err != nil {
		return errServer("tmux kill-session: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (h *Host) claudeNew(args map[string]string) (map[string]string, error) {
	profile := args["profile"]
	name := strings.TrimSpace(args["name"])
	if name != "" && h.sessionAlive(name) {
		return nil, errBad("session name already in use: %s", name)
	}
	// The Claude title can differ from the tmux session name; it defaults to
	// the tmux name when not given (renaming both is the historical behaviour).
	claudeName := strings.TrimSpace(args["claude_name"])
	if claudeName == "" {
		claudeName = name
	}
	if claudeName != "" && !safeTitle(claudeName) {
		return nil, errBad("invalid claude name")
	}
	// The project (a name from projectsList) pins the working dir Claude starts
	// in, so the session boots with that project's CLAUDE.md. "principal" (or
	// empty) keeps the historical launch from home and is never tagged.
	project := args["project"]
	cwd, ok := h.projectCwd(project)
	if !ok {
		return nil, errBad("unknown project: %s", project)
	}

	var session, status string
	var waitRC bool
	var err error

	// A deterministic conversation id lets the web UI map this tmux session to
	// its transcript jsonl right away (the id travels in the process argv).
	sessionID := newUUID()

	if profile == "" {
		if activo := h.activeProfileName(); activo != "" && perfilSinRC(h.profilesPath+"/"+activo+".json") {
			session, status, err = h.lanzarConStaging(activo, "--session-id "+sessionID, name, cwd)
		} else {
			waitRC = true
			session, err = h.newSession("--remote-control --session-id "+sessionID, name, cwd)
			if err == nil {
				status = h.rcStatus(session)
			}
		}
	} else {
		profileFile := h.profilesPath + "/" + profile + ".json"
		if _, statErr := os.Stat(profileFile); statErr != nil {
			return nil, errNotFound("profile not found: %s", profile)
		}
		if perfilSinRC(profileFile) {
			session, status, err = h.lanzarConStaging(profile, "--session-id "+sessionID, name, cwd)
		} else {
			waitRC = true
			session, err = h.newSession("--settings "+profileFile+" --remote-control --session-id "+sessionID, name, cwd)
			if err == nil {
				status = h.rcStatus(session)
			}
		}
	}

	if err != nil {
		return nil, errServer("%v", err)
	}
	// Tag the session with its project so the sessions list can show it. The
	// "principal" entry is the implicit default and carries no tag. The bare
	// name is required: set-option's target doesn't accept the "=" prefix that
	// kill-session does (it would look for a session literally named "=name").
	if project != "" && project != "principal" {
		exec.Command(h.tmuxBinary, "set-option", "-t", session, "@ccsm_project", project).Run()
	}
	if claudeName != "" {
		h.renameClaudeAfterReady(session, claudeName, waitRC)
	}
	return map[string]string{
		"session": session,
		"status":  rcState(status),
	}, nil
}

func (h *Host) claudeResume(id string) (map[string]string, error) {
	if _, err := os.Stat(h.convFileFor(id)); err != nil {
		return nil, errNotFound("conversation not found: %s", id)
	}
	// Gatillo del 4090 "no longer the active worker": retomar una conversación
	// que otra sesión tiene abierta con el bridge del móvil registrado reclama el
	// rol de worker activo y desconecta esa sesión. Advertir antes de tumbarla en
	// silencio; para relanzar una sesión caída existe el botón "RC: re-registrar"
	// (sessionRc), que hace el kill+resume controlado.
	if other := h.activeSessionUsingConv(id); other != "" {
		return nil, errConflict("la conversación ya está conectada al móvil en la sesión %s; retomarla la desconectaría (code 4090). Ciérrala o usa el botón 'RC: re-registrar' de esa sesión para relanzarla", other)
	}

	var session, status string
	var err error
	activo := h.activeProfileName()
	if activo != "" && perfilSinRC(h.profilesPath+"/"+activo+".json") {
		session, status, err = h.lanzarConStaging(activo, "--resume "+id, "", h.home)
	} else {
		session, err = h.newSession("--resume "+id+" --remote-control", "", h.home)
		if err == nil {
			status = h.rcStatus(session)
		}
	}
	if err != nil {
		return nil, errServer("%v", err)
	}
	return map[string]string{
		"session": session,
		"status":  rcState(status),
	}, nil
}

// rcState maps a session status to the API vocabulary ("rc_connected",
// "rc_failed", "rc_pending"). It accepts either the staging outcome from
// lanzarConStaging ("ok"/"fail"/"timeout"/"dead") or an rcStatus value already
// in the API vocabulary — RC-clean sessions get a live rcStatus() result and
// must pass through unchanged, not collapse to rc_pending.
func rcState(staging string) string {
	switch staging {
	case "ok":
		return "rc_connected"
	case "fail":
		return "rc_failed"
	case "rc_connected", "rc_failed":
		return staging
	default:
		return "rc_pending"
	}
}

// lanzarConStaging is the two-phase start for profiles that disable Remote
// Control (ported from olivetin-cmd): apply the clean bootstrap profile so RC
// enables at startup, create the session with --remote-control, wait for the
// bridge to connect (2 consecutive ok polls), then hot-apply the target
// profile — RC survives the switch. Returns (session, "ok"|"fail"|"timeout"|"dead").
func (h *Host) lanzarConStaging(destino, extra, name, cwd string) (string, string, error) {
	if err := h.applyProfile(h.rcBootstrap); err != nil {
		return "", "", fmt.Errorf("bootstrap profile: %w", err)
	}
	session, err := h.newSession("--remote-control "+extra, name, cwd)
	if err != nil {
		h.applyProfile(destino) // restore target; best effort
		return "", "", err
	}
	estado := h.waitRCBridgeSettled(session)
	// Always restore the target profile (even on failure) so the global
	// settings end in the requested state.
	if err := h.applyProfile(destino); err != nil {
		return session, estado, err
	}
	return session, estado, nil
}

// waitRCBridge polls rcStatusLive until the bridge registers (2 consecutive ok
// polls), the wait window elapses (timeout), or the session dies (dead).
// Returns "ok" | "fail" | "timeout" | "dead".
func (h *Host) waitRCBridge(session string) string {
	estado := "pending"
	confirmed := 0
	deadline := time.Now().Add(time.Duration(h.rcWaitSeconds) * time.Second)
	for time.Now().Before(deadline) {
		if !h.sessionAlive(session) {
			estado = "dead"
			break
		}
		switch h.rcStatusLive(session) {
		case "rc_connected":
			confirmed++
			if confirmed >= 2 {
				estado = "ok"
			}
		case "rc_failed":
			estado = "fail"
		}
		if estado == "ok" || estado == "fail" {
			break
		}
		time.Sleep(time.Duration(h.rcPollSeconds) * time.Second)
	}
	if estado == "pending" {
		estado = "timeout"
	}
	return estado
}

// sessionIdle reports whether the session's process has finished loading (its
// session file says status:"idle"). A resumed session is only ready once it has
// loaded its transcript; the mobile bridge registers only after that. Without a
// session file there is nothing to read, so it reports idle (don't block).
func (h *Host) sessionIdle(name string) bool {
	pid := h.panePID(name)
	if pid <= 0 {
		return true
	}
	f := filepath.Join(h.home, ".claude", "sessions", fmt.Sprintf("%d.json", pid))
	data, err := os.ReadFile(f)
	if err != nil {
		return true
	}
	var sf struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &sf); err != nil {
		return true
	}
	return sf.Status == "idle"
}

// waitRCBridgeSettled waits for the bridge to register AND the session process
// to be idle (a resume finished loading its transcript), holding that state for
// a settle margin. Restoring the target profile right after the bridge appears
// drops it on resumed sessions (visto 2026-08-12): the bridge needs the process
// idle before a switch to a perfilSinRC profile survives. The settle margin is
// a short confirmation (rcSettleSeconds, 1s), not a fixed wait — the time to
// reach idle is covered by rcWaitSeconds. Same vocabulary as waitRCBridge.
func (h *Host) waitRCBridgeSettled(session string) string {
	estado := "timeout"
	deadline := time.Now().Add(time.Duration(h.rcWaitSeconds) * time.Second)
	for time.Now().Before(deadline) {
		if !h.sessionAlive(session) {
			return "dead"
		}
		switch h.rcStatusLive(session) {
		case "rc_connected":
			if h.sessionIdle(session) {
				estado = "ok"
			}
		case "rc_failed":
			return "fail"
		}
		if estado == "ok" {
			break
		}
		time.Sleep(time.Duration(h.rcPollSeconds) * time.Second)
	}
	if estado != "ok" {
		return "timeout"
	}
	deadline = time.Now().Add(time.Duration(h.rcSettleSeconds) * time.Second)
	for time.Now().Before(deadline) {
		if !h.sessionAlive(session) {
			return "dead"
		}
		if !h.sessionIdle(session) || h.rcStatusLive(session) != "rc_connected" {
			return "fail"
		}
		time.Sleep(time.Duration(h.rcPollSeconds) * time.Second)
	}
	return "ok"
}

func (h *Host) claudeProfile(name string) error {
	if err := h.applyProfile(name); err != nil {
		return err
	}
	return nil
}

// applyProfile copies the named catalog profile over Claude Code's
// settings.json. The file is validated as JSON before writing so a broken
// profile can never leave settings.json unparseable (Claude Code then fails to
// start). os.WriteFile follows a symlinked settings.json (writes through to the
// shared target, never replacing the link) so Syncthing sharing survives.
func (h *Host) applyProfile(name string) error {
	data, err := os.ReadFile(h.profilesPath + "/" + name + ".json")
	if err != nil {
		return errNotFound("profile not found: %s", name)
	}
	if !json.Valid(data) {
		return errServer("profile '%s' is not valid JSON", name)
	}
	if err := os.WriteFile(h.settingsPath, data, 0600); err != nil {
		return errServer("write settings: %v", err)
	}
	return nil
}

// perfilSinRC reports whether a profile disables Remote Control at startup.
// apiKeyHelper, an API key or auth token in env, or a non-Anthropic base URL
// all make Claude Code skip RC (matches olivetin-cmd's perfil_sin_rc).
func perfilSinRC(profilePath string) bool {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return false
	}
	var doc struct {
		APIKeyHelper json.RawMessage `json:"apiKeyHelper"`
		Env          struct {
			ANTHROPIC_API_KEY    json.RawMessage `json:"ANTHROPIC_API_KEY"`
			ANTHROPIC_AUTH_TOKEN json.RawMessage `json:"ANTHROPIC_AUTH_TOKEN"`
			ANTHROPIC_BASE_URL   string          `json:"ANTHROPIC_BASE_URL"`
		} `json:"env"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	has := func(r json.RawMessage) bool { return len(r) > 0 && string(r) != "null" }
	if has(doc.APIKeyHelper) || has(doc.Env.ANTHROPIC_API_KEY) || has(doc.Env.ANTHROPIC_AUTH_TOKEN) {
		return true
	}
	if doc.Env.ANTHROPIC_BASE_URL == "" {
		return false
	}
	anthropicHost := regexp.MustCompile(`^(https?://)?(api\.)?anthropic\.com($|/)`)
	return !anthropicHost.MatchString(doc.Env.ANTHROPIC_BASE_URL)
}

// activeProfileName returns the catalog profile whose content matches the
// currently applied settings.json ("" if none matches).
func (h *Host) activeProfileName() string {
	settingsData, err := os.ReadFile(h.settingsPath)
	if err != nil {
		return ""
	}
	active := normalizeJSON(settingsData)
	entries, err := os.ReadDir(h.profilesPath)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if !safeProfileName(name) {
			continue
		}
		if pf, err := os.ReadFile(h.profilesPath + "/" + e.Name()); err == nil && normalizeJSON(pf) == active {
			return name
		}
	}
	return ""
}

// sessionAlive reports whether a tmux session with the given name still exists.
func (h *Host) sessionAlive(name string) bool {
	return exec.Command(h.tmuxBinary, "has-session", "-t", "="+name).Run() == nil
}

// normalizeJSON compacts arbitrary JSON so two semantically equal documents
// compare equal regardless of key order or formatting (matches olivetin-cmd's
// `jq -S -c .` when detecting the active profile).
func normalizeJSON(data []byte) string {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func (h *Host) profileContent(name string) (map[string]string, error) {
	path := h.profilesPath + "/" + name + ".json"
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotFound("profile not found: %s", name)
		}
		return nil, errServer("read profile: %v", err)
	}
	return map[string]string{"name": name, "content": string(data)}, nil
}

func (h *Host) settingsContent() (map[string]string, error) {
	data, err := os.ReadFile(h.settingsPath)
	if err != nil {
		return nil, errServer("read settings: %v", err)
	}
	return map[string]string{"content": string(data)}, nil
}

func (h *Host) profilesList() ([]map[string]any, error) {
	entries, err := os.ReadDir(h.profilesPath)
	if err != nil {
		return nil, errServer("read profiles dir: %v", err)
	}

	settingsData, _ := os.ReadFile(h.settingsPath)
	activeSettings := normalizeJSON(settingsData)

	var profiles []map[string]any
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if !safeProfileName(name) {
			continue
		}
		isActive := false
		if pf, err := os.ReadFile(h.profilesPath + "/" + e.Name()); err == nil && activeSettings != "" {
			isActive = normalizeJSON(pf) == activeSettings
		}
		profiles = append(profiles, map[string]any{
			"name":      name,
			"label":     name,
			"is_active": isActive,
		})
	}

	if profiles == nil {
		profiles = []map[string]any{}
	}

	return profiles, nil
}

// projectsList discovers launchable "projects": the home itself (entry
// "principal") plus every directory under home, up to 3 levels deep (skipping
// dot-dirs), that contains a CLAUDE.md. The name is the path relative to home,
// so entries are unique and round-trip through projectCwd without ambiguity.
// Descent is never cut short at a CLAUDE.md dir: a container dir like
// ~/projects may have its own CLAUDE.md while its children are the projects.
func (h *Host) projectsList() ([]map[string]any, error) {
	out := []map[string]any{{"name": "principal", "path": h.home}}

	hasDoc := func(p string) bool {
		for _, name := range []string{"CLAUDE.md", "claude.md"} {
			if _, err := os.Stat(filepath.Join(p, name)); err == nil {
				return true
			}
		}
		return false
	}

	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 3 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if p == h.home {
				continue
			}
			if hasDoc(p) {
				if rel, err := filepath.Rel(h.home, p); err == nil {
					out = append(out, map[string]any{"name": rel, "path": p})
				}
			}
			walk(p, depth+1)
		}
	}
	walk(h.home, 1)

	sort.Slice(out[1:], func(i, j int) bool { return out[i+1]["name"].(string) < out[j+1]["name"].(string) })
	return out, nil
}

// projectCwd resolves a project name to the working dir to launch Claude in,
// or "" when the name is not a known project. "principal" (or empty) maps to
// the home itself, keeping the historical launch behaviour.
func (h *Host) projectCwd(project string) (string, bool) {
	if project == "" || project == "principal" {
		return h.home, true
	}
	if !safeProjectName(project) {
		return "", false
	}
	want := filepath.Join(h.home, filepath.FromSlash(project))
	projects, err := h.projectsList()
	if err != nil {
		return "", false
	}
	for _, pr := range projects {
		if pr["name"] == project && pr["path"] == want {
			return want, true
		}
	}
	return "", false
}

// convLine is one JSON object of a Claude Code conversation .jsonl file.
type convLine struct {
	Type      string `json:"type"`
	IsMeta    *bool  `json:"isMeta"`
	Cwd       string `json:"cwd"`
	Timestamp string `json:"timestamp"`
	// queue-operation lines: a message typed while the session was mid-turn is
	// recorded here (operation enqueue/remove) with the text in the top-level
	// content field, not in message.content.
	Operation string `json:"operation"`
	Content   string `json:"content"`
	Message   struct {
		ID      string          `json:"id"`
		Model   string          `json:"model"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// extractText pulls the human-readable text out of a Claude message content,
// which can be a plain string or an array of content blocks.
func extractText(content json.RawMessage) (string, bool) {
	if len(content) == 0 {
		return "", false
	}
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s, true
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &blocks); err != nil {
		return "", false
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, " "), true
}

// conversationSummary finds the first real user message (no meta, no tool
// results, no slash commands) in the first 200 lines of a conversation file.
func conversationSummary(path string) (text, cwd string, ok bool) {
	fh, err := os.Open(path)
	if err != nil {
		return "", "", false
	}
	defer fh.Close()
	scanner := bufio.NewScanner(fh)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for scanner.Scan() {
		n++
		if n > 200 {
			break
		}
		var line convLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "user" || (line.IsMeta != nil && *line.IsMeta) {
			continue
		}
		t, okT := extractText(line.Message.Content)
		t = strings.TrimSpace(t)
		if !okT || t == "" || strings.HasPrefix(t, "<") {
			continue
		}
		return t, line.Cwd, true
	}
	return "", "", false
}

// conversationTitle returns the latest name of a conversation: the manually set
// /rename title wins, otherwise the AI-generated one. Claude Code writes
// ai-title (and, after a rename, custom-title) lines on every turn, so the
// current name is near the tail; we scan the last 64KB and fall back to the
// beginning of the file if the tail has none.
func conversationTitle(path string) string {
	fh, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer fh.Close()
	const tailSize = 64 * 1024
	if st, err := fh.Stat(); err == nil && st.Size() > tailSize {
		if _, err := fh.Seek(st.Size()-tailSize, 0); err != nil {
			return ""
		}
	}
	title := ""
	scanner := bufio.NewScanner(fh)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var line struct {
			Type        string `json:"type"`
			AITitle     string `json:"aiTitle"`
			CustomTitle string `json:"customTitle"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		switch line.Type {
		case "custom-title":
			if t := strings.TrimSpace(line.CustomTitle); t != "" {
				title = t
			}
		case "ai-title":
			if title == "" {
				if t := strings.TrimSpace(line.AITitle); t != "" {
					title = t
				}
			}
		}
	}
	if title != "" {
		return truncateRunes(title, 80)
	}
	// Tail had no title (very old files): scan the beginning once.
	if _, err := fh.Seek(0, 0); err != nil {
		return ""
	}
	scanner = bufio.NewScanner(fh)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for scanner.Scan() {
		n++
		if n > 200 {
			break
		}
		var line struct {
			Type        string `json:"type"`
			AITitle     string `json:"aiTitle"`
			CustomTitle string `json:"customTitle"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		switch line.Type {
		case "custom-title":
			if t := strings.TrimSpace(line.CustomTitle); t != "" {
				return truncateRunes(t, 80)
			}
		case "ai-title":
			if t := strings.TrimSpace(line.AITitle); t != "" {
				return truncateRunes(t, 80)
			}
		}
	}
	return ""
}

// cleanText flattens a prompt to a single line, safe for JSON and for display.
func cleanText(s string) string {
	s = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

var (
	ansiCSI = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)      // CSI: \x1b[1m, \x1b[22m, \x1b[2J, \x1b[?25l…
	ansiOSC = regexp.MustCompile(`\x1b\][^\a\x1b]*(\a|\x1b\\)`) // OSC: \x1b]0;title\x07 | \x1b]…\x1b\
)

// stripANSI removes ANSI escape sequences (CSI + OSC) from captured pane text.
// Claude Code renders the /model feedback (and its spinner) in bold/dim, and
// without this the Terminal tab would show raw codes like [1m…[22m.
func stripANSI(s string) string {
	s = ansiOSC.ReplaceAllString(s, "")
	return ansiCSI.ReplaceAllString(s, "")
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func originFor(cwd string) string {
	switch {
	case strings.HasPrefix(cwd, "/home/admin"):
		return "pi"
	case strings.HasPrefix(cwd, "/home/luis"):
		return "pc"
	}
	return "?"
}

type convFileEntry struct {
	path string
	id   string
	mod  time.Time
}

// convProjectsDir is the parent of the conversation dir. Transcripts live in
// projects/<cwd-slug>/<uuid>.jsonl and the slug depends on the session's cwd,
// so lookups must search every project subfolder, not just h.convPath.
func (h *Host) convProjectsDir() string {
	return filepath.Dir(h.convPath)
}

// convFileFor resolves the path of a conversation file searching every project
// subfolder, falling back to h.convPath (legacy flat layout).
func (h *Host) convFileFor(id string) string {
	if matches, _ := filepath.Glob(filepath.Join(h.convProjectsDir(), "*", id+".jsonl")); len(matches) > 0 {
		return matches[0]
	}
	return filepath.Join(h.convPath, id+".jsonl")
}

// convFiles lists every conversation file under projects/ (h.convPath plus its
// immediate subfolders, one level deep), newest first, bare-uuid ids only.
// Error only when h.convPath itself is unreadable/missing; the subfolder scan
// is best-effort. Callers doing best-effort lookups may discard the error.
func (h *Host) convFiles() ([]convFileEntry, error) {
	dirs := []string{h.convPath}
	subs, err := os.ReadDir(h.convProjectsDir())
	if err == nil {
		for _, s := range subs {
			if s.IsDir() {
				d := filepath.Join(h.convProjectsDir(), s.Name())
				if d == h.convPath {
					continue // h.convPath is scanned on its own; do not scan it twice
				}
				dirs = append(dirs, d)
			}
		}
	}
	var files []convFileEntry
	for i, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			if i == 0 {
				return nil, err // unreadable convPath → 500 in the handler
			}
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			id := strings.TrimSuffix(name, ".jsonl")
			// Only bare UUIDs: Syncthing leaves `<uuid>.sync-conflict-*.jsonl`
			// files that --resume cannot find.
			if !safeUUID(id) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, convFileEntry{filepath.Join(d, name), id, info.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	return files, nil
}

// cwdSlug maps a working directory to Claude Code's project transcript folder
// slug: /home/admin → -home-admin, /home/admin/projects/x → -home-admin-projects-x.
func cwdSlug(cwd string) string {
	if cwd == "" || cwd == "/" {
		return ""
	}
	return strings.ReplaceAll(cwd, "/", "-")
}

// sessionCwd returns the working directory of a session's pane ("" when the
// tmux session is gone or display-message fails).
func (h *Host) sessionCwd(name string) string {
	out, err := exec.Command(h.tmuxBinary, "display-message", "-p", "-t", h.paneTarget(name), "#{pane_current_path}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// convFilesIn lists conversation files, preferring the subfolder matching the
// session cwd's slug and falling back to a full scan when that folder does not
// exist. Best-effort disambiguation: a session whose cwd we can see should map
// to its own transcript folder, not a global most-recent pick from every project.
func (h *Host) convFilesIn(cwd string) []convFileEntry {
	if slug := cwdSlug(cwd); slug != "" {
		dir := filepath.Join(h.convProjectsDir(), slug)
		if entries, err := os.ReadDir(dir); err == nil {
			var files []convFileEntry
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				id := strings.TrimSuffix(name, ".jsonl")
				if !safeUUID(id) {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				files = append(files, convFileEntry{filepath.Join(dir, name), id, info.ModTime()})
			}
			sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
			return files
		}
	}
	files, _ := h.convFiles()
	return files
}

// fileBirthTime returns a file's birth (creation) time in epoch seconds via
// coreutils stat (%W); 0 when the filesystem doesn't report it. Used to tell a
// fresh session's own transcript (born when the session starts writing) from an
// older conversation another running session keeps touching.
func fileBirthTime(path string) int64 {
	out, err := exec.Command("stat", "-c", "%W", path).Output()
	if err != nil {
		return 0
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return secs
}

func (h *Host) conversationsList(args map[string]string) ([]map[string]any, error) {
	q := strings.ToLower(strings.TrimSpace(args["q"]))
	origin := strings.TrimSpace(args["origin"])
	aliveOnly := args["alive"] == "1" || args["alive"] == "true"
	archived := args["archived"] // "only" = archivadas; "all" = incluir; "" = ocultar
	from, _ := time.Parse("02/01/2006", strings.TrimSpace(args["from"]))
	to, _ := time.Parse("02/01/2006", strings.TrimSpace(args["to"]))
	if !to.IsZero() {
		// The "to" filter includes the whole day, not just midnight.
		to = to.Add(24*time.Hour - time.Nanosecond)
	}
	page, _ := strconv.Atoi(args["page"])
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(args["per_page"])
	if perPage < 1 {
		perPage = 20
	}

	files, err := h.convFiles()
	if err != nil {
		return nil, errServer("read conversations dir: %v", err)
	}

	alive := h.aliveConversations()

	// Build the full filtered list, then paginate. Pinned conversations float
	// to the top (by date among themselves); the rest stay date-desc.
	var result []map[string]any
	for _, f := range files {
		text, cwd, ok := conversationSummary(f.path)
		if !ok {
			continue
		}
		text = cleanText(text)
		if q != "" && !strings.Contains(strings.ToLower(text), q) {
			continue
		}
		if origin != "" && originFor(cwd) != origin {
			continue
		}
		if aliveOnly && !alive[f.id] {
			continue
		}
		if !from.IsZero() && f.mod.Before(from) {
			continue
		}
		if !to.IsZero() && f.mod.After(to) {
			continue
		}
		meta := h.readConversationMeta(f.id)
		// A conversation being written by a running claude is never hidden:
		// archiving is for finished sessions, and hiding a live one reads as
		// "not alive" (it vanished from the list, even from the alive filter).
		if meta.Archived && !alive[f.id] && archived != "only" && archived != "all" {
			continue
		}
		if !meta.Archived && archived == "only" {
			continue
		}
		result = append(result, map[string]any{
			"id":       f.id,
			"date":     f.mod.Format("02/01 15:04"),
			"origin":   originFor(cwd),
			"title":    conversationTitle(f.path),
			"preview":  truncateRunes(text, 120),
			"is_alive": alive[f.id],
			"tags":     meta.Tags,
			"notes":    truncateRunes(meta.Notes, 60),
			"pinned":   meta.Pinned,
			"archived": meta.Archived,
		})
	}
	// An empty result must serialize as [] (json.Marshal of nil is null, which
	// the UI treats as a broken response).
	if result == nil {
		result = []map[string]any{}
	}
	sort.SliceStable(result, func(i, j int) bool {
		pi, pj := result[i]["pinned"].(bool), result[j]["pinned"].(bool)
		if pi != pj {
			return pi
		}
		return false
	})

	start := (page - 1) * perPage
	if start > len(result) {
		start = len(result)
	}
	end := start + perPage
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}

func (h *Host) conversationGet(id, linesStr string) (map[string]any, error) {
	path := h.convFileFor(id)
	nLines := 50
	if l, err := strconv.Atoi(linesStr); err == nil && l > 0 && l <= 200 {
		nLines = l
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, errNotFound("conversation not found")
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, errServer("open conversation: %v", err)
	}
	defer fh.Close()

	var cwd string
	scanner := bufio.NewScanner(fh)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var dd chatDedup
	for scanner.Scan() {
		var line convLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Cwd != "" {
			cwd = line.Cwd
		}
		role, text, source, ok := chatRoleAndText(line)
		if ok {
			dd.add(role, text, source, line.Message.ID)
		}
	}
	dd.flushPending()
	var msgs []map[string]any
	for i, t := range dd.turns {
		msgs = append(msgs, map[string]any{
			"index":   i,
			"role":    t.role,
			"content": truncateRunes(cleanText(t.text), 400),
		})
	}
	if len(msgs) > nLines {
		msgs = msgs[len(msgs)-nLines:]
	}

	return map[string]any{
		"id":       id,
		"date":     info.ModTime().Format("02/01 15:04"),
		"origin":   originFor(cwd),
		"title":    conversationTitle(path),
		"is_alive": h.aliveConversations()[id],
		"messages": msgs,
	}, nil
}

// conversationExport returns the full conversation for download. Format
// "jsonl" is the raw Claude Code transcript; "txt" is a readable rendering.
func (h *Host) conversationExport(id, format string) (map[string]any, error) {
	path := h.convFileFor(id)
	info, err := os.Stat(path)
	if err != nil {
		return nil, errNotFound("conversation not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errServer("read conversation: %v", err)
	}

	filename := id + ".jsonl"
	var content string
	if format == "txt" {
		filename = id + ".txt"
		content = "Conversación " + id + "\nFecha: " + info.ModTime().Format("02/01/2006 15:04") + "\n\n"
		var dd chatDedup
		for _, line := range strings.Split(string(data), "\n") {
			var l convLine
			if json.Unmarshal([]byte(line), &l) != nil {
				continue
			}
			role, text, source, ok := chatRoleAndText(l)
			if ok {
				dd.add(role, text, source, l.Message.ID)
			}
		}
		dd.flushPending()
		for _, t := range dd.turns {
			content += "[" + t.role + "] " + cleanText(t.text) + "\n\n"
		}
	} else {
		content = string(data)
	}

	return map[string]any{
		"id":       id,
		"filename": filename,
		"content":  content,
	}, nil
}

// conversationMeta is the per-conversation sidecar (tags, notes, pin, archive).
// Stored next to the .jsonl transcript as <uuid>.meta.json.
type conversationMeta struct {
	Tags     []string `json:"tags"`
	Notes    string   `json:"notes"`
	Pinned   bool     `json:"pinned"`
	Archived bool     `json:"archived"`
}

var tagPattern = regexp.MustCompile(`^[\p{L}\p{N}_ -]{1,32}$`)

func (h *Host) conversationMetaPath(id string) string {
	return strings.TrimSuffix(h.convFileFor(id), ".jsonl") + ".meta.json"
}

func (h *Host) readConversationMeta(id string) conversationMeta {
	var meta conversationMeta
	if data, err := os.ReadFile(h.conversationMetaPath(id)); err == nil {
		json.Unmarshal(data, &meta) // best effort; malformed sidecars reset to defaults
	}
	if meta.Tags == nil {
		meta.Tags = []string{}
	}
	return meta
}

func (h *Host) conversationMetaGet(id string) (map[string]any, error) {
	if _, err := os.Stat(h.convFileFor(id)); err != nil {
		return nil, errNotFound("conversation not found")
	}
	meta := h.readConversationMeta(id)
	return map[string]any{
		"id":       id,
		"tags":     meta.Tags,
		"notes":    meta.Notes,
		"pinned":   meta.Pinned,
		"archived": meta.Archived,
	}, nil
}

func (h *Host) conversationMetaSet(id string, args map[string]string) (map[string]any, error) {
	if _, err := os.Stat(h.convFileFor(id)); err != nil {
		return nil, errNotFound("conversation not found")
	}

	meta := h.readConversationMeta(id)
	var err error
	if meta.Tags, err = parseTags(args["tags"]); err != nil {
		return nil, err
	}
	if meta.Notes, err = parseNotes(args["notes"]); err != nil {
		return nil, err
	}
	meta.Pinned = args["pinned"] == "1" || args["pinned"] == "true"
	meta.Archived = args["archived"] == "1" || args["archived"] == "true"

	if err := writeJSONAtomic(h.conversationMetaPath(id), meta); err != nil {
		return nil, errServer("write meta: %v", err)
	}
	return map[string]any{
		"id":       id,
		"tags":     meta.Tags,
		"notes":    meta.Notes,
		"pinned":   meta.Pinned,
		"archived": meta.Archived,
	}, nil
}

// parseTags splits a comma-separated tag list and validates each tag.
func parseTags(s string) ([]string, error) {
	raw := strings.Split(s, ",")
	var tags []string
	seen := map[string]bool{}
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !tagPattern.MatchString(t) {
			return nil, errBad("invalid tag: %s", t)
		}
		if !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}
	if tags == nil {
		tags = []string{}
	}
	return tags, nil
}

func parseNotes(s string) (string, error) {
	notes := strings.TrimSpace(s)
	if len([]rune(notes)) > 500 {
		return "", errBad("notes too long (max 500 chars)")
	}
	return notes, nil
}

func writeJSONAtomic(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// metricsDays is the window for the per-day series returned by the metrics command.
const metricsDays = 14

// metrics aggregates usage stats from the audit log and the conversation files:
// sessions per day and most-used profiles (from session_new entries) and token
// usage per day (from assistant message.usage in the conversation jsonl files).
// The audit file lives next to the server, so its path comes in as an argument.
func (h *Host) metrics(auditPath, modelFilter string) (map[string]any, error) {
	sessPerDay := map[string]int{}
	profiles := map[string]int{}
	if auditPath != "" {
		f, err := os.Open(auditPath)
		if err == nil {
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for sc.Scan() {
				var e struct {
					Time   time.Time `json:"time"`
					Action string    `json:"action"`
					Detail string    `json:"detail"`
				}
				if json.Unmarshal(sc.Bytes(), &e) != nil {
					continue
				}
				if e.Action != "session_new" {
					continue
				}
				sessPerDay[e.Time.Local().Format("02/01/2006")]++
				if prof := extractProfile(e.Detail); prof != "" {
					profiles[prof]++
				}
			}
			f.Close()
		}
	}

	now := time.Now()
	sessionsPerDay := make([]map[string]any, 0, metricsDays)
	tokenPerDay := make([]map[string]any, 0, metricsDays)
	for i := metricsDays - 1; i >= 0; i-- {
		key := now.AddDate(0, 0, -i).Format("02/01/2006")
		sessionsPerDay = append(sessionsPerDay, map[string]any{"date": key, "count": sessPerDay[key]})
		tokenPerDay = append(tokenPerDay, map[string]any{"date": key, "input": 0, "output": 0, "cache": 0})
	}
	byModel := map[string]*modelUsage{}
	h.aggregateTokenUsage(now, tokenPerDay, byModel, modelFilter)

	topProfiles := make([]map[string]any, 0, len(profiles))
	for name, count := range profiles {
		topProfiles = append(topProfiles, map[string]any{"name": name, "count": count})
	}
	sort.Slice(topProfiles, func(i, j int) bool {
		a, b := topProfiles[i]["count"].(int), topProfiles[j]["count"].(int)
		if a != b {
			return a > b
		}
		return topProfiles[i]["name"].(string) < topProfiles[j]["name"].(string)
	})

	tokenByModel := make([]map[string]any, 0, len(byModel))
	for model, u := range byModel {
		tokenByModel = append(tokenByModel, map[string]any{
			"model":    model,
			"input":    u.Input,
			"output":   u.Output,
			"cache":    u.Cache,
			"messages": u.Messages,
		})
	}
	sort.Slice(tokenByModel, func(i, j int) bool {
		a, b := tokenByModel[i]["input"].(int), tokenByModel[j]["input"].(int)
		if a != b {
			return a > b
		}
		return tokenByModel[i]["model"].(string) < tokenByModel[j]["model"].(string)
	})

	return map[string]any{
		"sessions_per_day":     sessionsPerDay,
		"top_profiles":         topProfiles,
		"token_usage_per_day":  tokenPerDay,
		"token_usage_by_model": tokenByModel,
	}, nil
}

func extractProfile(detail string) string {
	for _, part := range strings.Split(detail, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "profile=") {
			return strings.TrimPrefix(part, "profile=")
		}
	}
	return ""
}

func (h *Host) aggregateTokenUsage(now time.Time, perDay []map[string]any, byModel map[string]*modelUsage, modelFilter string) {
	idx := make(map[string]*map[string]any, len(perDay))
	for i := range perDay {
		idx[perDay[i]["date"].(string)] = &perDay[i]
	}
	cutoff := now.AddDate(0, 0, -metricsDays).Add(-time.Hour)
	files, _ := h.convFiles()
	for _, f := range files {
		if f.mod.Before(cutoff) {
			continue
		}
		h.aggregateFileUsage(f.path, idx, byModel, modelFilter)
	}
}

type usageLine struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Message   *struct {
		Model string `json:"model"`
		Usage *struct {
			InputTokens         int `json:"input_tokens"`
			OutputTokens        int `json:"output_tokens"`
			CacheCreationTokens int `json:"cache_creation_input_tokens"`
			CacheReadTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// modelUsage accumulates per-model token totals and message counts.
type modelUsage struct {
	Input    int
	Output   int
	Cache    int
	Messages int
}

func (h *Host) aggregateFileUsage(path string, idx map[string]*map[string]any, byModel map[string]*modelUsage, modelFilter string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var line usageLine
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if line.Type != "assistant" || line.Message == nil || line.Message.Usage == nil {
			continue
		}
		if modelFilter != "" && line.Message.Model != modelFilter {
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, line.Timestamp)
		if err != nil {
			continue
		}
		m, ok := idx[ts.Local().Format("02/01/2006")]
		if !ok {
			continue
		}
		u := line.Message.Usage
		(*m)["input"] = (*m)["input"].(int) + u.InputTokens
		(*m)["output"] = (*m)["output"].(int) + u.OutputTokens
		(*m)["cache"] = (*m)["cache"].(int) + u.CacheCreationTokens + u.CacheReadTokens

		model := line.Message.Model
		if model == "" {
			model = "desconocido"
		}
		b := byModel[model]
		if b == nil {
			b = &modelUsage{}
			byModel[model] = b
		}
		b.Input += u.InputTokens
		b.Output += u.OutputTokens
		b.Cache += u.CacheCreationTokens + u.CacheReadTokens
		b.Messages++
	}
}

func (h *Host) newSession(extraArgs, name, cwd string) (string, error) {
	cmd := h.claudeBinary + " " + extraArgs
	fullCmd := h.bashBinary + " -lc '" + strings.ReplaceAll(cmd, "'", "'\\''") + "'"
	if cwd == "" {
		cwd = h.home
	}
	args := []string{"new-session", "-d", "-P", "-F", "#S", "-c", cwd}
	if name != "" {
		args = append(args, "-s", name)
	}
	args = append(args, fullCmd)
	out, err := exec.Command(h.tmuxBinary, args...).Output()
	if err != nil {
		return "", fmt.Errorf("tmux new-session: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// sessionPane returns the visible content of a tmux session's pane plus its
// scrollback, joined so wrapped lines keep their shape. Feeds the SSE live
// view. -S - grabs the full history; apps that draw the alternate screen
// (Claude Code, less) keep no scrollback, so for them this is just the
// current screen.
func (h *Host) sessionPane(name string) (map[string]string, error) {
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}
	out, err := exec.Command(h.tmuxBinary, "capture-pane", "-p", "-J", "-S", "-", "-t", h.paneTarget(name)).Output()
	if err != nil {
		return nil, errServer("tmux capture-pane: %v", err)
	}
	return map[string]string{"session": name, "content": stripANSI(string(out))}, nil
}

// newUUID returns a random RFC 4122 v4 UUID used to pin a session's
// conversation id (claude --session-id).
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst)
}

// sessionConv resolves the Claude conversation id running inside a tmux
// session. Sessions created by CCSM carry --session-id / --resume in the
// process argv, so the id is recoverable from the pane's pid tree; sessions
// started outside CCSM fall back to the most recently written transcript.
func (h *Host) sessionConv(name string) (map[string]any, error) {
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}
	id := h.convIDForSession(name)
	return map[string]any{"session": name, "id": id, "ready": id != ""}, nil
}

// convIDForSession finds the conversation id of a live session.
func (h *Host) convIDForSession(name string) string {
	if pid := h.panePID(name); pid > 0 {
		if id := findUUIDInTree(pid, 3); id != "" {
			return id
		}
	}
	if id := h.lifetimeConv(name); id != "" {
		return id
	}
	return h.newestActiveConv(name)
}

// sessionCreated returns the tmux session's creation time in epoch seconds, or
// 0 when it can't be determined (display-message is unavailable or malformed).
func (h *Host) sessionCreated(name string) int64 {
	out, err := exec.Command(h.tmuxBinary, "display-message", "-p", "-t", h.paneTarget(name), "#{session_created}").Output()
	if err != nil {
		return 0
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil || secs <= 0 {
		return 0
	}
	return secs
}

// lifetimeConv is the fallback mapping for sessions whose claude argv has no
// pinned --session-id (started before Nivel 2 or outside CCSM). It picks the
// most recently written transcript whose last write falls inside the session's
// lifetime (mtime after the tmux session was created). That also matches idle
// sessions: their transcript was last written minutes or hours ago, yet still
// by this running session. A fixed freshness window (newestActiveConv) misses
// those and leaves the chat empty until the next message.
func (h *Host) lifetimeConv(name string) string {
	created := h.sessionCreated(name)
	if created == 0 {
		return h.newestActiveConv(name)
	}
	cutoff := time.Unix(created, 0).Add(-2 * time.Minute)
	bestID, bestTime := "", time.Time{}
	for _, f := range h.convFilesIn(h.sessionCwd(name)) {
		if f.mod.Before(cutoff) {
			continue // written before this session existed: an older run
		}
		// A fresh session's transcript is born when the session starts writing;
		// drop transcripts born before this session (an older conversation that
		// another running session keeps touching, which would hijack the chat).
		if birth := fileBirthTime(f.path); birth != 0 && birth < cutoff.Unix() {
			continue
		}
		if f.mod.After(bestTime) {
			bestID, bestTime = f.id, f.mod
		}
	}
	return bestID
}

// paneInfo resolves a session's pane id (%N) and pid from the FULL pane table,
// matched by session name. tmux's own target resolution of a bare name like
// "0" is ambiguous without a client context: without $TMUX, "-t 0" is parsed
// as window 0 of the most recent session, not the session literally named "0".
// Returns ("", 0) when the session has no pane.
func (h *Host) paneInfo(name string) (string, int) {
	out, err := exec.Command(h.tmuxBinary, "list-panes", "-a", "-F", "#{session_name}\t#{pane_id}\t#{pane_pid}").Output()
	if err != nil {
		return "", 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) != 3 || parts[0] != name {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil || pid <= 1 {
			continue
		}
		return strings.TrimSpace(parts[1]), pid
	}
	return "", 0
}

// paneTarget returns the pane id (%N) to address a session's active pane, or
// the bare session name as a fallback when the pane table can't be resolved.
// Pane-target commands (capture-pane, send-keys) reject the "=" prefix and a
// raw name "0" misresolves, so the pane id is what makes session "0" safe.
func (h *Host) paneTarget(name string) string {
	if id, _ := h.paneInfo(name); id != "" {
		return id
	}
	return name
}

// panePID returns the first pane's pid of a tmux session (0 if unavailable).
func (h *Host) panePID(name string) int {
	_, pid := h.paneInfo(name)
	return pid
}

// sessionBridgeID reports the bridgeSessionId a claude process wrote to its
// per-process session file (~/.claude/sessions/<pid>.json). The pane's pid is
// the claude process itself (newSession runs it directly under bash -lc, which
// execs into claude), so the file name matches the pane pid. The id is present
// exactly when the mobile-app bridge registered — the authoritative RC signal
// since 2.1.228 removed the /rc status-bar badge. "" means the file is
// missing, unreadable, or the bridge never registered.
func (h *Host) sessionBridgeID(name string) string {
	pid := h.panePID(name)
	if pid <= 0 {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(h.home, ".claude", "sessions", fmt.Sprintf("%d.json", pid)))
	if err != nil {
		return ""
	}
	var sf struct {
		BridgeSessionID string `json:"bridgeSessionId"`
	}
	if err := json.Unmarshal(data, &sf); err != nil {
		return ""
	}
	return sf.BridgeSessionID
}

// rcFlagPattern matches --remote-control as a standalone argv token (NUL- or
// space-delimited). That covers both claude's own argv and the bash -lc
// wrapper, which embeds the whole command line in a single quoted token.
var rcFlagPattern = regexp.MustCompile(`(^|[\x00\s])--remote-control([\x00\s]|$)`)

// paneRCActive reports whether a live process in the session's pane was started
// with --remote-control, by walking the pane's process tree argv.
func (h *Host) paneRCActive(name string) bool {
	pid := h.panePID(name)
	if pid <= 0 {
		return false
	}
	return pidTreeHasRc(pid, 3)
}

// pidTreeHasRc searches a pid and its descendants (to a depth) for a
// --remote-control argv token. Uses the kernel's children file, which is
// O(children) instead of a full /proc scan.
func pidTreeHasRc(pid, depth int) bool {
	if depth <= 0 {
		return false
	}
	if rcFlagPattern.Match(procCmdlineBytes(pid)) {
		return true
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", pid, pid))
	if err != nil {
		return false
	}
	for _, f := range strings.Fields(string(b)) {
		if c, err := strconv.Atoi(f); err == nil && pidTreeHasRc(c, depth-1) {
			return true
		}
	}
	return false
}

// procCmdline reads a process's argv as a single string (NUL separators are
// irrelevant for regexp matching).
func procCmdline(pid int) string {
	return string(procCmdlineBytes(pid))
}

func procCmdlineBytes(pid int) []byte {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return nil
	}
	return b
}

// childrenOf returns the direct children of a pid.
func childrenOf(pid int) []int {
	var out []int
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := strconv.Atoi(e.Name())
		if err != nil || p <= 0 {
			continue
		}
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", p))
		if err != nil {
			continue
		}
		// Format: pid (comm) state ppid ...
		if i := bytes.LastIndexByte(stat, ')'); i >= 0 {
			rest := strings.Fields(string(stat[i+1:]))
			if len(rest) >= 2 && rest[1] == strconv.Itoa(pid) {
				out = append(out, p)
			}
		}
	}
	return out
}

// convIDFromCmdline extracts a conversation UUID from raw argv bytes, but only
// when it directly follows a --session-id / --resume flag (NUL-separated, as
// claude's real argv is). A bare uuid anywhere else in argv — e.g. tool-call
// shell text that merely mentions one — is NOT a conversation id, so scanning
// must not treat it as one.
func convIDFromCmdline(b []byte) string {
	for _, marker := range [][]byte{[]byte("--session-id\x00"), []byte("--resume\x00")} {
		if i := bytes.Index(b, marker); i >= 0 {
			if id := uuidRe.Find(b[i+len(marker):]); id != nil {
				return string(id)
			}
		}
	}
	return ""
}

// findUUIDInTree searches a pid and its descendants (to a depth) for a
// conversation UUID in argv (claude --resume <id> / --session-id <id>).
func findUUIDInTree(pid, depth int) string {
	if depth <= 0 {
		return ""
	}
	if id := convIDFromCmdline(procCmdlineBytes(pid)); id != "" {
		return id
	}
	for _, child := range childrenOf(pid) {
		if id := findUUIDInTree(child, depth-1); id != "" {
			return id
		}
	}
	return ""
}

// newestActiveConv returns the most recently modified conversation jsonl
// written while a session runs (fallback for sessions started outside CCSM,
// whose argv has no conversation id). Best effort: with several concurrent
// fresh sessions the match may be ambiguous.
func (h *Host) newestActiveConv(name string) string {
	cutoff := time.Now().Add(-2 * time.Minute)
	bestID, bestTime := "", time.Time{}
	for _, f := range h.convFilesIn(h.sessionCwd(name)) {
		if birth := fileBirthTime(f.path); birth != 0 && birth < cutoff.Unix() {
			continue // born before the freshness window: an older conversation
		}
		if f.mod.After(cutoff) && f.mod.After(bestTime) {
			bestID, bestTime = f.id, f.mod
		}
	}
	return bestID
}

// sessionChat returns the full conversation of a live session plus status
// metadata for the clean chat view: every user/assistant turn (no truncation,
// no terminal chrome) plus alive/rc status, mode and session start.
const maxChatMsgs = 200 // most recent messages kept for the live chat view

// chatRoleAndText classifies a transcript line into a displayable chat message:
// its role ("user"/"assistant"), the text to show and the source ("user" for a
// real user line, "enqueue" for a queue-operation, "assistant"). ok=false means
// the line is not part of the chat (meta events, tool results, control commands,
// empty text).
//
// It also surfaces `queue-operation` lines. Claude Code records a message typed
// while the session was mid-turn as a queue-operation — the text lives in the
// top-level content field and the operation is enqueue (arrived) / remove
// (drained into the running turn) — never as a `user` line. Without this the
// user's mid-turn messages would be invisible to the chat view and the app.
// Only the enqueue is a message; control commands (/remote-control, /plan...)
// are filtered out because the chat view shows real turns, not commands.
//
// Note that once a queued message drains it ALSO becomes a real `user` line
// (promptSource "queued") with the same text, so the enqueue and the user line
// would render the message twice. Callers must fold them through chatDedup.
func chatRoleAndText(line convLine) (role, text, source string, ok bool) {
	switch line.Type {
	case "user":
		if line.IsMeta != nil && *line.IsMeta {
			return "", "", "", false
		}
		t, ok := extractText(line.Message.Content)
		return "user", t, "user", ok
	case "assistant":
		t, ok := extractText(line.Message.Content)
		return "assistant", t, "assistant", ok
	case "queue-operation":
		if line.Operation != "enqueue" {
			return "", "", "", false
		}
		t := strings.TrimSpace(line.Content)
		if t == "" || strings.HasPrefix(t, "/") {
			return "", "", "", false
		}
		return "user", t, "enqueue", true
	default:
		return "", "", "", false
	}
}

// chatTurn is one displayable chat message after transcript deduplication.
type chatTurn struct {
	role   string // "user" | "assistant"
	text   string
	source string // "user" (real user line) | "enqueue" (queue-op) | "assistant"
	msgID  string // assistant message.id (streaming snapshots share it)
}

// chatDedup folds transcript lines into chat turns, dropping the duplicates
// Claude Code itself writes:
//   - a mid-turn message is BOTH a queue-operation (enqueue) and, once it drains,
//     a real `user` line (promptSource "queued") with the same text. The enqueue
//     is dropped when its text later arrives as a real user line, so the message
//     shows once. Enqueues that never drain (the message reaches the turn as a
//     `queued_command` attachment instead) keep showing — they are the only trace
//     of it in the transcript.
//   - an assistant message is written once per streaming snapshot / content
//     block, all sharing the same message.id. They collapse into a single turn,
//     keeping the last (most complete) text.
type chatDedup struct {
	pending      []chatTurn // enqueues still awaiting their drain
	hasAssistant bool       // whether a prior assistant turn exists to collapse into
	lastIdx      int        // index of that assistant turn in turns
	turns        []chatTurn
}

// add processes one displayable transcript line in order.
func (d *chatDedup) add(role, text, source, msgID string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	t := chatTurn{role: role, text: text, source: source, msgID: msgID}
	switch {
	case source == "enqueue":
		d.pending = append(d.pending, t)
	case role == "user":
		for j := range d.pending {
			if d.pending[j].text == text {
				// the drain arrived: drop that enqueue (the real line stands for
				// it), flushing any enqueues that were queued before it.
				d.turns = append(d.turns, d.pending[:j]...)
				d.pending = d.pending[j+1:]
				d.append(t)
				return
			}
		}
		// not a drain of any pending enqueue: it is its own message.
		d.flushPending()
		d.append(t)
	case role == "assistant":
		if d.hasAssistant && d.turns[d.lastIdx].msgID != "" && d.turns[d.lastIdx].msgID == msgID {
			d.turns[d.lastIdx].text = text // streaming snapshot: keep the fuller one
			return
		}
		d.flushPending()
		d.append(t)
	default:
		d.flushPending()
		d.append(t)
	}
}

// flushPending emits the enqueues that never drained, in order.
func (d *chatDedup) flushPending() {
	d.turns = append(d.turns, d.pending...)
	d.pending = nil
}

func (d *chatDedup) append(t chatTurn) {
	if t.role == "assistant" {
		d.hasAssistant = true
		d.lastIdx = len(d.turns)
	} else {
		d.hasAssistant = false
	}
	d.turns = append(d.turns, t)
}

func (h *Host) sessionChat(name string) (map[string]any, error) {
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}
	id := h.convIDForSession(name)
	status := rcState(h.rcStatus(name))
	mode := h.paneMode(name)
	waiting, choice := h.paneWaitingWithChoice(name)
	var model string
	if id == "" {
		return map[string]any{
			"session": name, "id": "", "ready": false,
			"is_alive": true, "status": status, "mode": mode,
			"model":    model,
			"waiting":  waiting,
			"choice":   choice,
			"modes":    h.cachedModeWheel(),
			"messages": []map[string]any{},
		}, nil
	}

	path := h.convFileFor(id)
	info, err := os.Stat(path)
	if err != nil {
		// The id is pinned but Claude writes the transcript lazily, on the
		// first real message. Signal not-ready instead of a hard 404.
		return map[string]any{
			"session": name, "id": id, "ready": false,
			"is_alive": true, "status": status, "mode": mode,
			"model":    model,
			"waiting":  waiting,
			"choice":   choice,
			"modes":    h.cachedModeWheel(),
			"messages": []map[string]any{},
		}, nil
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, errServer("open conversation: %v", err)
	}
	defer fh.Close()

	var cwd string
	var created int64
	scanner := bufio.NewScanner(fh)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var dd chatDedup
	for scanner.Scan() {
		var line convLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Cwd != "" {
			cwd = line.Cwd
		}
		if created == 0 && line.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, line.Timestamp); err == nil {
				created = t.Unix()
			}
		}
		if line.Message.Model != "" {
			model = line.Message.Model
		}
		role, text, source, ok := chatRoleAndText(line)
		if ok {
			dd.add(role, text, source, line.Message.ID)
		}
	}
	dd.flushPending()
	// Cap to the most recent messages: the chat view is for the live session and
	// re-emits this payload on every SSE frame, so an unbounded transcript (tens
	// of MB) floods mobile clients. Full history lives in the conversations
	// browser, not here. Capping the deduplicated turns keeps index stable.
	msgs := make([]map[string]any, 0, len(dd.turns))
	for i, t := range dd.turns {
		msgs = append(msgs, map[string]any{
			"index":   i,
			"role":    t.role,
			"content": strings.TrimSpace(t.text),
		})
	}
	if len(msgs) > maxChatMsgs {
		msgs = msgs[len(msgs)-maxChatMsgs:]
	}
	if created == 0 {
		created = info.ModTime().Unix()
	}

	return map[string]any{
		"session": name, "id": id, "ready": true,
		"title":    conversationTitle(path),
		"origin":   originFor(cwd),
		"created":  created,
		"updated":  info.ModTime().Format(time.RFC3339Nano),
		"size":     info.Size(),
		"is_alive": true,
		"status":   status,
		"mode":     mode,
		"model":    model,
		"waiting":  waiting,
		"choice":   choice,
		"modes":    h.cachedModeWheel(),
		"messages": msgs,
	}, nil
}

// sessionStatus returns a compact, cheap status for the turn watcher: whether
// the pane is mid-generation, the waiting reason / choice picker (if any), and
// the last assistant message id + text preview from the transcript tail. It
// reads only the tail so polling every few seconds stays cheap on large
// transcripts (the full re-read happens in sessionChat).
func (h *Host) sessionStatus(name string) (map[string]any, error) {
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}
	waiting, choice := h.paneWaitingWithChoice(name)
	lastID, lastText := h.lastAssistant(name)
	return map[string]any{
		"session":             name,
		"working":             h.paneWorking(name),
		"waiting":             waiting,
		"choice":              choice,
		"last_assistant_id":   lastID,
		"last_assistant_text": lastText,
	}, nil
}

// paneWorking reports whether the session's pane is mid-generation rather than
// idle, reading the footer's hint tokens (verified on real footers):
//   - auto mode, working:  "… esc to interrupt · ctrl+t to hide tasks"
//   - auto mode, idle:     "… esc to interrupt · ← for agents"
//   - manual mode, working: "… manual mode on · esc to interrupt · ← for agents"
//   - manual mode, idle:   "… manual mode on · ? for shortcuts · ← for agents"
//
// Best-effort: the transcript-stability side of the watcher guards false hits.
func (h *Host) paneWorking(name string) bool {
	out, err := exec.Command(h.tmuxBinary, "capture-pane", "-p", "-t", h.paneTarget(name)).Output()
	if err != nil {
		return false
	}
	last := ""
	for _, l := range strings.Split(string(out), "\n") {
		if t := strings.TrimSpace(l); t != "" {
			last = t
		}
	}
	if last == "" {
		return false
	}
	switch {
	case strings.Contains(last, "? for shortcuts"): // manual idle
		return false
	case strings.Contains(last, "ctrl+t to hide"): // auto working
		return true
	case strings.Contains(last, "manual mode") && strings.Contains(last, "esc to interrupt"):
		return true // manual working
	default:
		// auto idle (has "← for agents") vs other working forms
		return strings.Contains(last, "esc to interrupt") && !strings.Contains(last, "← for agents")
	}
}

// lastAssistant reads the tail of the session's transcript and returns the id
// and text of the last assistant message. Streaming snapshots collapse onto the
// same message.id, so the final record is the complete one. "" when none.
func (h *Host) lastAssistant(name string) (id, text string) {
	cid := h.convIDForSession(name)
	if cid == "" {
		return "", ""
	}
	f, err := os.Open(h.convFileFor(cid))
	if err != nil {
		return "", ""
	}
	defer f.Close()
	const maxTail = 256 * 1024
	st, err := f.Stat()
	if err != nil {
		return "", ""
	}
	off := st.Size() - maxTail
	if off < 0 {
		off = 0
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return "", ""
	}
	buf := make([]byte, st.Size()-off)
	if _, err := io.ReadFull(f, buf); err != nil && err != io.EOF {
		return "", ""
	}
	scanner := bufio.NewScanner(bytes.NewReader(buf))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	first := true
	for scanner.Scan() {
		if first { // the first line may be cut mid-record
			first = false
			continue
		}
		var line convLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type == "assistant" {
			// Track the id unconditionally: a turn whose final record carries no
			// text (only tool_use, or an error) must still advance the watcher,
			// or its completion is never announced. Text stays best-effort.
			id = line.Message.ID
			if t, ok := extractText(line.Message.Content); ok {
				text = t
			}
		}
	}
	return id, text
}

// paneMode does a best-effort read of Claude Code's footer (last visible line
// of the pane) for the current mode token. The default footer has no mode
// name, so this often returns "" — the chat footer then omits it.
func (h *Host) paneMode(name string) string {
	out, err := exec.Command(h.tmuxBinary, "capture-pane", "-p", "-t", h.paneTarget(name)).Output()
	if err != nil {
		return ""
	}
	last := ""
	for _, l := range strings.Split(string(out), "\n") {
		if t := strings.TrimSpace(l); t != "" {
			last = t
		}
	}
	return modeFromBadge(last)
}

// paneWaiting reports whether the session's pane is blocked on an interactive
// prompt (e.g. command-permission approval) that the transcript won't contain.
// Non-empty signals the chat view that the session is waiting, not working.
func (h *Host) paneWaiting(name string) string {
	w, _ := h.paneWaitingWithChoice(name)
	return w
}

// paneWaitingWithChoice reports the waiting reason plus, when the pane shows an
// AskUserQuestion option picker (reason "choice"), the parsed question, options
// and selected index so the chat can render it.
func (h *Host) paneWaitingWithChoice(name string) (string, map[string]any) {
	out, err := exec.Command(h.tmuxBinary, "capture-pane", "-p", "-t", h.paneTarget(name)).Output()
	if err != nil {
		return "", nil
	}
	pane := string(out)
	reason := paneWaitingReason(pane)
	if reason != "choice" {
		return reason, nil
	}
	q, opts, sel, ok := paneChoice(pane)
	if !ok {
		return reason, nil
	}
	return reason, map[string]any{
		"question": q,
		"options":  opts,
		"selected": sel,
	}
}

// modeFromBadge extracts the live mode token from a footer line. The badge
// renders "accept edits" with a space (⏵⏵ accept edits on), not the config
// key acceptEdits — map it back here so callers compare against one spelling.
func modeFromBadge(line string) string {
	for _, m := range []string{"insert", "accept edits", "edit", "manual", "plan", "auto", "bypassPermissions", "budget", "normal"} {
		if strings.Contains(line, m) {
			if m == "accept edits" {
				return "accept-edits"
			}
			return m
		}
	}
	return ""
}

// maxSendLen caps a chat message sent into a live session.
const maxSendLen = 2000

// tmuxKeyMap is the closed whitelist of special keys a client may send to a
// live session (mapped to tmux key names). Text is always sent literally.
var tmuxKeyMap = map[string]string{
	"enter":  "Enter",
	"ctrl-c": "C-c",
	"escape": "Escape",
	"up":     "Up",
	"down":   "Down",
	"tab":    "Tab",
	"btab":   "BTab",
	"space":  "Space",
}

// sessionSend types a message into a live session (literal text + Enter) or
// sends one whitelisted special key. Mirrors claudeRename's send-keys pattern.
func (h *Host) sessionSend(name, text, key string) (map[string]any, error) {
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}
	if text == "" && key == "" {
		return nil, errBad("nothing to send")
	}
	if text != "" {
		if len([]rune(text)) > maxSendLen {
			return nil, errBad("text too long (max %d chars)", maxSendLen)
		}
		if err := h.sendKeys(name, text); err != nil {
			return nil, err
		}
		return map[string]any{"session": name, "sent": "text"}, nil
	}
	tmuxKey, ok := tmuxKeyMap[key]
	if !ok {
		return nil, errBad("unsupported key: %s", key)
	}
	if err := exec.Command(h.tmuxBinary, "send-keys", "-t", h.paneTarget(name), tmuxKey).Run(); err != nil {
		return nil, errServer("tmux send-keys: %v", err)
	}
	return map[string]any{"session": name, "sent": key}, nil
}

// paneWaitingReason detects when a live session is blocked on an interactive
// TUI question — the command-permission dialog ("Do you want to proceed?") is
// the common one. These prompts never reach the transcript, so the chat view
// would otherwise show a silent session with no sign it is waiting on the user.
func paneWaitingReason(pane string) string {
	// Choice dialog (AskUserQuestion with options): footer "Enter to select ·
	// ↑/↓ to navigate · n to add notes · Esc to cancel". The user picks one of
	// several options, not yes/no.
	if strings.Contains(pane, "Enter to select") || strings.Contains(pane, "to navigate") {
		return "choice"
	}
	// Approval dialogs: command permission ("Do you want to proceed"), the generic
	// "requires approval" line, the file-edit dialog ("Do you want to make this
	// edit to <file>?"), and — as a catch-all — the shared footer "Esc to cancel ·
	// Tab to amend". The normal running footer ("⏵⏵ auto mode on (shift+tab to
	// cycle) · ← for agents") never contains it.
	if strings.Contains(pane, "Do you want to proceed") ||
		strings.Contains(pane, "requires approval") ||
		strings.Contains(pane, "Do you want to make this edit") ||
		strings.Contains(pane, "Esc to cancel") {
		return "approval"
	}
	return ""
}

// paneChoice extracts an AskUserQuestion option picker from the pane text:
// the question line, the option labels in order, and the currently highlighted
// index (the one prefixed with ❯). The picker renders as
//
//	☐ <title>
//	<question line>
//	❯ 1. <option A>
//	  2. <option B>
//	  …
//	Enter to select · ↑/↓ to navigate · n to add notes · Esc to cancel
//
// ok=false when no numbered option list is present.
func paneChoice(pane string) (question string, options []string, selected int, ok bool) {
	lines := strings.Split(pane, "\n")
	firstOpt := -1
	for i, l := range lines {
		if m := choiceOptionRe.FindStringSubmatch(l); m != nil {
			firstOpt = i
			break
		}
	}
	if firstOpt < 0 {
		return "", nil, 0, false
	}
	// Question = the last non-empty line above the first option.
	for i := firstOpt - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" && !strings.Contains(t, "☐") {
			question = t
			break
		}
	}
	for _, l := range lines[firstOpt:] {
		m := choiceOptionRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		if strings.TrimSpace(m[1]) == "❯" {
			selected = len(options)
		}
		options = append(options, strings.TrimSpace(m[2]))
	}
	return question, options, selected, true
}

// choiceOptionRe matches a numbered option line, optionally with the ❯ cursor:
// "❯ 1. Label" or "  2. Label". Group 1 is the cursor token ("" or "❯"),
// group 2 the label (drops the number and any trailing panel box-drawing).
var choiceOptionRe = regexp.MustCompile(`^\s*(❯)?\s*\d+\.\s+([^\n┌─│┐└┘]*)\s*$`)

// modeWheel is the Shift+Tab cycle of Claude Code's live modes. Order verified
// empirically on 2.1.227 (account without bypassPermissions): the badge cycles
// manual → accept-edits → plan → auto → manual. plan is the deterministic
// anchor: /plan always enters plan, so any other mode is reachable from it
// with N shift+tabs, even when the current mode is unknown. auto (and
// bypassPermissions) only appear on the wheel if the account enables them, so
// the order is a single knob to adjust per account.
var modeWheel = []string{"manual", "accept-edits", "plan", "auto"}

// modePosIn returns the position of m on an arbitrary wheel, or -1.
func modePosIn(wheel []string, m string) int {
	for i, x := range wheel {
		if x == m {
			return i
		}
	}
	return -1
}

// distanceIn returns how many Shift+Tab presses take wheel from mode from to
// mode to (both must be on it). ok=false when either is not on it.
func distanceIn(wheel []string, from, to string) (n int, ok bool) {
	fi, ti := modePosIn(wheel, from), modePosIn(wheel, to)
	if fi < 0 || ti < 0 {
		return 0, false
	}
	return (ti - fi + len(wheel)) % len(wheel), true
}

// modeDistance is distanceIn over the standard wheel (kept for tests that
// reason about the default account layout).
func modeDistance(from, to string) (n int, ok bool) {
	return distanceIn(modeWheel, from, to)
}

// modePressDelay pauses between raw shift+tab presses so Claude registers each
// cycle before the next key.
const modePressDelay = 500 * time.Millisecond

// maxModeCycle bounds any wheel walk: real wheels are 3-5 modes, and the
// probe/restore loops must always terminate even when the badge never becomes
// readable again.
const maxModeCycle = 12

// rawShiftTab sends one Shift+Tab as raw bytes (\e[Z). tmux's S-Tab key name
// does not reach Claude's mode wheel (verified): only the literal escape
// sequence does.
func (h *Host) rawShiftTab(name string) error {
	if err := exec.Command(h.tmuxBinary, "send-keys", "-t", h.paneTarget(name), "-l", "\x1b[Z").Run(); err != nil {
		return errServer("tmux send-keys: %v", err)
	}
	time.Sleep(modePressDelay)
	return nil
}

// cachedModeWheel returns the discovered wheel for the active profile without
// probing (nil when it hasn't been calibrated yet). sessionChat uses it so the
// UI dropdown reflects the account's real modes once discovered.
func (h *Host) cachedModeWheel() []string {
	profile := h.activeProfileName()
	h.modeMu.Lock()
	defer h.modeMu.Unlock()
	return h.modeWheelCache[profile]
}

// wheelFor returns the Shift+Tab wheel to drive a session's modes. It probes
// the live session once per profile (walking the real wheel) and caches the
// order; when the probe is impossible (session working or in a dialog, or the
// badge unreadable) it falls back to the standard wheel, matching the previous
// behaviour.
func (h *Host) wheelFor(name string) []string {
	profile := h.activeProfileName()
	h.modeMu.Lock()
	w, ok := h.modeWheelCache[profile]
	h.modeMu.Unlock()
	if ok {
		return w
	}
	if w, err := h.discoverModeWheel(name); err == nil && len(w) >= 2 {
		h.modeMu.Lock()
		h.modeWheelCache[profile] = w
		h.modeMu.Unlock()
		return w
	}
	return append([]string(nil), modeWheel...)
}

// discoverModeWheel walks a live session's Shift+Tab wheel, recording the mode
// badge after each press until the sequence repeats (a full cycle). The wheel
// order is account-dependent — auto/bypassPermissions only appear when enabled
// — so distances derived from it are correct for any account. The session must
// be idle and its badge readable; a full cycle ends exactly at the start, so
// the session is left untouched. On any unreadable badge the probe restores the
// starting mode (best effort) and returns an error so the caller falls back.
func (h *Host) discoverModeWheel(name string) ([]string, error) {
	if h.paneWorking(name) {
		return nil, errServer("session %s is working; can't calibrate the mode wheel", name)
	}
	if waiting, _ := h.paneWaitingWithChoice(name); waiting != "" {
		return nil, errServer("session %s is waiting; can't calibrate the mode wheel", name)
	}
	start := h.paneMode(name) // already the parsed token ("" when unreadable)
	if start == "" {
		return nil, errServer("can't read the current mode badge to calibrate")
	}
	seen := []string{start}
	for i := 0; i < maxModeCycle; i++ {
		if err := h.rawShiftTab(name); err != nil {
			return nil, err
		}
		b := h.paneMode(name)
		if b == "" {
			h.restoreMode(name, start)
			return nil, errServer("mode badge unreadable while calibrating")
		}
		if b == start {
			return seen, nil // full cycle: the session is back where it started
		}
		if modePosIn(seen, b) >= 0 {
			h.restoreMode(name, start)
			return nil, errServer("mode wheel repeated %q mid-cycle", b)
		}
		seen = append(seen, b)
	}
	h.restoreMode(name, start)
	return nil, errServer("mode wheel did not complete a cycle")
}

// restoreMode cycles a session forward until its badge shows want (bounded),
// best effort: used to undo a probe that aborted mid-cycle.
func (h *Host) restoreMode(name, want string) {
	for i := 0; i < maxModeCycle; i++ {
		if h.paneMode(name) == want {
			return
		}
		if err := h.rawShiftTab(name); err != nil {
			return
		}
	}
}

// sessionMode changes a live session's mode. plan is sent as the /plan slash
// command (deterministic from anywhere); the rest are reached via the real
// Shift+Tab wheel (discovered per account), either from the current mode (when
// the badge is readable) or anchored at plan via /plan when it isn't.
func (h *Host) sessionMode(name, target string) (map[string]any, error) {
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}
	if target == "plan" {
		if err := h.sendKeys(name, "/plan"); err != nil {
			return nil, err
		}
		return map[string]any{"session": name, "mode": target, "pressed": 1}, nil
	}
	wheel := h.wheelFor(name)
	if modePosIn(wheel, target) < 0 {
		return nil, errBad("unsupported mode %s (available: %s)", target, strings.Join(wheel, ", "))
	}
	cur := h.paneMode(name)
	n, ok := distanceIn(wheel, cur, target)
	if !ok {
		// Badge unreadable (pane in a dialog, starting up, etc.): anchor at
		// plan, which /plan always reaches, and count from there. The /plan
		// itself counts as one more move.
		if err := h.sendKeys(name, "/plan"); err != nil {
			return nil, err
		}
		n, _ = distanceIn(wheel, "plan", target)
		n++
	}
	for i := 0; i < n; i++ {
		if err := h.rawShiftTab(name); err != nil {
			return nil, err
		}
	}
	return map[string]any{"session": name, "mode": target, "pressed": n}, nil
}

// rcPressDelay is the pause between the two /remote-control presses when a
// session already has RC up, so the toggle-off completes before the toggle-on.
const rcPressDelay = 700 * time.Millisecond

// sessionRc forces a fresh Remote Control re-registration in a live session.
// /remote-control is a toggle: when the heuristic says RC is up it's typed
// twice (off then on — a fresh bridge id on claude.ai); when it's down or
// unknown, a single press enables it.
//
// The command only exists under an Anthropic endpoint: a session launched under
// a perfilSinRC profile (deepseek, custom base URL) rejects it as "Unknown
// command". Such sessions are re-staged exactly like a fresh launch
// (lanzarConStaging): bootstrap the clean profile → send /remote-control →
// wait for the real bridge → restore the target profile. Returns presses sent
// and, when staged, the wait outcome.
func (h *Host) sessionRc(name string) (map[string]any, error) {
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}

	// Limpia el input residual antes de teclear: un comando slash solo se ejecuta
	// si el `/` queda al inicio del input. Con texto pendiente (p.ej. un mensaje
	// que el móvil no llegó a enviar), /remote-control se pegaría al texto y se
	// mandaría como mensaje de usuario en lugar de como comando.
	if err := h.clearInput(name); err != nil {
		return nil, err
	}

	// Si Claude está en mitad de una generación, el comando se encola detrás del
	// turno; y si luego no aparece el bridge, la auto-recuperación no debe matar
	// una sesión que está trabajando. Se avisa y se para.
	if h.paneWorking(name) {
		return nil, errBad("Claude está trabajando en la sesión %s; espera a que termine antes de re-registrar el Remote Control", name)
	}

	activo := h.activeProfileName()
	staging := activo != "" && perfilSinRC(h.profilesPath+"/"+activo+".json")
	if staging {
		if err := h.applyProfile(h.rcBootstrap); err != nil {
			return nil, errServer("bootstrap profile: %v", err)
		}
	}
	// Decide las pulsaciones por el bridge REAL (rcStatusLive), no por rcStatus:
	// el fallback argv de rcStatus reporta rc_connected en cuanto el proceso lleva
	// --remote-control, aunque el bridge esté caído, y haría un toggle off+on que
	// no re-registra nada.
	presses := 1
	if h.rcStatusLive(name) == "rc_connected" {
		presses = 2
	}
	for i := 0; i < presses; i++ {
		if err := h.sendKeys(name, "/remote-control"); err != nil {
			if staging {
				h.applyProfile(activo) // best effort
			}
			return nil, err
		}
		if i+1 < presses {
			time.Sleep(rcPressDelay)
		}
	}

	out := map[string]any{"session": name, "presses": presses, "staging": staging}
	if staging {
		out["status"] = h.waitRCBridgeSettled(name)
		if err := h.applyProfile(activo); err != nil {
			return nil, errServer("restore profile: %v", err)
		}
	} else {
		out["status"] = h.waitRCBridgeSettled(name)
	}

	// El bridge no se ha recuperado en vivo: un proceso ya degradado no puede
	// re-registrarlo (perfil sin RC / 4090 "no longer the active worker"), y el
	// staging no lo revierte. La vía fiable es relanzar: matar la sesión y retomar
	// la conversación, cuyo arranque en dos fases sí registra el bridge. Eso es lo
	// que "recolectar" significa cuando el re-tecleo no basta.
	if out["status"] != "ok" {
		conv := h.convIDForSession(name)
		if conv == "" {
			return out, nil // sin id de conversación no se puede relanzar; se reporta el fallo
		}
		if err := h.tmuxKill(name); err != nil {
			return out, err
		}
		resumed, err := h.claudeResume(conv)
		if err != nil {
			return out, err
		}
		out["recovered"] = true
		out["session"] = resumed["session"]
		out["status"] = rcState(resumed["status"])
	}
	return out, nil
}

// clearInput borra el texto residual del input del TUI (C-u) antes de teclear un
// comando slash, para que el `/` quede al inicio y el comando se ejecute.
func (h *Host) clearInput(name string) error {
	if err := exec.Command(h.tmuxBinary, "send-keys", "-t", h.paneTarget(name), "C-u").Run(); err != nil {
		return errServer("tmux send-keys C-u: %v", err)
	}
	return nil
}

// activeSessionUsingConv devuelve el nombre de una sesión viva que tiene <id>
// como conversación y el bridge del móvil registrado. Es el caso en que retomar
// <id> desde otra sesión desconectaría la existente (code 4090).
func (h *Host) activeSessionUsingConv(id string) string {
	sessions, err := h.tmuxList()
	if err != nil {
		return ""
	}
	for _, s := range sessions {
		name := s["name"].(string)
		if h.convIDForSession(name) == id && h.sessionBridgeID(name) != "" {
			return name
		}
	}
	return ""
}

func (h *Host) tmuxRename(name, newName string) (map[string]string, error) {
	if newName == name {
		return nil, errBad("new name must differ")
	}
	if h.sessionAlive(newName) {
		return nil, errBad("session name already in use: %s", newName)
	}
	if !h.sessionAlive(name) {
		return nil, errNotFound("session not found: %s", name)
	}
	out, err := exec.Command(h.tmuxBinary, "rename-session", "-t", "="+name, newName).CombinedOutput()
	if err != nil {
		return nil, errServer("tmux rename-session: %s", strings.TrimSpace(string(out)))
	}
	return map[string]string{"old_name": name, "new_name": newName}, nil
}

func (h *Host) claudeRename(session, title string) (map[string]string, error) {
	if !h.sessionAlive(session) {
		return nil, errNotFound("session not found: %s", session)
	}
	if err := h.sendKeys(session, "/rename "+title); err != nil {
		return nil, err
	}
	return map[string]string{"session": session, "name": title}, nil
}

func (h *Host) sendKeys(session, literal string) error {
	target := h.paneTarget(session)
	if err := exec.Command(h.tmuxBinary, "send-keys", "-t", target, "-l", literal).Run(); err != nil {
		return errServer("tmux send-keys: %v", err)
	}
	return exec.Command(h.tmuxBinary, "send-keys", "-t", target, "Enter").Run()
}

func (h *Host) renameClaudeAfterReady(session, title string, waitRC bool) {
	deadline := time.Now().Add(time.Duration(h.rcWaitSeconds) * time.Second)
	for {
		if !h.sessionAlive(session) {
			return
		}
		if waitRC {
			switch h.rcStatusLive(session) {
			case "rc_connected":
				waitRC = false
			case "rc_failed":
				return
			}
		}
		if !waitRC {
			h.sendKeys(session, "/rename "+title)
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(time.Duration(h.rcPollSeconds) * time.Second)
	}
}

// aliveConversations returns the set of conversation UUIDs currently being
// written by a running Claude Code process. Resumed sessions carry their
// --resume <uuid> in argv; freshly-started sessions do not, so while a claude
// process is running we also mark any conversation whose .jsonl was modified
// in the last minute.
func (h *Host) aliveConversations() map[string]bool {
	alive := map[string]bool{}
	out, err := exec.Command("ps", "-eo", "comm,args").Output()
	if err != nil {
		return alive
	}
	claudeRunning := false
	bin := filepath.Base(h.claudeBinary)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == bin {
			claudeRunning = true
		}
		for _, tok := range fields[1:] {
			if uuidPattern.MatchString(tok) {
				alive[tok] = true
			}
		}
	}
	if !claudeRunning {
		return alive
	}

	// Fresh sessions never show their uuid in argv (no --resume). While claude
	// is running, any conversation modified in the last minute is that session.
	cutoff := time.Now().Add(-time.Minute)
	files, _ := h.convFiles()
	for _, f := range files {
		if f.mod.After(cutoff) {
			alive[f.id] = true
		}
	}
	return alive
}
