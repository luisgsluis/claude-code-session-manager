package host

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Full-text search over conversations. Word-based, not substring: a bare term
// matches any word it prefixes; a quoted phrase must appear as a complete,
// contiguous word sequence. Case- and accent-insensitive (Spanish folding), so
// "config" finds "Configuración". conversationsList uses it for both the title
// field (q) and the full-conversation field (q_text).

// accentMap folds Latin diacritics (Spanish + common French/German/Scandinavian)
// to their base letters, so search is accent-insensitive: "deja" finds "déjà".
var accentMap = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ñ': 'n', 'ç': 'c',
}

var accentReplacer = func() *strings.Replacer {
	pairs := make([]string, 0, len(accentMap)*2)
	for from, to := range accentMap {
		pairs = append(pairs, string(from), string(to))
	}
	return strings.NewReplacer(pairs...)
}()

// ftsFold normalizes text for indexing and querying: lowercase + accent fold.
func ftsFold(s string) string {
	return accentReplacer.Replace(strings.ToLower(s))
}

// ftsTokens splits a string into word tokens on any non-letter/digit. A Go
// identifier like loadConversations stays one word; punctuation, underscores
// and slashes separate words (config.yaml → config, yaml). Tokens are not
// further split by case, so searching "conversations" never matches
// "loadConversations" — only its prefix "load" would.
func ftsTokens(s string) []string {
	var toks []string
	start := -1
	for i, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			toks = append(toks, s[start:i])
			start = -1
		}
	}
	if start >= 0 {
		toks = append(toks, s[start:])
	}
	return toks
}

// ftsJoin folds, tokenizes and joins a string with single spaces — the form
// stored per conversation and matched against. Collapsing every separator to a
// single space is what lets a phrase like "hola mundo" match "hola, mundo"
// while still requiring its words to be contiguous.
func ftsJoin(s string) string {
	return strings.Join(ftsTokens(ftsFold(s)), " ")
}

// ftsTerm is one constraint of a query: a quoted phrase (must appear as a
// contiguous, complete word sequence) or a bare term (word prefix).
type ftsTerm struct {
	phrase bool
	text   string // folded: one word, or a phrase's words joined with spaces
}

// ftsQuery is a parsed query. Every term must match (AND).
type ftsQuery struct {
	terms []ftsTerm
}

// parseFTSQuery folds and parses a search string. Double-quoted spans become
// phrases (an unclosed quote consumes to the end); everything else is split
// into word-prefix terms on punctuation. Empty terms and "" are dropped; a
// query with no terms returns nil, meaning "no filter".
func parseFTSQuery(raw string) *ftsQuery {
	runes := []rune(ftsFold(raw))
	q := &ftsQuery{}
	n := len(runes)
	i := 0
	for i < n {
		switch c := runes[i]; {
		case c == '"':
			j := i + 1
			phrase := make([]rune, 0, 8)
			for j < n && runes[j] != '"' {
				phrase = append(phrase, runes[j])
				j++
			}
			i = j + 1 // skip the closing quote; may overshoot, ending the loop
			if p := ftsJoin(string(phrase)); p != "" {
				q.terms = append(q.terms, ftsTerm{phrase: true, text: p})
			}
		case unicode.IsLetter(c) || unicode.IsDigit(c):
			j := i
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j])) {
				j++
			}
			q.terms = append(q.terms, ftsTerm{text: string(runes[i:j])})
			i = j
		default:
			i++
		}
	}
	if len(q.terms) == 0 {
		return nil
	}
	return q
}

// matchFTS reports whether a conversation's folded, space-joined text satisfies
// every term. The leading space pins a bare term to a word start (so "config"
// matches "configuración" but not "myconfig"), and the trailing space pins a
// phrase's ends (so "hola m" never matches "hola mundo").
func matchFTS(joined string, q *ftsQuery) bool {
	padded := " " + joined + " "
	for _, t := range q.terms {
		if t.phrase {
			if !strings.Contains(padded, " "+t.text+" ") {
				return false
			}
		} else if !strings.Contains(padded, " "+t.text) {
			return false
		}
	}
	return true
}

// ftsTextEntry is the indexed form of one conversation's chat text.
type ftsTextEntry struct {
	joined string // folded tokens joined with single spaces
	mod    time.Time
	size   int64
}

// ftsTextCache lazily holds the folded chat text of conversations, so a
// full-text query only re-reads a file when it changed (live sessions append
// constantly). It stays empty — the agent keeps its lean footprint — until the
// first full-conversation search, and the actual chat text is far smaller than
// the raw jsonl (tool results, meta and image payloads are not chat).
type ftsTextCache struct {
	mu   sync.Mutex
	byID map[string]ftsTextEntry
}

// conversationFullTextJoined reads a conversation and folds its chat text
// (user + assistant turns, the same view chatRoleAndText gives the chat UI)
// into the space-joined token form search matches against. Tool results, meta
// events and slash commands are not part of the conversation, so they are not
// indexed — and never pollute searches with whatever files a tool happened to
// read.
func conversationFullTextJoined(path string) (string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fh.Close()
	var parts []string
	forEachLine(fh, func(raw []byte) bool {
		var line convLine
		if err := json.Unmarshal(raw, &line); err != nil {
			return true
		}
		_, text, _, ok := chatRoleAndText(line)
		if ok {
			parts = append(parts, text)
		}
		return true
	})
	return ftsJoin(strings.Join(parts, "\n")), nil
}

// match re-indexes the conversation when its file changed, then reports whether
// its chat text satisfies q. Thread-safe: the cache lock serializes the read
// and the map update (fine at this scale — a rebuild is a handful of MB at
// most, and only when the file actually changed).
func (c *ftsTextCache) match(id, path string, mod time.Time, size int64, q *ftsQuery) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byID == nil {
		c.byID = make(map[string]ftsTextEntry)
	}
	if e, ok := c.byID[id]; ok && e.mod.Equal(mod) && e.size == size {
		return matchFTS(e.joined, q), nil
	}
	joined, err := conversationFullTextJoined(path)
	if err != nil {
		return false, err
	}
	c.byID[id] = ftsTextEntry{joined: joined, mod: mod, size: size}
	return matchFTS(joined, q), nil
}

// prune drops entries whose conversation is no longer among the current files,
// so the cache never hoards indexed text for deleted transcripts.
func (c *ftsTextCache) prune(keep map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.byID {
		if _, ok := keep[id]; !ok {
			delete(c.byID, id)
		}
	}
}
