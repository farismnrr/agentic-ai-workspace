package domain

import (
	"path"
	"strings"
)

const maxUploadName = 40

// SafeBase reduces a client filename to a short ASCII base name without its extension.
func SafeBase(given, fallback string) string {
	base := path.Base(strings.ReplaceAll(given, `\`, "/"))
	base = strings.TrimSuffix(base, path.Ext(base))
	var kept strings.Builder
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			kept.WriteRune(r)
		case r == '-' || r == '_' || r == ' ' || r == '.':
			kept.WriteByte('-')
		}
	}
	name := strings.Trim(kept.String(), "-")
	if name == "" {
		name = fallback
	}
	return name[:min(len(name), maxUploadName)]
}
