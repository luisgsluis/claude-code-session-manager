// Package sessionname validates and normalizes tmux session names.
//
// It is the single source of truth for what a valid session name looks like,
// shared by the host (which enforces it at the only place sessions are created
// or renamed) and the handlers (which mirror it for a clean 400 before the
// agent is even reached).
package sessionname

import (
	"regexp"
	"strings"
)

// Pattern is the whitelist a session name must satisfy. It is the same contract
// as the agent's historical whitelist: 1-32 ASCII letters, digits, '_' or '-'.
var Pattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// Valid reports whether name is an acceptable tmux session name.
func Valid(name string) bool { return Pattern.MatchString(name) }

// Normalize turns raw user input into a valid session name. It trims
// whitespace, transliterates accented Latin letters to ASCII, and replaces
// every remaining rune outside [A-Za-z0-9_-] (spaces, punctuation, symbols,
// newlines, shell metacharacters…) with '-'. Runs of '-' collapse to one and
// surrounding '-' are trimmed, so input that is empty or only symbols yields
// the empty string.
//
// The second result is false when nothing valid remains: the name is empty or
// consisted only of symbols, so the caller must reject it.
func Normalize(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}

	var b strings.Builder
	b.Grow(len(s))
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r == '-' {
			if b.Len() == 0 || lastDash {
				continue
			}
			b.WriteRune('-')
			lastDash = true
			continue
		}
		if repl, ok := translit[r]; ok {
			if b.Len() > 0 {
				b.WriteString(repl)
				lastDash = false
			}
			continue
		}
		// Any other rune (space, punctuation, symbol, control char) becomes a
		// separator. Leading and repeated separators are dropped.
		if b.Len() == 0 || lastDash {
			continue
		}
		b.WriteRune('-')
		lastDash = true
	}

	name := strings.TrimRight(b.String(), "-")
	if len(name) > 32 {
		name = strings.TrimRight(name[:32], "-")
	}
	if name == "" {
		return "", false
	}
	return name, true
}

// translit maps accented Latin letters and a few ligatures to their ASCII
// equivalents. It is deliberately not exhaustive — anything not listed falls
// back to the '-' separator in Normalize, which is still safe.
var translit = map[rune]string{
	'a': "a", 'á': "a", 'à': "a", 'â': "a", 'ä': "a", 'ã': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'A': "A", 'Á': "A", 'À': "A", 'Â': "A", 'Ä': "A", 'Ã': "A", 'Å': "A", 'Ā': "A", 'Ă': "A", 'Ą': "A",
	'c': "c", 'ç': "c", 'ć': "c", 'ĉ': "c", 'č': "c",
	'C': "C", 'Ç': "C", 'Ć': "C", 'Ĉ': "C", 'Č': "C",
	'e': "e", 'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ě': "e", 'ę': "e",
	'E': "E", 'É': "E", 'È': "E", 'Ê': "E", 'Ë': "E", 'Ē': "E", 'Ĕ': "E", 'Ě': "E", 'Ę': "E",
	'i': "i", 'í': "i", 'ì': "i", 'î': "i", 'ï': "i", 'ī': "i", 'ĭ': "i", 'į': "i",
	'I': "I", 'Í': "I", 'Ì': "I", 'Î': "I", 'Ï': "I", 'Ī': "I", 'Ĭ': "I", 'Į': "I",
	'n': "n", 'ñ': "n", 'ń': "n", 'ň': "n", 'ņ': "n",
	'N': "N", 'Ñ': "N", 'Ń': "N", 'Ň': "N", 'Ņ': "N",
	'o': "o", 'ó': "o", 'ò': "o", 'ô': "o", 'ö': "o", 'õ': "o", 'ø': "o", 'ō': "o", 'ŏ': "o", 'ő': "o",
	'O': "O", 'Ó': "O", 'Ò': "O", 'Ô': "O", 'Ö': "O", 'Õ': "O", 'Ø': "O", 'Ō': "O", 'Ŏ': "O", 'Ő': "O",
	'u': "u", 'ú': "u", 'ù': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ŭ': "u", 'ů': "u", 'ű': "u", 'ų': "u",
	'U': "U", 'Ú': "U", 'Ù': "U", 'Û': "U", 'Ü': "U", 'Ū': "U", 'Ŭ': "U", 'Ů': "U", 'Ű': "U", 'Ų': "U",
	'y': "y", 'ý': "y", 'ÿ': "y",
	'Y': "Y", 'Ý': "Y", 'Ÿ': "Y",
	's': "s", 'š': "s", 'ś': "s",
	'S': "S", 'Š': "S", 'Ś': "S",
	'z': "z", 'ž': "z", 'ź': "z", 'ż': "z",
	'Z': "Z", 'Ž': "Z", 'Ź': "Z", 'Ż': "Z",
	'g': "g", 'ğ': "g",
	'G': "G", 'Ğ': "G",
	'd': "d", 'ď': "d", 'đ': "d", 'ð': "d",
	'D': "D", 'Ď': "D", 'Đ': "D", 'Ð': "D",
	'l': "l", 'ł': "l",
	'L': "L", 'Ł': "L",
	'r': "r", 'ř': "r",
	'R': "R", 'Ř': "R",
	't': "t", 'ť': "t", 'þ': "th",
	'T': "T", 'Ť': "T", 'Þ': "TH",
	'æ': "ae", 'œ': "oe", 'ß': "ss",
	'Æ': "AE", 'Œ': "OE",
}
