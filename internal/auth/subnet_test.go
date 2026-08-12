package auth

import (
	"net/http/httptest"
	"testing"
)

func TestIsLANLocalhost(t *testing.T) {
	subnets := []string{"127.0.0.0/8"}
	if !IsLAN("127.0.0.1:54321", subnets) {
		t.Error("127.0.0.1 should be LAN")
	}
	if !IsLAN("127.0.0.1", subnets) {
		t.Error("127.0.0.1 (no port) should be LAN")
	}
}

func TestIsLANPrivate(t *testing.T) {
	subnets := []string{"192.168.1.0/24", "10.0.0.0/8"}

	tests := []struct {
		addr  string
		isLAN bool
	}{
		{"192.168.1.5:8080", true},
		{"192.168.1.254", true},
		{"192.168.2.1:1234", false},
		{"10.0.0.1:9999", true},
		{"10.255.255.255", true},
		{"172.16.0.1:80", false},
		{"8.8.8.8:443", false},
		{"[::1]:8080", false}, // IPv6 loopback
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsLAN(tt.addr, subnets)
		if result != tt.isLAN {
			t.Errorf("IsLAN(%q, %v) = %v, want %v", tt.addr, subnets, result, tt.isLAN)
		}
	}
}

func TestIsLANEmptySubnets(t *testing.T) {
	if IsLAN("192.168.1.5", nil) {
		t.Error("nil subnets should return false")
	}
	if IsLAN("192.168.1.5", []string{}) {
		t.Error("empty subnets should return false")
	}
}

func TestIsLANInvalidCIDR(t *testing.T) {
	subnets := []string{"not-a-cidr", "192.168.1.0/24"}
	if !IsLAN("192.168.1.100", subnets) {
		t.Error("should match valid CIDR even if another is invalid")
	}
}

func TestIsLANIPv6(t *testing.T) {
	subnets := []string{"fd00::/8"}
	if !IsLAN("[fd00::1]:8080", subnets) {
		t.Error("IPv6 ULA should match fd00::/8")
	}
}

func TestClientIP(t *testing.T) {
	trusted := []string{"172.24.0.1/32"}

	tests := []struct {
		name    string
		remote  string
		xff     string
		trusted []string
		want    string
	}{
		{"trusted proxy, real client in XFF", "172.24.0.1:12345", "192.168.1.50", trusted, "192.168.1.50"},
		{"trusted proxy, chained XFF takes first hop", "172.24.0.1:12345", "192.168.1.50, 8.8.8.8", trusted, "192.168.1.50"},
		{"trusted proxy, no XFF falls back to peer", "172.24.0.1:12345", "", trusted, "172.24.0.1:12345"},
		{"untrusted peer ignores forged XFF", "192.168.1.10:54321", "192.168.1.50", trusted, "192.168.1.10:54321"},
		{"no trusted proxies configured", "172.24.0.1:12345", "192.168.1.50", nil, "172.24.0.1:12345"},
		{"untrusted peer, no XFF", "8.8.8.8:443", "", trusted, "8.8.8.8:443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remote
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := ClientIP(r, tt.trusted); got != tt.want {
				t.Errorf("ClientIP(remote=%q, xff=%q) = %q, want %q", tt.remote, tt.xff, got, tt.want)
			}
		})
	}
}
