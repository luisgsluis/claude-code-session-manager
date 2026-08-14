package auth

import (
	"sync"
	"time"
)

// RateLimiter throttles failed authentication attempts per client IP.
//
// It exists because CCSM is commonly published to the internet behind a
// reverse proxy (see docs/security.md): a login form and a 6-digit TOTP code
// are both brute-forceable in minutes without a limit. Password and TOTP
// failures share one counter on purpose — counting them separately would leave
// the second factor as an open window.
//
// State is in memory, like the session store. A restart clears the counters;
// the durable record of the abuse is the audit log, which is what the homelab
// ipban detector reads.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rlEntry

	max    int           // failures allowed inside window
	window time.Duration // sliding window the failures are counted over
	block  time.Duration // how long a blocked IP stays blocked
}

type rlEntry struct {
	fails   []time.Time
	blocked time.Time // zero = not blocked
}

// Default policy: 5 failures within 15 minutes block the IP for 15 minutes.
const (
	DefaultLoginMaxFails = 5
	DefaultLoginWindow   = 15 * time.Minute
	DefaultLoginBlock    = 15 * time.Minute
)

// NewRateLimiter builds a limiter. Zero or negative max disables it.
func NewRateLimiter(max int, window, block time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*rlEntry),
		max:     max,
		window:  window,
		block:   block,
	}
}

// NewLoginRateLimiter is NewRateLimiter with the default login policy.
func NewLoginRateLimiter() *RateLimiter {
	return NewRateLimiter(DefaultLoginMaxFails, DefaultLoginWindow, DefaultLoginBlock)
}

// Blocked reports whether ip is currently blocked.
func (rl *RateLimiter) Blocked(ip string) bool {
	if rl == nil || rl.max <= 0 {
		return false
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.blockedAt(ip, time.Now())
}

func (rl *RateLimiter) blockedAt(ip string, now time.Time) bool {
	e, ok := rl.entries[ip]
	if !ok {
		return false
	}
	return now.Before(e.blocked)
}

// Fail records one failed attempt from ip. It returns true only on the
// transition into a block, so the caller audits the block once instead of
// once per attempt.
func (rl *RateLimiter) Fail(ip string) (justBlocked bool) {
	if rl == nil || rl.max <= 0 || ip == "" {
		return false
	}
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.sweep(now)

	e, ok := rl.entries[ip]
	if !ok {
		e = &rlEntry{}
		rl.entries[ip] = e
	}
	if now.Before(e.blocked) {
		return false // already blocked; don't re-announce
	}

	// Drop failures that fell out of the window before counting this one.
	kept := e.fails[:0]
	for _, t := range e.fails {
		if now.Sub(t) < rl.window {
			kept = append(kept, t)
		}
	}
	e.fails = append(kept, now)

	if len(e.fails) >= rl.max {
		e.blocked = now.Add(rl.block)
		e.fails = nil
		return true
	}
	return false
}

// Reset clears an IP's history, called after a successful authentication so a
// user who mistyped a few times isn't left one slip away from a block.
func (rl *RateLimiter) Reset(ip string) {
	if rl == nil || ip == "" {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.entries, ip)
}

// sweep drops entries with no live block and no failure inside the window, so
// a port scan cycling through source IPs cannot grow the map without bound.
// Called under the lock from Fail.
func (rl *RateLimiter) sweep(now time.Time) {
	for ip, e := range rl.entries {
		if now.Before(e.blocked) {
			continue
		}
		live := false
		for _, t := range e.fails {
			if now.Sub(t) < rl.window {
				live = true
				break
			}
		}
		if !live {
			delete(rl.entries, ip)
		}
	}
}
