package server

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// withSecurityHeaders adds OWASP-recommended security headers.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// microphone=(self), not (): the voice dictation button needs
		// getUserMedia / SpeechRecognition, and a bare () denies the mic to this
		// origin too. Denied that way there is no console error and no
		// permission prompt — the button just silently does nothing. Camera
		// stays denied: nothing in CCSM uses it.
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(self), geolocation=(), interest-cohort=()")
		// 'unsafe-eval' is required by Alpine.js (it compiles x-data expressions with new Function).
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-eval'")
		next.ServeHTTP(w, r)
	})
}

// withCrossSiteGuard rejects state-changing requests that a browser sends on
// behalf of another site. The LAN bypass authenticates by IP, not by cookie, so
// any page open in a LAN browser could otherwise POST (text/plain, no preflight)
// to /api/sessions/{name}/send. Non-browser clients send neither header and pass.
func withCrossSiteGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && crossSite(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"cross-site request refused"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// crossSite: Sec-Fetch-Site when the browser sends it; otherwise an Origin
// whose host is not the one the request was made to.
func crossSite(r *http.Request) bool {
	if sfs := r.Header.Get("Sec-Fetch-Site"); sfs != "" {
		return sfs != "same-origin" && sfs != "none"
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return true // "null" (sandboxed iframe, file://) or garbage
	}
	return !strings.EqualFold(u.Host, r.Host)
}

// withLogging logs each request with method, path, status, and duration.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lrw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, lrw.status, time.Since(start))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.status = code
	l.ResponseWriter.WriteHeader(code)
}

// Flush promotes the underlying writer so SSE handlers can use it through the
// logging wrapper. Without it w.(http.Flusher) fails in production.
func (l *loggingResponseWriter) Flush() {
	if f, ok := l.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.NewResponseController reach deadline controls on the real
// writer (SetWriteDeadline) instead of reporting ErrNotSupported.
func (l *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return l.ResponseWriter
}
