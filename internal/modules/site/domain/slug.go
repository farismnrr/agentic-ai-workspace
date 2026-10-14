package domain

import "strings"

const (
	maxSlug     = 40
	defaultSlug = "site"
)

// Slugify turns a project name into the readable part of a site address. The result is only
// cosmetic: the token alone decides which site is opened.
func Slugify(name string) string {
	var out strings.Builder

	dash := false

	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)

			dash = false
		case !dash && out.Len() > 0:
			out.WriteByte('-')

			dash = true
		}
	}

	slug := strings.Trim(out.String(), "-")
	if len(slug) > maxSlug {
		slug = strings.Trim(slug[:maxSlug], "-")
	}

	if slug == "" {
		return defaultSlug
	}

	return slug
}
