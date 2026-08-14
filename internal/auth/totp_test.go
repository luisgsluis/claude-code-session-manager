package auth

import (
	"strings"
	"testing"
	"time"
)

// RFC 6238 Appendix B uses the ASCII secret "12345678901234567890" (20 bytes),
// which is this in base32. The published codes are 8 digits; CCSM uses 6, so
// the expectations below are their last six digits.
const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestTOTPCodeRFC6238Vectors(t *testing.T) {
	cases := []struct {
		unix int64
		want string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}
	for _, c := range cases {
		got, err := TOTPCode(rfcSecret, time.Unix(c.unix, 0))
		if err != nil {
			t.Fatalf("t=%d: %v", c.unix, err)
		}
		if got != c.want {
			t.Errorf("t=%d: got %s, want %s", c.unix, got, c.want)
		}
	}
}

func TestVerifyTOTPAcceptsSkew(t *testing.T) {
	now := time.Unix(1234567890, 0)
	for _, delta := range []time.Duration{-30 * time.Second, 0, 30 * time.Second} {
		code, err := TOTPCode(rfcSecret, now.Add(delta))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := VerifyTOTP(rfcSecret, code, now); !ok {
			t.Errorf("code from %v offset rejected", delta)
		}
	}
}

func TestVerifyTOTPRejectsOutsideSkew(t *testing.T) {
	now := time.Unix(1234567890, 0)
	code, err := TOTPCode(rfcSecret, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := VerifyTOTP(rfcSecret, code, now); ok {
		t.Error("a code two minutes out was accepted")
	}
}

func TestVerifyTOTPReturnsStep(t *testing.T) {
	now := time.Unix(1234567890, 0)
	code, _ := TOTPCode(rfcSecret, now)
	step, ok := VerifyTOTP(rfcSecret, code, now)
	if !ok {
		t.Fatal("valid code rejected")
	}
	if step != TOTPStep(now) {
		t.Errorf("got step %d, want %d", step, TOTPStep(now))
	}
}

func TestVerifyTOTPRejectsGarbage(t *testing.T) {
	now := time.Unix(1234567890, 0)
	code, _ := TOTPCode(rfcSecret, now)

	cases := []struct{ name, secret, code string }{
		{"wrong code", rfcSecret, "000000"},
		{"empty code", rfcSecret, ""},
		{"short code", rfcSecret, "12345"},
		{"non-base32 secret", "not base32!!", code},
		{"empty secret", "", code},
	}
	for _, c := range cases {
		if _, ok := VerifyTOTP(c.secret, c.code, now); ok {
			t.Errorf("%s: accepted", c.name)
		}
	}
}

// Authenticator apps display the secret in space-separated groups and users
// paste codes with stray spaces; both must still work.
func TestVerifyTOTPNormalizesInput(t *testing.T) {
	now := time.Unix(1234567890, 0)
	code, _ := TOTPCode(rfcSecret, now)
	if _, ok := VerifyTOTP("gezd gnbv gy3t qojq gezd gnbv gy3t qojq", " "+code+" ", now); !ok {
		t.Error("lowercase spaced secret / padded code rejected")
	}
}

func TestGenerateTOTPSecretIsUsable(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) != 32 { // 20 bytes in unpadded base32
		t.Errorf("got %d chars, want 32", len(secret))
	}
	other, _ := GenerateTOTPSecret()
	if secret == other {
		t.Error("two generated secrets are identical")
	}

	now := time.Now()
	code, err := TOTPCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := VerifyTOTP(secret, code, now); !ok {
		t.Error("a freshly generated secret does not verify its own code")
	}
}

func TestTOTPURI(t *testing.T) {
	uri := TOTPURI("admin", rfcSecret)
	for _, want := range []string{
		"otpauth://totp/CCSM:admin?",
		"secret=" + rfcSecret,
		"issuer=CCSM",
		"digits=6",
		"period=30",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("uri %q is missing %q", uri, want)
		}
	}
}

func TestReplayGuard(t *testing.T) {
	g := NewReplayGuard()

	if !g.Use("admin", 100) {
		t.Fatal("first use rejected")
	}
	if g.Use("admin", 100) {
		t.Error("the same step was accepted twice")
	}
	if g.Use("admin", 99) {
		t.Error("an older step was accepted")
	}
	if !g.Use("admin", 101) {
		t.Error("a newer step was rejected")
	}
	if !g.Use("other", 100) {
		t.Error("a different user was affected by admin's history")
	}

	g.Forget("admin")
	if !g.Use("admin", 100) {
		t.Error("Forget did not clear the user's history")
	}
}
