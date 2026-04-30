package local

import (
	"regexp"
	"strings"

	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

// Entities holds all detected components from user text.
type Entities struct {
	Apps      []AppEntity // detected app mentions (sorted by position)
	Paths     []string    // detected file/directory paths
	URLs      []string    // detected URLs
	Actions   []string    // detected action words
	Modifier  *Modifier   // "of/in/on" link between entities
	Command   string      // extracted command text (after app+action)
	Message   string      // extracted message text (in quotes or after "say/قل")
	Contact   string      // extracted contact name
	Query     string      // search query
	RawText   string      // original text
	LowerText string      // lowercase version
	HasButton bool        // user said "button" / "زر"
}

// AppEntity is a detected application mention.
type AppEntity struct {
	Name     string // resolved macOS name: "Visual Studio Code"
	RawMatch string // what user typed: "vs code"
	Position int    // start position in text
	Length   int    // length of match
}

// Modifier represents a linking word like "of/in/on" between two parts.
type Modifier struct {
	Word   string // "of", "in", "on", "في", etc.
	Before string // text segment before modifier
	After  string // text segment after modifier
}

// ── Action word lists ──────────────────────────────────────────────────

var openActions = map[string]bool{
	"open": true, "launch": true, "start": true,
	"افتح": true, "شغل": true, "شغّل": true, "إفتح": true,
}

var runActions = map[string]bool{
	"run": true, "do": true, "execute": true, "exec": true, "enter": true, "perform": true, "try": true,
	"نفذ": true, "شغل": true, "اكتب": true, "نفّذ": true, "سوي": true, "جرب": true,
}

var browseActions = map[string]bool{
	"browse": true, "browser": true, "brows": true, "list": true, "show": true,
	"view": true, "display": true, "see": true, "check": true, "get": true,
	"تصفح": true, "استعرض": true, "اعرض": true, "عرض": true, "شوف": true, "وريني": true,
}

var searchActions = map[string]bool{
	"search": true, "find": true, "google": true, "look": true,
	"ابحث": true, "بحث": true, "فتش": true, "جد": true, "ابحثلي": true,
}

var sendActions = map[string]bool{
	"send": true, "message": true, "tell": true, "told": true, "write": true, "dm": true,
	"ارسل": true, "أرسل": true, "راسل": true, "ابعث": true, "إبعث": true,
}

var clickActions = map[string]bool{
	"click": true, "tap": true, "press": true, "hit": true, "select": true, "choose": true,
	"انقر": true, "اضغط": true, "اختر": true, "حدد": true, "إضغط": true, "إنقر": true,
}

var typeActions = map[string]bool{
	"type": true, "write": true, "input": true,
	"اكتب": true, "أكتب": true, "أدخل": true, "ادخل": true, "طبع": true,
}

var navigateActions = map[string]bool{
	"go": true, "navigate": true, "visit": true, "browse": true,
	"اذهب": true, "روح": true, "انتقل": true, "إنتقل": true, "إذهب": true, "زور": true,
}

var backActions = map[string]bool{
	"back": true, "previous": true,
	"ارجع": true, "رجوع": true, "خلف": true, "السابق": true, "إرجع": true,
}

var forwardActions = map[string]bool{
	"forward": true, "next": true,
	"الأمام": true, "التالي": true, "التالى": true,
}

var refreshActions = map[string]bool{
	"refresh": true, "reload":  true,
	"تحديث": true, "حدث":     true, "أعد":     true,
}

var closeActions = map[string]bool{
	"close": true, "quit": true,
	"اغلق": true, "أغلق": true, "اقفل": true, "إغلاق": true, "إغلق": true,
}

var gitActions = map[string]bool{
	"push": true, "pull": true, "commit": true, "status": true, "clone": true, "fetch": true,
}

// buttonWords are markers that indicate the user wants to click a button.
var buttonWords = map[string]bool{
	"button": true, "btn": true,
	"زر": true, "الزر": true, "زرّ": true,
}

var modifierWords = map[string]bool{
	"of": true, "in": true, "on": true, "at": true, "from": true, "for": true,
	"في": true, "بـ": true, "من": true, "داخل": true, "على": true,
}

// pathShortcuts maps common words to home-relative paths.
var pathShortcuts = map[string]string{
	"documents": "Documents", "docs": "Documents", "المستندات": "Documents",
	"desktop": "Desktop", "سطح المكتب": "Desktop",
	"downloads": "Downloads", "التنزيلات": "Downloads",
	"home": "", "الرئيسية": "",
}

// urlPattern detects URLs.
var urlPattern = regexp.MustCompile(`(?i)(?:https?://\S+|(?:www\.)\S+|\S+\.(?:com|org|net|io|dev|app|ai|co)\b\S*)`)

// quotedPattern extracts text in quotes.
var quotedPattern = regexp.MustCompile(`["""\x60](.+?)["""\x60]`)

// ── Extract function ───────────────────────────────────────────────────

// Extract analyzes user text and returns all detected entities.
func Extract(text string) *Entities {
	e := &Entities{
		RawText:   text,
		LowerText: strings.ToLower(text),
	}

	e.extractApps()
	e.extractURLs()
	e.extractPaths()
	e.extractActions()
	e.extractModifier()
	e.extractQuoted()
	e.extractContact()
	e.extractCommand()

	return e
}

// ── Extraction methods ─────────────────────────────────────────────────

func (e *Entities) extractApps() {
	lower := e.LowerText

	// Try longest matches first to avoid partial matches
	// e.g. "vs code" before "code", "google chrome" before "chrome"
	type candidate struct {
		key    string
		name   string
		pos    int
		length int
	}

	var found []candidate
	for key, name := range intents.AppNameMap {
		idx := findWholeWord(lower, key)
		if idx >= 0 {
			found = append(found, candidate{key, name, idx, len(key)})
		}
	}

	// Remove overlapping matches (keep longest)
	var apps []AppEntity
	for _, c := range found {
		overlaps := false
		for _, existing := range apps {
			if c.pos >= existing.Position && c.pos < existing.Position+existing.Length {
				overlaps = true
				break
			}
			if existing.Position >= c.pos && existing.Position < c.pos+c.length {
				// New match is longer, replace
				if c.length > existing.Length {
					existing.Name = c.name
					existing.RawMatch = c.key
					existing.Position = c.pos
					existing.Length = c.length
				}
				overlaps = true
				break
			}
		}
		if !overlaps {
			apps = append(apps, AppEntity{
				Name:     c.name,
				RawMatch: c.key,
				Position: c.pos,
				Length:    c.length,
			})
		}
	}

	e.Apps = apps
}

func (e *Entities) extractURLs() {
	matches := urlPattern.FindAllString(e.RawText, -1)
	e.URLs = matches
}

func (e *Entities) extractPaths() {
	lower := e.LowerText

	// Check for explicit paths
	if strings.Contains(lower, "~/") || strings.Contains(lower, "/users/") {
		// Extract path-like segments
		words := strings.Fields(e.RawText)
		for _, w := range words {
			if strings.HasPrefix(w, "~/") || strings.HasPrefix(w, "/") {
				e.Paths = append(e.Paths, w)
			}
		}
	}

	// Check for path shortcuts
	for shortcut, resolved := range pathShortcuts {
		if strings.Contains(lower, shortcut) {
			if resolved == "" {
				e.Paths = append(e.Paths, "~")
			} else {
				e.Paths = append(e.Paths, "~/"+resolved)
			}
			break
		}
	}
}

func (e *Entities) extractActions() {
	words := strings.Fields(e.LowerText)
	for _, w := range words {
		w = strings.TrimSuffix(w, ".")
		w = strings.TrimSuffix(w, "،")
		switch {
		case openActions[w]:
			e.Actions = append(e.Actions, "open")
		case runActions[w]:
			e.Actions = append(e.Actions, "run")
		case browseActions[w]:
			e.Actions = append(e.Actions, "browse")
		case searchActions[w]:
			e.Actions = append(e.Actions, "search")
		case sendActions[w]:
			e.Actions = append(e.Actions, "send")
		case clickActions[w]:
			e.Actions = append(e.Actions, "click")
		case typeActions[w]:
			e.Actions = append(e.Actions, "type")
		case navigateActions[w]:
			e.Actions = append(e.Actions, "navigate")
		case backActions[w]:
			e.Actions = append(e.Actions, "back")
		case forwardActions[w]:
			e.Actions = append(e.Actions, "forward")
		case refreshActions[w]:
			e.Actions = append(e.Actions, "refresh")
		case closeActions[w]:
			e.Actions = append(e.Actions, "close")
		case gitActions[w]:
			e.Actions = append(e.Actions, "git_"+w)
		}
		if buttonWords[w] {
			e.HasButton = true
		}
	}
}

func (e *Entities) extractModifier() {
	words := strings.Fields(e.LowerText)
	for i, w := range words {
		if modifierWords[w] && i > 0 && i < len(words)-1 {
			before := strings.Join(words[:i], " ")
			after := strings.Join(words[i+1:], " ")
			e.Modifier = &Modifier{Word: w, Before: before, After: after}
			return
		}
	}
}

func (e *Entities) extractQuoted() {
	matches := quotedPattern.FindStringSubmatch(e.RawText)
	if len(matches) >= 2 {
		e.Message = matches[1]
	}
}

func (e *Entities) extractContact() {
	lower := e.LowerText

	// Pattern: "send/ارسل" ... "to/ل" CONTACT "say/قل" MESSAGE
	contactPatterns := []string{
		`(?:send|message|ارسل|راسل)\s+(?:to\s+|ل|لـ\s*)(\S+)`,
		`(?:send|message|ارسل|راسل)\s+(\S+)`,
	}

	for _, p := range contactPatterns {
		re := regexp.MustCompile(`(?i)` + p)
		m := re.FindStringSubmatch(lower)
		if len(m) >= 2 {
			contact := m[1]
			// Don't capture action words as contact
			if !modifierWords[contact] && !openActions[contact] && !runActions[contact] {
				e.Contact = contact
				return
			}
		}
	}
}

func (e *Entities) extractCommand() {
	// The command is the remaining text after removing known entities
	// This is a best-effort extraction used when the classifier knows
	// we need a command (e.g. terminal run X)
	// Will be refined by the classifier based on context.
}

// ── Helper queries ─────────────────────────────────────────────────────

// HasApp checks if a specific resolved app name was detected.
func (e *Entities) HasApp(name string) bool {
	for _, a := range e.Apps {
		if a.Name == name {
			return true
		}
	}
	return false
}

// HasAction checks if any action of a given type was detected.
func (e *Entities) HasAction(action string) bool {
	for _, a := range e.Actions {
		if a == action {
			return true
		}
	}
	return false
}

// HasAnyAction checks if any of the given actions was detected.
func (e *Entities) HasAnyAction(actions ...string) bool {
	for _, a := range actions {
		if e.HasAction(a) {
			return true
		}
	}
	return false
}

// HasGitAction returns the git action if detected.
func (e *Entities) HasGitAction() string {
	for _, a := range e.Actions {
		if strings.HasPrefix(a, "git_") {
			return strings.TrimPrefix(a, "git_")
		}
	}
	return ""
}

// PrimaryApp returns the first detected app, or empty.
func (e *Entities) PrimaryApp() string {
	if len(e.Apps) > 0 {
		return e.Apps[0].Name
	}
	return ""
}

// SecondaryApp returns the second detected app (if modifier links two apps).
func (e *Entities) SecondaryApp() string {
	if len(e.Apps) > 1 {
		return e.Apps[1].Name
	}
	return ""
}

// PrimaryPath returns the first detected path.
func (e *Entities) PrimaryPath() string {
	if len(e.Paths) > 0 {
		return e.Paths[0]
	}
	return ""
}

// IsIDE returns true if the app is VS Code, Cursor, or Antigravity.
func IsIDE(app string) bool {
	switch app {
	case "Visual Studio Code", "Cursor", "Antigravity":
		return true
	}
	return false
}

// IsTerminal returns true if the app is Terminal or iTerm.
func IsTerminal(app string) bool {
	return app == "Terminal" || app == "iTerm"
}

// IsMessaging returns true if the app is a messaging app.
func IsMessaging(app string) bool {
	switch app {
	case "WhatsApp", "Slack", "Discord", "Telegram", "Messages":
		return true
	}
	return false
}

// IsBrowserApp returns true if the app is a web browser.
func IsBrowserApp(app string) bool {
	switch app {
	case "Safari", "Google Chrome", "Firefox", "Brave Browser", "Arc":
		return true
	}
	return false
}

// TextAfterApps returns the text remaining after removing all app mentions and action words.
func (e *Entities) TextAfterApps() string {
	result := e.LowerText

	// Remove app mentions (whole word only to avoid corrupting other words)
	for _, app := range e.Apps {
		result = replaceWholeWord(result, app.RawMatch, "")
	}

	// Remove common action/modifier words
	removeWords := []string{
		"open", "launch", "start", "and", "then", "run", "do", "execute", "type", "write",
		"search", "find", "browse", "show", "list", "view", "display", "get", "check", "see",
		"send", "message", "tell", "go", "to", "navigate", "visit",
		"افتح", "شغل", "و", "ثم", "نفذ", "اكتب", "ابحث", "بحث", "تصفح", "اعرض", "شوف",
		"of", "in", "on", "at", "the", "a", "for", "from", "with",
		"في", "بـ", "من", "داخل", "على", "ل", "لـ",
	}
	words := strings.Fields(result)
	var filtered []string
	for _, w := range words {
		keep := true
		for _, r := range removeWords {
			if w == r {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, w)
		}
	}

	return strings.TrimSpace(strings.Join(filtered, " "))
}

// findWholeWord finds a key in text only if it appears as a whole word
// (bounded by space, start, or end of string). Returns -1 if not found.
func findWholeWord(text, word string) int {
	idx := 0
	for {
		pos := strings.Index(text[idx:], word)
		if pos < 0 {
			return -1
		}
		absPos := idx + pos
		end := absPos + len(word)

		// Check left boundary: must be start of string or preceded by space/punctuation
		leftOK := absPos == 0 || text[absPos-1] == ' ' || text[absPos-1] == ',' || text[absPos-1] == '.'

		// Check right boundary: must be end of string or followed by space/punctuation
		rightOK := end >= len(text) || text[end] == ' ' || text[end] == ',' || text[end] == '.'

		if leftOK && rightOK {
			return absPos
		}

		idx = absPos + 1
		if idx >= len(text) {
			return -1
		}
	}
}

// replaceWholeWord replaces a whole-word occurrence in text.
func replaceWholeWord(text, word, replacement string) string {
	idx := findWholeWord(text, word)
	if idx < 0 {
		return text
	}
	return text[:idx] + replacement + text[idx+len(word):]
}
