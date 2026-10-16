package domain

import (
	"embed"
	"strings"
)

// skillFiles holds the built-in playbooks, one folder per built-in role.
//
//go:embed skills/*/SKILL.md
var skillFiles embed.FS

// SkillFor is the playbook of a built-in role: what to do and how, written as markdown. It is empty for a role
// without one. The playbook ships with the app, so it is always the current one, unlike a prompt the person
// saved earlier.
func SkillFor(key string) string {
	data, err := skillFiles.ReadFile("skills/" + key + "/SKILL.md")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(skillBody(string(data)))
}

// skillBody drops the frontmatter (name and description) from a SKILL.md.
func skillBody(text string) string {
	rest, found := strings.CutPrefix(text, "---\n")
	if !found {
		return text
	}

	_, body, found := strings.Cut(rest, "\n---\n")
	if !found {
		return text
	}

	return body
}
