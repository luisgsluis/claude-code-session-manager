package auth

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksAtThreshold(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute, time.Minute)

	for i := 0; i < 2; i++ {
		if rl.Fail("1.2.3.4") {
			t.Fatalf("blocked after %d failures, threshold is 3", i+1)
		}
		if rl.Blocked("1.2.3.4") {
			t.Fatalf("reported blocked after %d failures", i+1)
		}
	}
	if !rl.Fail("1.2.3.4") {
		t.Fatal("the third failure did not trigger the block")
	}
	if !rl.Blocked("1.2.3.4") {
		t.Fatal("not blocked after reaching the threshold")
	}
}

// The transition is what gets audited, so it must be reported exactly once.
func TestRateLimiterAnnouncesBlockOnce(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute, time.Minute)
	rl.Fail("1.2.3.4")
	if !rl.Fail("1.2.3.4") {
		t.Fatal("block not announced")
	}
	if rl.Fail("1.2.3.4") {
		t.Error("block announced twice")
	}
}

func TestRateLimiterIsolatesIPs(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute, time.Minute)
	rl.Fail("1.2.3.4")
	rl.Fail("1.2.3.4")
	if !rl.Blocked("1.2.3.4") {
		t.Fatal("1.2.3.4 should be blocked")
	}
	if rl.Blocked("5.6.7.8") {
		t.Error("an unrelated IP was blocked")
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute, time.Minute)
	rl.Fail("1.2.3.4")
	rl.Fail("1.2.3.4")
	rl.Reset("1.2.3.4")

	// After a reset the counter starts over: two more failures must not block.
	rl.Fail("1.2.3.4")
	rl.Fail("1.2.3.4")
	if rl.Blocked("1.2.3.4") {
		t.Error("Reset did not clear the failure history")
	}
}

func TestRateLimiterWindowExpires(t *testing.T) {
	rl := NewRateLimiter(3, 40*time.Millisecond, time.Minute)
	rl.Fail("1.2.3.4")
	rl.Fail("1.2.3.4")
	time.Sleep(60 * time.Millisecond)

	// The two old failures fell out of the window; two fresh ones must not
	// reach the threshold of three.
	rl.Fail("1.2.3.4")
	rl.Fail("1.2.3.4")
	if rl.Blocked("1.2.3.4") {
		t.Error("failures outside the window still counted")
	}
}

func TestRateLimiterBlockExpires(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute, 40*time.Millisecond)
	rl.Fail("1.2.3.4")
	rl.Fail("1.2.3.4")
	if !rl.Blocked("1.2.3.4") {
		t.Fatal("not blocked")
	}
	time.Sleep(60 * time.Millisecond)
	if rl.Blocked("1.2.3.4") {
		t.Error("still blocked after the block expired")
	}
}

func TestRateLimiterSweepsStaleEntries(t *testing.T) {
	// Threshold high enough that none of these IPs gets blocked: a blocked
	// entry is kept on purpose, and only unblocked stale ones are swept.
	rl := NewRateLimiter(100, 40*time.Millisecond, time.Minute)
	for i := 0; i < 50; i++ {
		rl.Fail("10.0.0." + string(rune('0'+i%10)))
	}
	time.Sleep(60 * time.Millisecond)
	rl.Fail("192.0.2.1") // any Fail sweeps

	rl.mu.Lock()
	n := len(rl.entries)
	rl.mu.Unlock()
	if n != 1 {
		t.Errorf("stale entries left behind: %d (want 1)", n)
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	rl := NewRateLimiter(0, time.Minute, time.Minute)
	for i := 0; i < 100; i++ {
		if rl.Fail("1.2.3.4") {
			t.Fatal("a disabled limiter blocked")
		}
	}
	if rl.Blocked("1.2.3.4") {
		t.Error("a disabled limiter reports blocked")
	}
}

// A nil limiter must behave as "no limiting" rather than panic: it keeps the
// call sites free of nil checks.
func TestRateLimiterNilSafe(t *testing.T) {
	var rl *RateLimiter
	if rl.Blocked("1.2.3.4") || rl.Fail("1.2.3.4") {
		t.Error("a nil limiter reported activity")
	}
	rl.Reset("1.2.3.4")
}
