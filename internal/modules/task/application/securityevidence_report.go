package application

import (
	"strings"
)

func reportSection(report, title string) string {
	var section strings.Builder
	reading := false
	for _, raw := range strings.Split(report, "\n") {
		line := strings.TrimSpace(raw)
		heading, content, hasColon := strings.Cut(line, ":")
		if hasColon && strings.EqualFold(strings.TrimSpace(heading), title) {
			reading = true
			section.WriteString(strings.ToLower(strings.TrimSpace(content)))
			continue
		}
		if reading && hasColon && !strings.HasPrefix(line, "-") {
			break
		}
		if reading {
			section.WriteString(" ")
			section.WriteString(strings.ToLower(line))
		}
	}
	return strings.TrimSpace(section.String())
}

func declaredChecks(report string) []string {
	checks := []string{}
	reading := false
	for _, raw := range strings.Split(report, "\n") {
		line := strings.TrimSpace(raw)
		if strings.EqualFold(line, "Checks:") {
			reading = true
			continue
		}
		if !reading {
			continue
		}
		if line == "" {
			if len(checks) > 0 {
				break
			}
			continue
		}
		check, valid := checkLine(line)
		if !valid {
			break
		}
		checks = append(checks, check)
	}
	return checks
}

func checkLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "-") {
		return "", false
	}
	command := strings.TrimSpace(strings.TrimPrefix(line, "-"))
	return command, command != ""
}

func uniqueChecks(checks []string) []string {
	seen := make(map[string]struct{}, len(checks))
	unique := make([]string, 0, len(checks))
	for _, check := range checks {
		if _, found := seen[check]; found {
			continue
		}
		seen[check] = struct{}{}
		unique = append(unique, check)
	}
	return unique
}
