package domain

import (
	"bytes"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxSVGCheckBytes = MaxSVGBytes

var (
	foreignObjectPattern = regexp.MustCompile(`(?i)<\s*foreignobject\b`)
	scriptPattern        = regexp.MustCompile(`(?i)<\s*(?:[\w.-]+\s*:\s*)?script\b`)
	eventPattern         = regexp.MustCompile(`(?i)\bon\w+\s*=`)
	javascriptPattern    = regexp.MustCompile(`(?i)javascript\s*:`)
	hrefPattern          = regexp.MustCompile(
		`(?i)(?:^|[\s<])(?:[\w.-]+\s*:\s*)?href\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`,
	)
)

type contentCheck struct {
	err     error
	kind    string
	ext     string
	pending []byte
	svg     []byte
}

func newContentCheck(kind, ext string) *contentCheck {
	return &contentCheck{kind: kind, ext: ext}
}

func (c *contentCheck) Write(data []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	switch {
	case c.kind == "text":
		c.checkText(data)
	case c.ext == ".svg":
		c.svg = append(c.svg, data...)
		if len(c.svg) > maxSVGCheckBytes {
			c.err = ErrTooLarge
		}
	}
	if c.err != nil {
		return 0, c.err
	}
	return len(data), nil
}

func (c *contentCheck) Err() error {
	if c.err == nil && c.kind == "text" && len(c.pending) != 0 {
		return ErrBinary
	}
	if c.err == nil && c.ext == ".svg" && unsafeSVG(c.svg) {
		return ErrBinary
	}
	return c.err
}

func (c *contentCheck) checkText(data []byte) {
	combined := append([]byte(nil), c.pending...)
	combined = append(combined, data...)
	if bytes.IndexByte(combined, 0) >= 0 {
		c.err = ErrBinary
		return
	}
	for offset := 0; offset < len(combined); {
		if !utf8.FullRune(combined[offset:]) {
			c.pending = append(c.pending[:0], combined[offset:]...)
			return
		}
		_, width := utf8.DecodeRune(combined[offset:])
		if width == 1 && combined[offset] >= utf8.RuneSelf {
			c.err = ErrBinary
			return
		}
		offset += width
	}
	c.pending = c.pending[:0]
}

func unsafeSVG(data []byte) bool {
	decoded := strings.ToLower(html.UnescapeString(string(data)))
	if foreignObjectPattern.MatchString(decoded) || scriptPattern.MatchString(decoded) ||
		eventPattern.MatchString(decoded) || javascriptPattern.MatchString(decoded) {
		return true
	}
	for _, match := range hrefPattern.FindAllStringSubmatch(decoded, -1) {
		if !strings.HasPrefix(strings.TrimSpace(firstNonempty(match[1], match[2], match[3])), "#") {
			return true
		}
	}
	return false
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
