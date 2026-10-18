package domain

import "strings"

// UsableLicense reports whether a picture under this licence may go on a person's own page, even a commercial one:
// public domain, CC0, or Creative Commons that asks only for credit (and, for share-alike, the same terms on
// the picture itself). NonCommercial and NoDerivatives licences are not usable, since the picture will be
// edited and the page may earn money. The text is how the sources write it ("CC BY-SA 4.0", "by-sa", "Public domain").
func UsableLicense(name string) bool {
	text := strings.ToLower(strings.TrimSpace(name))
	if text == "" {
		return false
	}

	for _, banned := range []string{"nc", "nd", "noncommercial", "noderiv", "all rights", "fair use", "unknown"} {
		if hasWord(text, banned) {
			return false
		}
	}

	for _, allowed := range []string{"public domain", "pdm", "cc0", "cc by", "cc-by", "by", "by-sa", "pd "} {
		if text == allowed || strings.HasPrefix(text, allowed+" ") || strings.HasPrefix(text, allowed+"-") {
			return true
		}
	}

	return false
}

// FreeToUse reports whether a picture may be used with nothing to show for it: public domain or CC0. Pictures that
// ask for the author to be named (CC BY and the like) are not offered at all, since the page must stay clean.
func FreeToUse(name string) bool {
	text := strings.ToLower(strings.TrimSpace(name))
	free := strings.HasPrefix(text, "cc0") || strings.HasPrefix(text, "pdm") ||
		strings.HasPrefix(text, "public domain") || strings.HasPrefix(text, "pd ")

	return free && UsableLicense(name)
}

// hasWord reports whether the licence name contains the word as a separate part: "by-nc" has "nc", "bunch" does not.
func hasWord(text, word string) bool {
	for _, part := range strings.FieldsFunc(text, isSeparator) {
		if part == word {
			return true
		}
	}

	return false
}

func isSeparator(r rune) bool { return r == ' ' || r == '-' || r == '_' || r == '/' }
