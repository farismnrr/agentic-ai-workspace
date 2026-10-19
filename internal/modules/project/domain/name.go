package domain

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A project's name is unique across the server because the web app it runs is opened at /run/<name>/.
// Two names are the same when they make the same address, so "Bakery Site" and "bakery-site" collide.

// Named is a project's id and name, enough to tell whether a name is taken.
type Named struct {
	ID   string
	Name string
}

var notAddress = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a name into the address it makes: lower-case letters and digits joined by single dashes.
// It follows the same rule as the site address (site/domain.Slugify), kept here so a project does not
// import another feature.
func Slug(name string) string {
	return strings.Trim(notAddress.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// Taken reports whether another project already makes the same address as name.
func Taken(name, exceptID string, existing []Named) bool {
	slug := Slug(name)

	for _, other := range existing {
		if other.ID != exceptID && Slug(other.Name) == slug {
			return true
		}
	}

	return false
}

// Suggest returns name with the first number that is free ("Bakery 2"), or name itself when it is free.
func Suggest(name, exceptID string, existing []Named) string {
	if !Taken(name, exceptID, existing) {
		return name
	}

	for n := 2; n <= maxSuggestions; n++ {
		candidate := name + " " + strconv.Itoa(n)
		if utf8.RuneCountInString(candidate) <= maxName && !Taken(candidate, exceptID, existing) {
			return candidate
		}
	}

	return name
}

const maxSuggestions = 1000
