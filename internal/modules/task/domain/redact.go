package domain

import "regexp"

type secretPattern struct {
	pattern     *regexp.Regexp
	replacement string
}

var workSecretPatterns = []secretPattern{
	{regexp.MustCompile(`(?i)(["']?authorization["']?\s*[:=]\s*["']?)(?:(?:bearer|basic)\s+)?` +
		`(?:\\.|[^"'\\\s,;}])+`), "$1[redacted]"},
	{regexp.MustCompile(`(?i)\bbearer\s+[^\s,;]+`), "[redacted]"},
	{regexp.MustCompile(
		`(?i)(["']?(?:[a-z0-9]+[_-])*(?:password|passwd|pwd|token|secret|key)["']?\s*[:=]\s*)` +
			`(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s&;,]+)`,
	), "$1[redacted]"},
	{regexp.MustCompile(`(?i)(\bcurl\b[^;\n]*?\s(?:-u|--user)(?:=|\s+))` +
		`(?:"[^"\n]*:[^"\n]*"|'[^'\n]*:[^'\n]*'|[^\s;&:]+:[^\s;&]+)`), "$1[redacted]"},
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{4,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bsk_live_[A-Za-z0-9_-]{4,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bgh[pus]_[A-Za-z0-9_-]{4,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_-]{4,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bAKIA[A-Z0-9]{8,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bAIza[A-Za-z0-9_-]{8,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bnpm_[A-Za-z0-9_-]{4,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{4,}\b`), "[redacted]"},
	{regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{4,}\b`), "[redacted]"},
	{regexp.MustCompile(`\beyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`), "[redacted]"},
	{regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----` +
		`[\s\S]*?-----END (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`), "[redacted]"},
	{regexp.MustCompile(`://[^/:@\s]+:[^/@\s]+@`), "[redacted]"},
}

// Redact removes common credentials and returns the number of redactions made.
func Redact(text string) (string, int) {
	count := 0
	for _, pattern := range workSecretPatterns {
		count += len(pattern.pattern.FindAllStringIndex(text, -1))
		text = pattern.pattern.ReplaceAllString(text, pattern.replacement)
	}
	return text, count
}
