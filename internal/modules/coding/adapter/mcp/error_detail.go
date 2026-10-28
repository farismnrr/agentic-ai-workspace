package mcp

import "strings"

func cleanErrorDetail(err, known error) string {
	msg := err.Error()
	knownMsg := known.Error()
	if !strings.HasSuffix(msg, knownMsg) {
		return ""
	}
	prefix := strings.TrimSuffix(msg, knownMsg)
	prefix = strings.TrimSuffix(strings.TrimSpace(prefix), ":")
	if idx := strings.LastIndex(prefix, ": "); idx >= 0 {
		prefix = prefix[idx+2:]
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || strings.ContainsAny(prefix, "\r\n") || len(prefix) > 120 {
		return ""
	}
	return prefix
}
