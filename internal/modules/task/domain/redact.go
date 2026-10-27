package domain

import "regexp"

var workSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)authorization\s*[:=]\s*(?:(?:bearer|basic)\s+)?[^\s,;]+`),
	regexp.MustCompile(`(?i)\bbearer\s+[^\s,;]+`),
	regexp.MustCompile(
		`(?i)["']?(?:[a-z0-9]+[_-])*(password|token|secret|key)["']?\s*[:=]\s*` +
			`(?:"[^"]*"|'[^']*'|[^\s&;,]+)`,
	),
	regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{4,}\b`),
	regexp.MustCompile(`\bsk_live_[A-Za-z0-9_-]{4,}\b`),
	regexp.MustCompile(`\bgh[pus]_[A-Za-z0-9_-]{4,}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_-]{4,}\b`),
	regexp.MustCompile(`\bAKIA[A-Z0-9]{8,}\b`),
	regexp.MustCompile(`\bAIza[A-Za-z0-9_-]{8,}\b`),
	regexp.MustCompile(`\bnpm_[A-Za-z0-9_-]{4,}\b`),
	regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{4,}\b`),
	regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{4,}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
	regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----` +
		`[\s\S]*?-----END (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`),
	regexp.MustCompile(`://[^/:@\s]+:[^/@\s]+@`),
}

// Redact removes common credentials and returns the number of redactions made.
func Redact(text string) (string, int) {
	count := 0
	for _, pattern := range workSecretPatterns {
		count += len(pattern.FindAllStringIndex(text, -1))
		text = pattern.ReplaceAllString(text, "[redacted]")
	}
	return text, count
}
