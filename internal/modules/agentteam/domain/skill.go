package domain

import (
	"embed"
	"io/fs"
	"strings"
)

// skillFiles holds the built-in playbooks: one folder per built-in role, one folder per skill inside it, each
// with a SKILL.md (and the licence of the skill when it comes from elsewhere).
//
//go:embed skills
var skillFiles embed.FS

// Skill is one playbook of a built-in role: what to do and how, written as markdown. Name and Description come
// from the frontmatter of its SKILL.md; Body is the rest.
type Skill struct {
	Name        string
	Description string
	Body        string
}

// SkillsFor lists the playbooks of a built-in role in name order, or none for a role without any. They ship
// with the app, so they are always the current ones, unlike a prompt the person saved earlier.
func SkillsFor(key string) []Skill {
	entries, err := fs.ReadDir(skillFiles, "skills/"+key)
	if err != nil {
		return nil
	}

	skills := []Skill{}

	for _, entry := range entries {
		data, err := skillFiles.ReadFile("skills/" + key + "/" + entry.Name() + "/SKILL.md")
		if err != nil {
			continue
		}

		skill := parseSkill(string(data))
		if skill.Name == "" {
			skill.Name = entry.Name()
		}

		skills = append(skills, skill)
	}

	return skills
}

// parseSkill splits a SKILL.md into the name and description of its frontmatter and the body after it. The
// description may sit on the line or in an indented block that follows it.
func parseSkill(text string) Skill {
	rest, found := strings.CutPrefix(text, "---\n")
	if !found {
		return Skill{Body: strings.TrimSpace(text)}
	}

	front, body, found := strings.Cut(rest, "\n---\n")
	if !found {
		return Skill{Body: strings.TrimSpace(text)}
	}

	skill := Skill{Body: strings.TrimSpace(body)}
	lines := strings.Split(front, "\n")

	for index, line := range lines {
		switch {
		case strings.HasPrefix(line, "name:"):
			skill.Name = scalar(strings.TrimPrefix(line, "name:"))
		case strings.HasPrefix(line, "description:"):
			skill.Description = describe(strings.TrimPrefix(line, "description:"), lines[index+1:])
		}
	}

	return skill
}

func scalar(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

// describe reads a description: the rest of its line, or for a block (| or >) the indented lines after it.
func describe(first string, following []string) string {
	first = strings.TrimSpace(first)
	if first != "|" && first != ">" {
		return scalar(first)
	}

	words := []string{}

	for _, line := range following {
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}

		words = append(words, strings.Fields(line)...)
	}

	return strings.Join(words, " ")
}
