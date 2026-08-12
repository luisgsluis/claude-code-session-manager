package server

import (
	"log"
	"net/http"
	"time"
)

// withSecurityHeaders adds OWASP-recommended security headers.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()")
		// 'unsafe-eval' is required by Alpine.js (it compiles x-data expressions with new Function).
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-eval'")
		next.ServeHTTP(w, r)
	})
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
