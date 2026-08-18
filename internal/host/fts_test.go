package host

import (
	"os"
	"strings"
	"testing"
)

func TestFTSFold(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"ABC", "abc"},
		{"ÁÉÍÓÚÜÑ", "aeiouun"},
		{"Déjà Vu", "deja vu"},
		{"Configuración", "configuracion"},
		{"日本", "日本"}, // non-latin untouched
		{"¿Qué?", "¿que?"},
	}
	for _, c := range cases {
		if got := ftsFold(c.in); got != c.want {
			t.Errorf("ftsFold(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFTSTokens(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"hola, mundo", []string{"hola", "mundo"}},
		{"loadConversations", []string{"loadConversations"}},
		{"config.yaml", []string{"config", "yaml"}},
		{"search_conv", []string{"search", "conv"}},
		{"kebab-case", []string{"kebab", "case"}},
		{"abc123def", []string{"abc123def"}},
		{"   \t  ", nil},
		{"hola...mundo!", []string{"hola", "mundo"}},
	}
	for _, c := range cases {
		if got := ftsTokens(c.in); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("ftsTokens(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func mustParse(t *testing.T, raw string) *ftsQuery {
	t.Helper()
	q := parseFTSQuery(raw)
	if q == nil {
		t.Fatalf("parseFTSQuery(%q) = nil, want non-nil", raw)
	}
	return q
}

func TestFTSParseQuery(t *testing.T) {
	// A bare term is a word-prefix constraint.
	q := mustParse(t, "config")
	if len(q.terms) != 1 || q.terms[0].phrase || q.terms[0].text != "config" {
		t.Fatalf("bare term: %+v", q.terms)
	}

	// A quoted span is a phrase constraint.
	q = mustParse(t, `"frase exacta"`)
	if len(q.terms) != 1 || !q.terms[0].phrase || q.terms[0].text != "frase exacta" {
		t.Fatalf("phrase: %+v", q.terms)
	}

	// Mixed: punctuation splits bare terms, folds accents, normalizes phrase.
	q = mustParse(t, `Config yaml "Frase, exacta"`)
	if len(q.terms) != 3 {
		t.Fatalf("mixed terms: %+v", q.terms)
	}
	if q.terms[0].text != "config" || q.terms[1].text != "yaml" {
		t.Errorf("bare texts: %+v", q.terms)
	}
	if !q.terms[2].phrase || q.terms[2].text != "frase exacta" {
		t.Errorf("phrase normalized: %+v", q.terms[2])
	}

	// Unclosed quote consumes to the end.
	q = mustParse(t, `"frase`)
	if len(q.terms) != 1 || !q.terms[0].phrase || q.terms[0].text != "frase" {
		t.Fatalf("unclosed quote: %+v", q.terms)
	}

	// Empty queries mean "no filter".
	for _, in := range []string{"", "   ", `""`} {
		if got := parseFTSQuery(in); got != nil {
			t.Errorf("parseFTSQuery(%q) = %+v, want nil", in, got)
		}
	}
}

func TestFTSMatchPrefix(t *testing.T) {
	// The joined side is already folded+joined (what the index stores).
	cases := []struct {
		joined, raw string
		want        bool
	}{
		{"configuracion remota", "config", true},
		{"configuracion", "configuracion", true},
		{"config yaml", "config", true},
		{"configuracion", "onfig", false},
		{"myconfig", "config", false}, // not a word start
		{"config", "my", false},
		{"", "config", false},
	}
	for _, c := range cases {
		if got := matchFTS(c.joined, mustParse(t, c.raw)); got != c.want {
			t.Errorf("matchFTS(%q, %q) = %v, want %v", c.joined, c.raw, got, c.want)
		}
	}
}

func TestFTSMatchPhrase(t *testing.T) {
	cases := []struct {
		text, raw string // text is pre-index raw text (folded+joined by matchFTS test harness)
		want      bool
	}{
		{"hola mundo xyz", `"hola mundo"`, true},
		{"inicio hola mundo", `"hola mundo"`, true},
		{"hola mundo", `"hola mundo"`, true},
		{"mundo hola", `"hola mundo"`, false},  // order matters
		{"a x b", `"a b"`, false},              // not contiguous
		{"hola mundo", `"hola m"`, false},      // "m" is not a whole word
		{"hola", `"hol"`, false},               // phrases match whole words, not prefixes
		{"hola mundo", `"hola"`, true},         // single-word phrase
	}
	for _, c := range cases {
		joined := ftsJoin(c.text) // simulate punctuation folding: "hola, mundo" → "hola mundo"
		if got := matchFTS(joined, mustParse(t, c.raw)); got != c.want {
			t.Errorf("matchFTS(%q, %q) = %v, want %v", joined, c.raw, got, c.want)
		}
	}

	// Punctuation between phrase words is ignored: "hola, mundo" contains the
	// phrase "hola mundo".
	if got := matchFTS(ftsJoin("hola, mundo"), mustParse(t, `"hola mundo"`)); !got {
		t.Error(`"hola, mundo" should match phrase "hola mundo"`)
	}
}

func TestFTSMatchAND(t *testing.T) {
	cases := []struct {
		text, raw string
		want      bool
	}{
		{"configuracion remota", "config remota", true},
		{"configuracion remota", "config remoto", false},
		{"config yaml", "config deepseek", false},
	}
	for _, c := range cases {
		if got := matchFTS(ftsJoin(c.text), mustParse(t, c.raw)); got != c.want {
			t.Errorf("matchFTS(%q, %q) = %v, want %v", c.text, c.raw, got, c.want)
		}
	}
}

func TestFTSTextCacheRefresh(t *testing.T) {
	h, _, _ := newTestHost(t)
	id := "20000000-0000-0000-0000-000000000001"
	path := h.convPath + "/" + id + ".jsonl"
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(`{"type":"user","message":{"content":"`+text+`"}}`+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	write("hola mundo")
	st, _ := os.Stat(path)
	if ok, err := h.ftsText.match(id, path, st.ModTime(), st.Size(), mustParse(t, "mundo")); err != nil || !ok {
		t.Fatalf("first match: ok=%v err=%v", ok, err)
	}

	// Appending rewrites the file (new mtime and size): the next search must see
	// the new content, not the cached one.
	write("adios mundo")
	st, _ = os.Stat(path)
	if ok, err := h.ftsText.match(id, path, st.ModTime(), st.Size(), mustParse(t, "adios")); err != nil || !ok {
		t.Fatalf("match after rewrite: ok=%v err=%v", ok, err)
	}
	if ok, _ := h.ftsText.match(id, path, st.ModTime(), st.Size(), mustParse(t, "hola")); ok {
		t.Fatal("stale cached content still matched after the file changed")
	}

	// prune drops ids that are no longer in the current file set.
	h.ftsText.prune(map[string]struct{}{})
	if n := len(h.ftsText.byID); n != 0 {
		t.Fatalf("prune left %d entries, want 0", n)
	}
}
