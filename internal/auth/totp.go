package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TOTP (RFC 6238) over the standard library. Implemented here rather than
// pulled in as a dependency: the algorithm is HMAC-SHA1 over a time counter,
// and the module deliberately carries only golang.org/x/crypto and yaml.
//
// SHA1 is not a security weakness here: RFC 6238's HMAC construction is what
// every authenticator app (Google Authenticator, Aegis, 1Password) implements,
// and HMAC-SHA1 is unaffected by SHA1's collision attacks.

const (
	totpPeriod = 30 * time.Second // RFC 6238 default step
	totpDigits = 6
	totpSkew   = 1 // accept the neighbouring steps too, for clock drift
	totpIssuer = "CCSM"
)

// b32 is the alphabet authenticator apps expect: uppercase base32, no padding.
var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// errBadSecret is returned for anything that is not decodable base32.
var errBadSecret = errors.New("invalid TOTP secret")

// GenerateTOTPSecret returns a new random 160-bit secret, base32-encoded.
// 160 bits is the size RFC 4226 recommends for HMAC-SHA1 keys.
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return b32.EncodeToString(buf), nil
}

// decodeSecret is tolerant with what a user may paste back: lowercase, spaces
// (authenticator apps display the secret in groups of four) and padding.
func decodeSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.Join(strings.Fields(secret), ""))
	s = strings.TrimRight(s, "=")
	if s == "" {
		return nil, errBadSecret
	}
	key, err := b32.DecodeString(s)
	if err != nil || len(key) == 0 {
		return nil, errBadSecret
	}
	return key, nil
}

// TOTPCode returns the code valid at time t for the given base32 secret.
func TOTPCode(secret string, t time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	return hotp(key, TOTPStep(t)), nil
}

// TOTPStep is the RFC 6238 time counter for t: the number of whole periods
// since the Unix epoch. It doubles as the replay key (see ReplayGuard).
func TOTPStep(t time.Time) int64 {
	return t.Unix() / int64(totpPeriod/time.Second)
}

// hotp is RFC 4226's truncation of HMAC-SHA1(key, counter).
func hotp(key []byte, counter int64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(counter))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	off := sum[len(sum)-1] & 0x0f
	v := uint32(sum[off]&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])

	mod := uint32(1)
	for i := 0; i < totpDigits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", totpDigits, v%mod)
}

// VerifyTOTP checks code against secret around now, accepting ±totpSkew steps
// so a device whose clock drifts by up to one period still works. It returns
// the step the code belongs to, which the caller must feed to a ReplayGuard —
// a code stays valid for its whole window, so without that a captured code
// could be replayed within 30 seconds.
//
// The comparison is constant-time, matching how the agent secret is checked.
func VerifyTOTP(secret, code string, now time.Time) (step int64, ok bool) {
	key, err := decodeSecret(secret)
	if err != nil {
		return 0, false
	}
	got := strings.Join(strings.Fields(code), "")
	if len(got) != totpDigits {
		return 0, false
	}

	cur := TOTPStep(now)
	for d := int64(-totpSkew); d <= totpSkew; d++ {
		want := hotp(key, cur+d)
		if subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1 {
			return cur + d, true
		}
	}
	return 0, false
}

// TOTPURI builds the otpauth:// URI an authenticator app consumes, either
// scanned as a QR or typed in by hand. The label is "CCSM:<account>" so the
// app groups the entry under the issuer.
func TOTPURI(account, secret string) string {
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", totpIssuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(int(totpPeriod/time.Second)))

	u := url.URL{
		Scheme:   "otpauth",
		Host:     "totp",
		Path:     "/" + totpIssuer + ":" + account,
		RawQuery: q.Encode(),
	}
	return u.String()
}

// ReplayGuard remembers the last TOTP step consumed per user, so a code cannot
// be used twice inside its validity window. In memory, like the session store:
// a restart clears it, which at worst re-opens a single 30-second window.
type ReplayGuard struct {
	mu   sync.Mutex
	last map[string]int64
}

// NewReplayGuard creates an empty guard.
func NewReplayGuard() *ReplayGuard {
	return &ReplayGuard{last: make(map[string]int64)}
}

// Use records that user consumed step, and reports whether it was fresh.
// A step at or below the last one accepted is a replay and returns false.
func (g *ReplayGuard) Use(user string, step int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if prev, ok := g.last[user]; ok && step <= prev {
		return false
	}
	g.last[user] = step
	return true
}

// Forget drops a user's replay state (used when their 2FA is disabled).
func (g *ReplayGuard) Forget(user string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.last, user)
}
