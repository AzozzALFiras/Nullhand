package audit

import "regexp"

// Redacted replaces secret values in audit lines.
const Redacted = "[REDACTED]"

// secretPatterns match credentials that commonly end up in logged commands
// and messages: `/shell curl -H "Authorization: Bearer …"`, `export
// GITHUB_TOKEN=…`, a pasted API key. Where a pattern has a label group (the
// key name, "Bearer ", the URL user) the label is kept so the log still says
// what was there.
var secretPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	// Telegram bot tokens: 123456789:AA… (35 chars after the colon).
	{regexp.MustCompile(`\b\d{6,12}:[A-Za-z0-9_-]{35,}`), Redacted},
	// Provider keys with distinctive prefixes. Minimum lengths are kept low
	// so a key cut short by log truncation is still caught.
	{regexp.MustCompile(`\bsk-(?:ant-|proj-)?[A-Za-z0-9_-]{16,}`), Redacted},
	{regexp.MustCompile(`\b(?:gh[pousr]_|github_pat_)[A-Za-z0-9_]{8,}`), Redacted},
	{regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{8,}`), Redacted},
	{regexp.MustCompile(`\bxai-[A-Za-z0-9]{8,}`), Redacted},
	{regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{8,}`), Redacted},
	{regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`), Redacted},
	// Authorization headers. "Bearer" is distinctive enough on its own;
	// "Basic"/"Token" only count after "Authorization:" so ordinary phrases
	// like "basic settings" are left alone.
	{regexp.MustCompile(`(?i)\b(bearer\s+)[A-Za-z0-9._~+/=-]{8,}`), "${1}" + Redacted},
	{regexp.MustCompile(`(?i)\b(authorization:\s*(?:basic|token)\s+)[A-Za-z0-9._~+/=-]{8,}`), "${1}" + Redacted},
	// Credentials embedded in URLs: scheme://user:password@host.
	{regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://[^\s:/@"]+:)[^\s@/"]+@`), "${1}" + Redacted + "@"},
	// key=value / key: value where the key name says it is a secret. The
	// optional (escaped) quotes cover JSON like {"api_key":"…"}, which the
	// audit line's %q formatting turns into {\"api_key\":\"…\"}.
	{regexp.MustCompile(`(?i)\b([a-z0-9_.-]*(?:password|passwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key)[a-z0-9_.-]*(?:\\?")?\s*[=:]\s*(?:\\?")?)[^\s"'\\&,;]+`), "${1}" + Redacted},
	// --password value / --token value style flags.
	{regexp.MustCompile(`(?i)(--?(?:password|passwd|pass|token|secret|api-?key)\s+)[^\s"'\\-][^\s"'\\]*`), "${1}" + Redacted},
}

// Redact replaces anything that looks like a credential in s with Redacted.
// It errs toward over-redaction: a lost token fragment in the audit log is
// harmless, a leaked token is not.
func Redact(s string) string {
	for _, p := range secretPatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}
	return s
}
