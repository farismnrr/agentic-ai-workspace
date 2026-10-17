package bootstrap

import (
	"context"
	"fmt"

	teamapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/application"
	teamdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// askUserTool is the loop tool with which a role asks the person a question.
const askUserTool = "ask_user"

// agentTeam gives the agent loop a person's team, with each role's access turned into tool names.
type agentTeam struct {
	team *teamapp.Service
}

func (a agentTeam) Roster(ctx context.Context, userID string) (taskapp.Roster, error) {
	team, err := a.team.Get(ctx, userID)
	if err != nil {
		return taskapp.Roster{}, fmt.Errorf("load agent team: %w", err)
	}

	var roster taskapp.Roster

	for _, role := range team.Roles {
		switch {
		case role.Lead:
			roster.Lead = agentRole(role)
		case role.Enabled:
			roster.Members = append(roster.Members, agentRole(role))
		}
	}

	return roster, nil
}

func (a agentTeam) UseByDefault(ctx context.Context, userID string) bool {
	team, err := a.team.Get(ctx, userID)

	return err == nil && team.UseByDefault
}

func agentRole(role teamdomain.Role) taskapp.AgentRole {
	effort := taskdomain.Effort(role.Effort)
	if !effort.Valid() {
		effort = taskdomain.EffortDefault
	}

	return taskapp.AgentRole{
		Key: role.Key, Name: role.Name, Purpose: role.Purpose, Prompt: role.Prompt, Skills: skillsFor(role),
		Library: libraryFor(role), Lookup: lookupFor(role), WritesPRD: role.WritesPRD, WritesResearch: role.WritesResearch,
		ProviderID: role.ProviderID, Model: role.Model, Effort: effort, Tools: toolsFor(role), Lead: role.Lead,
	}
}

// toolsFor lists the tools a role may use: those of the access groups it has. Starting and stopping the
// web app count as running commands.
func toolsFor(role teamdomain.Role) []string {
	names := []string{}

	for _, tool := range codingdomain.Tools() {
		if role.Can(string(tool.Group)) {
			names = append(names, tool.Name)
		}
	}

	// Looking at the running app needs no more than reading, so a reviewer can do it too.
	names = append(names, toolScreenshot)

	if role.Can(teamdomain.AccessExec) {
		names = append(names, toolStartSite, toolStopSite)
	}

	// What the web returns is a stranger's text, so the orchestrator, which holds the person's trust, never reads it raw.
	if !role.Lead && role.Can(teamdomain.AccessWeb) {
		names = append(names, toolWebSearch, toolFetchPage)
	}

	if !role.Lead && role.Can(teamdomain.AccessImages) {
		names = append(names, toolSearchImages)

		if role.Can(teamdomain.AccessWrite) {
			names = append(names, toolDownloadAsset)
		}
	}

	if role.Can(teamdomain.AccessAsk) {
		names = append(names, askUserTool)
	}

	return names
}

func skillsFor(role teamdomain.Role) []taskapp.Skill {
	skills := make([]taskapp.Skill, 0, len(role.Skills))
	for _, skill := range role.Skills {
		skills = append(skills, taskapp.Skill{
			Name: skill.Name, Description: skill.Description, Body: skill.Body, Craft: skill.Craft, Always: skill.Always,
		})
	}

	return skills
}

func libraryFor(role teamdomain.Role) []taskapp.Reference {
	refs := make([]taskapp.Reference, 0, len(role.Library))
	for _, ref := range role.Library {
		refs = append(refs, taskapp.Reference{Name: ref.Name, Summary: ref.Summary})
	}

	return refs
}

// lookupFor reads library items for a role; only built-in roles have any.
func lookupFor(role teamdomain.Role) func(string) (string, bool) {
	if len(role.Library) == 0 {
		return nil
	}

	return func(name string) (string, bool) { return teamdomain.ReferenceText(role.Key, name) }
}
