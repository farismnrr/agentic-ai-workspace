package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	toolUseSkill = "use_skill"
	// maxSkillSummary keeps one line of the skill list short.
	maxSkillSummary = 300
)

// appendSkills tells a teammate its playbook. A single one is followed as it is; with several, the teammate sees
// what each is for and loads the one that fits its job with use_skill.
func appendSkills(out *strings.Builder, skills []Skill) {
	switch {
	case len(skills) == 1:
		out.WriteString("\nYour playbook. Follow it; where it differs from the text above, the playbook wins.\n\n")
		out.WriteString(skills[0].Body + "\n")
	case len(skills) > 1:
		out.WriteString("\nYou have playbooks (skills). Before you start, call use_skill with the one that fits " +
			"your job, then follow it; where it differs from the text above, the playbook wins. You may load " +
			"another when the job changes.\n")

		for _, skill := range skills {
			fmt.Fprintf(out, "- %s: %s\n", skill.Name, clipTo(skill.Description, maxSkillSummary))
		}
	}
}

// hasSkillList reports whether this runner is a teammate with several playbooks to pick from.
func (r *Runner) hasSkillList() bool {
	return r.role != nil && len(r.role.Skills) > 1
}

// useSkillTool is the loop's use_skill tool, for a teammate with several playbooks.
func useSkillTool() domain.ToolDef {
	return domain.ToolDef{
		Name:        toolUseSkill,
		Description: "Load one of your playbooks (skills) by name and read its instructions.",
		Schema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},` +
			`"required":["name"]}`),
	}
}

// useSkill answers with the body of the playbook the teammate asked for.
func (r *Runner) useSkill(ctx context.Context, task domain.Task, call Call) {
	var args struct {
		Name string `json:"name"`
	}

	_ = json.Unmarshal(call.Input, &args)
	names := make([]string, 0, len(r.role.Skills))

	for _, skill := range r.role.Skills {
		if skill.Name == strings.TrimSpace(args.Name) {
			r.result(ctx, task, call.ID, skill.Body, false)

			return
		}

		names = append(names, skill.Name)
	}

	r.result(ctx, task, call.ID, "There is no skill with that name. Your skills: "+strings.Join(names, ", ")+".", true)
}
