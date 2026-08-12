package auth

import (
	"net"
	"net/http"
	"strings"
)

// IsLAN returns true if the request remote address belongs to any of the
// configured LAN/VPN subnets. The remote address may include a port
// ("192.168.1.5:12345") or be bare.
func IsLAN(remoteAddr string, subnets []string) bool {
	return inSubnets(remoteAddr, subnets)
}

// ClientIP returns the client's real IP for the LAN-bypass decision.
//
// Behind the caddy reverse proxy every request reaches this container from the
// same peer (the docker gateway, 172.24.0.1), and caddy overwrites the
// X-Forwarded-For header with the real client IP ({remote_host}). We therefore
// trust that header only when the immediate peer is a known proxy — otherwise
// a client connecting directly to :8080 could forge an XFF value and fake a LAN
// source. If the peer is untrusted, or no header is present, the peer itself is
// the client.
func ClientIP(r *http.Request, trusted []string) string {
	if !inSubnets(r.RemoteAddr, trusted) {
		return r.RemoteAddr
	}
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff == "" {
		return r.RemoteAddr
	}
	if i := strings.IndexByte(xff, ','); i >= 0 {
		xff = xff[:i] // first hop is the client; anything after it is our own proxies
	}
	return strings.TrimSpace(xff)
}

func inSubnets(remoteAddr string, subnets []string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr // no port
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	// Unwrap IPv4-mapped IPv6 (::ffff:1.2.3.4) so a client behind a dual-stack
	// listener matches IPv4 subnets.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	for _, cidr := range subnets {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, netw, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if netw.Contains(ip) {
			return true
		}
	}
	return false
}
