package domain

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	secretAssignment = regexp.MustCompile(`(?i)(token|password|secret|authorization|csrf)([=: ]+)[^\s&;,]+`)
	bearerSecret     = regexp.MustCompile(`(?i)bearer\s+[^\s]+`)
	queryString      = regexp.MustCompile(`\?[^\s]*`)
)

func safeCaptureText(value string) string {
	value = queryString.ReplaceAllString(value, "?[redacted]")
	return secretAssignment.ReplaceAllString(bearerSecret.ReplaceAllString(value, "Bearer [redacted]"), "$1$2[redacted]")
}

func safeCapturePath(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "/"
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	return strings.ReplaceAll(safeCaptureText(path), "?", "")
}

// CaptureReportPath removes query data from a page address before showing it in tool reports.
func CaptureReportPath(value string) string { return safeCapturePath(value) }
