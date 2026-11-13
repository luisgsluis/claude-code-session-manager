package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// The LAN bypass authenticates by IP, so a page on any other site open in a LAN
// browser must not be able to drive sessions with a simple (no-preflight) POST.
func TestCrossSiteGuard(t *testing.T) {
	srv := newTestServer(t)
	cases := []struct {
		name, method, path string
		headers            map[string]string
		refused            bool
	}{
		{"cross-site fetch to send", "POST", "/api/sessions/0/send", map[string]string{"Sec-Fetch-Site": "cross-site"}, true},
		{"same-site (other .lan host)", "POST", "/api/sessions/new", map[string]string{"Sec-Fetch-Site": "same-site"}, true},
		{"cross-site delete", "DELETE", "/api/sessions/0", map[string]string{"Sec-Fetch-Site": "cross-site"}, true},
		{"cross-site login", "POST", "/api/auth/login", map[string]string{"Sec-Fetch-Site": "cross-site"}, true},
		{"own UI", "POST", "/api/sessions/new", map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://example.com"}, false},
		{"typed / bookmarked", "POST", "/api/sessions/new", map[string]string{"Sec-Fetch-Site": "none"}, false},
		{"old browser, foreign Origin", "POST", "/api/sessions/new", map[string]string{"Origin": "http://evil.test"}, true},
		{"old browser, null Origin", "POST", "/api/sessions/new", map[string]string{"Origin": "null"}, true},
		{"old browser, own Origin", "POST", "/api/sessions/new", map[string]string{"Origin": "http://example.com"}, false},
		{"bot / curl (no headers)", "POST", "/api/sessions/new", nil, false},
		{"cross-site GET is not a write", "GET", "/api/health", map[string]string{"Sec-Fetch-Site": "cross-site"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(`{"name":"x"}`))
			req.RemoteAddr = "127.0.0.1:4321" // LAN bypass: no cookie needed
			req.Header.Set("Content-Type", "text/plain")
			for k, v := range c.headers {
				req.Header.Set(k, v)
			}
			w := serve(srv, req)
			refused := w.Code == 403 && strings.Contains(w.Body.String(), "cross-site")
			if refused != c.refused {
				t.Fatalf("code %d body %q: refused=%v, want %v", w.Code, w.Body.String(), refused, c.refused)
			}
			if refused && w.Header().Get("X-Frame-Options") != "DENY" {
				t.Errorf("refusal skipped the security headers")
			}
		})
	}
}
