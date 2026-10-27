package application

import (
	"strings"
	"unicode"
)

func stripControls(value string) string {
	return strings.Map(func(char rune) rune {
		if unicode.IsControl(char) {
			return -1
		}
		return char
	}, value)
}
