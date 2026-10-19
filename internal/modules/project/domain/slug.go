package domain

import (
	"regexp"
	"strconv"
	"strings"
)

// A project's ID is what the person sees as "Project ID", like Google Cloud's: it is chosen when the project is
// made, never changes, and no two projects share one, so the web app a project runs is opened at /run/<id>/.
// The name is only a label: anyone may reuse it and it can be changed at any time.
// (In code the ID is Project.Slug, because Project.ID is the internal identifier.)

// Limits of a project ID.
const (
	minSlug     = 6
	maxSlug     = 30
	fallbackID  = "project"
	padding     = "-app"
	digitPrefix = "p-"
)

var (
	notAddress = regexp.MustCompile(`[^a-z0-9]+`)
	// validSlug: starts with a letter, then lower-case letters, digits and dashes, and does not end with a dash.
	validSlug = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
)

// ValidSlug reports whether a person's chosen project ID is acceptable: 6 to 30 characters of lower-case
// letters, digits and dashes, starting with a letter and not ending with a dash.
func ValidSlug(slug string) bool {
	return len(slug) >= minSlug && len(slug) <= maxSlug && validSlug.MatchString(slug)
}

// SlugFromName suggests a project ID for a name, the way the console of Google Cloud fills the ID in while
// the name is typed. The suggestion always passes ValidSlug; the service makes it free.
func SlugFromName(name string) string {
	slug := strings.Trim(notAddress.ReplaceAllString(strings.ToLower(name), "-"), "-")

	switch {
	case slug == "":
		slug = fallbackID
	case slug[0] >= '0' && slug[0] <= '9':
		slug = digitPrefix + slug
	}

	if len(slug) > maxSlug {
		slug = strings.Trim(slug[:maxSlug], "-")
	}

	for len(slug) < minSlug {
		slug += padding
	}

	return slug
}

// WithNumber returns slug ending in "-n", cut short when needed so it stays within the length limit.
func WithNumber(slug string, n int) string {
	suffix := "-" + strconv.Itoa(n)
	if len(slug)+len(suffix) > maxSlug {
		slug = strings.Trim(slug[:maxSlug-len(suffix)], "-")
	}

	return slug + suffix
}
