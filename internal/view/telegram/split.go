package telegram

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxMessageLen is Telegram's limit for one text message. Telegram counts
// UTF-16 code units after parsing entities; Split measures the raw HTML,
// which is never shorter, so its parts always fit.
const MaxMessageLen = 4096

// Split breaks an HTML-formatted message into parts of at most limit UTF-16
// code units each. It prefers to cut at a line break and never cuts inside a
// tag or an entity like &amp;. Every part is valid HTML on its own: tags still
// open at a cut are closed at the end of that part and reopened at the start
// of the next, so a long <pre> block stays monospaced across messages.
// Parts with no visible text are dropped because Telegram rejects them.
func Split(text string, limit int) []string {
	if limit <= 0 || utf16Len(text) <= limit {
		return []string{text}
	}

	toks := tokenize(text)
	var parts []string
	var open []htmlTag
	for i := 0; i < len(toks); {
		prefix := openingTags(open)
		size := utf16Len(prefix)
		stack := open
		j := i
		cut, cutStack := -1, open
		for j < len(toks) {
			next := toks[j].apply(stack)
			// Always take at least one token so the loop makes progress.
			if j > i && size+toks[j].size+closingLen(next) > limit {
				break
			}
			size += toks[j].size
			stack = next
			j++
			// Remember the last line break past the halfway mark; cutting
			// there reads better than cutting mid-line.
			if toks[j-1].text == "\n" && size >= limit/2 {
				cut, cutStack = j, stack
			}
		}
		if j < len(toks) && cut > i {
			j, stack = cut, cutStack
		}

		if hasVisibleText(toks[i:j]) {
			var sb strings.Builder
			sb.WriteString(prefix)
			for _, t := range toks[i:j] {
				sb.WriteString(t.text)
			}
			sb.WriteString(closingTags(stack))
			parts = append(parts, sb.String())
		}
		open, i = stack, j
	}
	return parts
}

// token is an indivisible piece of the message: one tag, one entity, or one
// character of text.
type token struct {
	text string
	size int // UTF-16 code units
	tag  *htmlTag
	// closing is true for </name> tags.
	closing bool
}

type htmlTag struct {
	name string
	raw  string // the full opening tag, e.g. `<a href="…">`
}

// apply returns the open-tag stack after this token. The input slice is never
// modified, so earlier snapshots of the stack stay valid.
func (t token) apply(stack []htmlTag) []htmlTag {
	if t.tag == nil {
		return stack
	}
	if !t.closing {
		next := make([]htmlTag, len(stack), len(stack)+1)
		copy(next, stack)
		return append(next, *t.tag)
	}
	for k := len(stack) - 1; k >= 0; k-- {
		if stack[k].name == t.tag.name {
			return stack[:k:k]
		}
	}
	return stack
}

func tokenize(s string) []token {
	var toks []token
	for i := 0; i < len(s); {
		switch s[i] {
		case '<':
			if end := strings.IndexByte(s[i:], '>'); end > 0 {
				raw := s[i : i+end+1]
				toks = append(toks, tagToken(raw))
				i += end + 1
				continue
			}
		case '&':
			if end := strings.IndexByte(s[i:], ';'); end > 1 && end <= 10 && isEntityName(s[i+1:i+end]) {
				toks = append(toks, token{text: s[i : i+end+1], size: end + 1})
				i += end + 1
				continue
			}
		}
		r, w := utf8.DecodeRuneInString(s[i:])
		toks = append(toks, token{text: s[i : i+w], size: runeUTF16Len(r)})
		i += w
	}
	return toks
}

func tagToken(raw string) token {
	t := token{text: raw, size: utf16Len(raw)}
	inner := strings.TrimSuffix(strings.TrimPrefix(raw, "<"), ">")
	if strings.HasSuffix(inner, "/") {
		return t // self-closing: doesn't change nesting
	}
	if strings.HasPrefix(inner, "/") {
		t.closing = true
		inner = inner[1:]
	}
	name := inner
	if k := strings.IndexFunc(inner, unicode.IsSpace); k >= 0 {
		name = inner[:k]
	}
	if name == "" {
		return token{text: raw, size: t.size} // "<>" is text, not a tag
	}
	t.tag = &htmlTag{name: strings.ToLower(name), raw: raw}
	return t
}

func isEntityName(s string) bool {
	for _, r := range s {
		if !(r == '#' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

func hasVisibleText(toks []token) bool {
	for _, t := range toks {
		if t.tag == nil && strings.TrimSpace(t.text) != "" {
			return true
		}
	}
	return false
}

func openingTags(stack []htmlTag) string {
	var sb strings.Builder
	for _, t := range stack {
		sb.WriteString(t.raw)
	}
	return sb.String()
}

func closingTags(stack []htmlTag) string {
	var sb strings.Builder
	for k := len(stack) - 1; k >= 0; k-- {
		sb.WriteString("</" + stack[k].name + ">")
	}
	return sb.String()
}

func closingLen(stack []htmlTag) int {
	n := 0
	for _, t := range stack {
		n += len(t.name) + 3 // "</" + name + ">"
	}
	return n
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += runeUTF16Len(r)
	}
	return n
}

// runeUTF16Len is utf16.RuneLen for valid runes; spelled out because
// utf16.RuneLen needs Go 1.23 and go.mod targets 1.21.
func runeUTF16Len(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}
