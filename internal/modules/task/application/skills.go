package application

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	toolUseSkill = "use_skill"
	// maxSkillResult lets a playbook and the rules loaded with it arrive whole.
	maxSkillResult = 60_000
)

// appendSkills tells a teammate its playbooks. The ones marked always are followed as they are; the others are
// listed by what they are for, and the teammate loads the one that fits its job with use_skill.
func appendSkills(out *strings.Builder, role AgentRole) {
	listed := []Skill{}

	for _, skill := range role.Skills {
		if !skill.Always {
			listed = append(listed, skill)

			continue
		}

		out.WriteString("\nYour playbook. Follow it; where it differs from the text above, the playbook wins.\n\n")
		out.WriteString(skill.Body + "\n")
	}

	if len(listed) > 0 {
		out.WriteString("\nYou have more playbooks (skills). Call use_skill with the one that fits the job, then " +
			"follow it; where it differs from the text above, the playbook wins. You may load another when the " +
			"job changes. Load only those the job needs.\n")

		for _, skill := range listed {
			out.WriteString("- " + skill.Name + ": " + clipTo(skill.Description, maxSummary) + "\n")
		}
	}

	appendLibrary(out, role.Library)
}

// hasSkillList reports whether this runner is a teammate with playbooks to pick from.
func (r *Runner) hasSkillList() bool {
	if r.role == nil {
		return false
	}

	for _, skill := range r.role.Skills {
		if !skill.Always {
			return true
		}
	}

	return false
}

// useSkillTool is the loop's use_skill tool, for a teammate with playbooks to pick from.
func useSkillTool() domain.ToolDef {
	return domain.ToolDef{
		Name:        toolUseSkill,
		Description: "Load one of your playbooks (skills) by name and read its instructions.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	}
}

// useSkill answers with the playbook the teammate asked for, followed by the library rules it needs.
func (r *Runner) useSkill(ctx context.Context, task domain.Task, call Call) {
	var args struct {
		Name string `json:"name"`
	}

	_ = json.Unmarshal(call.Input, &args)
	names := make([]string, 0, len(r.role.Skills))

	for _, skill := range r.role.Skills {
		if skill.Name == strings.TrimSpace(args.Name) {
			r.resultUpTo(ctx, task, call.ID, r.withCraft(skill), false, maxSkillResult)

			return
		}

		names = append(names, skill.Name)
	}

	r.result(ctx, task, call.ID, "There is no skill with that name. Your skills: "+strings.Join(names, ", ")+".", true)
}

// withCraft is a skill's body followed by the library items it asks for.
func (r *Runner) withCraft(skill Skill) string {
	text := skill.Body

	for _, name := range skill.Craft {
		if rules, ok := r.lookup("craft/" + name); ok {
			text += "\n\n---\n\n## Rules: " + name + "\n\n" + rules
		}
	}

	return text
}
