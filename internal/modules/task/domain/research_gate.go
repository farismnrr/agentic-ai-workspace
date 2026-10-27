package domain

import (
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

const minResearchRunes = 400

// ResearchEnough accepts a saved research round only when it has substance and a web source.
func ResearchEnough(rounds []string) (bool, []string) {
	missing := []string{}
	for index, round := range rounds {
		if strings.Contains(strings.ToLower(round), "placeholder") ||
			strings.Contains(strings.ToLower(round), "lorem ipsum") ||
			utf8.RuneCountInString(strings.TrimSpace(round)) < minResearchRunes || !hasResearchSource(round) {
			missing = append(missing, "research round "+strconv.Itoa(index+1)+
				" needs at least 400 runes and an http(s) source")
			continue
		}
		return true, []string{}
	}
	if len(rounds) == 0 {
		missing = append(missing, "save at least one sourced research note")
	}
	return false, missing
}

func hasResearchSource(round string) bool {
	for _, line := range strings.Split(round, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Source:"))
		parsed, err := url.Parse(line)
		if err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" &&
			parsed.User == nil {
			return true
		}
	}
	return false
}
