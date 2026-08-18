package sessionname

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		// Already valid names pass through unchanged.
		{"sesion", "sesion", true},
		{"Mi-Sesion_2", "Mi-Sesion_2", true},
		{"123", "123", true},
		// Spaces become hyphens.
		{"mi sesión", "mi-sesion", true},
		{"  mi   sesión  ", "mi-sesion", true},
		{"mi sesion", "mi-sesion", true},
		// Accents and ñ transliterate to ASCII.
		{"Café Ñoño!", "Cafe-Nono", true},
		{"niño", "nino", true},
		{"crème brûlée", "creme-brulee", true},
		// Special characters and symbols collapse to a single separator.
		{"hola!!!mundo", "hola-mundo", true},
		{"a; rm -rf /", "a-rm-rf", true},
		{"$(id)", "id", true},
		{"../etc/passwd", "etc-passwd", true},
		{"x\nrm -rf /", "x-rm-rf", true},
		// Empty or only-symbols input yields false.
		{"", "", false},
		{"   ", "", false},
		{"...", "", false},
		{"!!!", "", false},
		{"$()", "", false},
		{"\n\t", "", false},
		// Leading/trailing separators are trimmed.
		{"-hola", "hola", true},
		{"hola-", "hola", true},
		{"--hola--mundo--", "hola-mundo", true},
	}

	for _, c := range cases {
		got, ok := Normalize(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("Normalize(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
		if ok && !Valid(got) {
			t.Errorf("Normalize(%q) returned %q which is not Valid()", c.in, got)
		}
	}
}

func TestNormalizeTruncatesTo32(t *testing.T) {
	in := "esto es un nombre de sesión demasiado largo para tmux"
	got, ok := Normalize(in)
	if !ok {
		t.Fatalf("Normalize(%q) unexpectedly rejected", in)
	}
	if len(got) > 32 {
		t.Errorf("name %q exceeds 32 chars (%d)", got, len(got))
	}
	if got[len(got)-1] == '-' {
		t.Errorf("name %q ends with a separator after truncation", got)
	}
	if !Valid(got) {
		t.Errorf("name %q is not Valid()", got)
	}
}

func TestValid(t *testing.T) {
	valid := []string{"a", "abc123", "mi-sesion", "Mi_Sesion", "12345678901234567890123456789012"}
	for _, s := range valid {
		if !Valid(s) {
			t.Errorf("Valid(%q) = false, want true", s)
		}
	}
	invalid := []string{"", "mi sesion", "mi/sesion", "mi:sesion", "sesión", "a.b", "123456789012345678901234567890123"}
	for _, s := range invalid {
		if Valid(s) {
			t.Errorf("Valid(%q) = true, want false", s)
		}
	}
}
