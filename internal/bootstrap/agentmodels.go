package bootstrap

import (
	"context"
	"errors"
	"fmt"

	prefapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/application"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	providerapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	providerdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
	anthropicllm "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/anthropic"
	openaillm "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/openai"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

var (
	errProviderOff = errors.New("provider is switched off")
	errNoDefault   = errors.New("no default model is set")
)

// agentModels gives the agent the model clients and default of a person's providers.
type agentModels struct {
	providers *providerapp.Service
	prefs     *prefapp.Service
	ownPort   string
}

// hostedProviders is where Anthropic, OpenAI and Gemini live: the public internet only.
var hostedProviders = egress.Policy{Rule: egress.PublicOnly}

// gateways is where a custom gateway may live: this computer, but never this server itself.
func gateways(ownPort string) egress.Policy {
	return egress.Policy{Rule: egress.LocalOnly, OwnPort: ownPort}
}

func (m agentModels) Client(ctx context.Context, userID, providerID string) (taskapp.Model, error) {
	providers, err := m.providers.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	for _, provider := range providers {
		if provider.ID == providerID && !provider.Enabled {
			return nil, errProviderOff
		}
	}

	credential, err := m.providers.Credential(ctx, userID, providerID)
	if err != nil {
		return nil, fmt.Errorf("load credential: %w", err)
	}

	policy := hostedProviders
	if credential.Kind.CustomAddress() {
		policy = gateways(m.ownPort)
	}

	if credential.Kind == providerdomain.KindAnthropic {
		return anthropicllm.New(credential.BaseURL, credential.APIKey, policy), nil
	}

	// OpenAI, Gemini (through Google's OpenAI-compatible endpoint) and gateways share one wire format.
	return openaillm.New(credential.BaseURL, credential.APIKey, policy), nil
}

func (m agentModels) Default(ctx context.Context, userID string) (string, string, error) {
	prefs, err := m.prefs.Get(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("load preferences: %w", err)
	}

	if prefs.DefaultProviderID == "" || prefs.DefaultModel == "" {
		return "", "", errNoDefault
	}

	return prefs.DefaultProviderID, prefs.DefaultModel, nil
}

// DefaultEffort is the effort the person prefers when a task does not say; empty when none is set.
func (m agentModels) DefaultEffort(ctx context.Context, userID string) taskdomain.Effort {
	prefs, err := m.prefs.Get(ctx, userID)
	if err != nil {
		return taskdomain.EffortDefault
	}

	return taskdomain.Effort(prefs.DefaultEffort)
}

// Tier is the model the person set for the Thinking or the Fast tier; ok is false when they set none.
func (m agentModels) Tier(
	ctx context.Context, userID, tier string,
) (providerID, model string, effort taskdomain.Effort, ok bool) {
	prefs, err := m.prefs.Get(ctx, userID)
	if err != nil {
		return "", "", taskdomain.EffortDefault, false
	}

	chosen := prefs.Thinking
	if tier == taskapp.TierFast {
		chosen = prefs.Fast
	}

	if chosen.Empty() {
		return "", "", taskdomain.EffortDefault, false
	}

	return chosen.ProviderID, chosen.Model, taskdomain.Effort(chosen.Effort), true
}

// agentPersona gives the agent a person's persona and the rules they switched on.
type agentPersona struct {
	prefs *prefapp.Service
}

func (p agentPersona) Persona(ctx context.Context, userID string) (taskapp.Persona, error) {
	prefs, err := p.prefs.Get(ctx, userID)
	if err != nil {
		return taskapp.Persona{}, fmt.Errorf("load preferences: %w", err)
	}

	persona := taskapp.Persona{
		Name: prefs.PersonaName, TalkStyle: prefs.TalkStyle, Language: prefs.Language, About: prefs.About,
	}

	for _, rule := range prefs.Rules {
		if rule.On {
			persona.Rules = append(persona.Rules, rule.Text)
		}
	}

	return persona, nil
}

// agentProjects lets the agent look up the project a task belongs to.
type agentProjects struct {
	projects *projectapp.Service
}

func (p agentProjects) Name(ctx context.Context, userID, projectID string) (string, error) {
	project, err := p.projects.Get(ctx, userID, projectID)
	if err != nil {
		return "", fmt.Errorf("find project: %w", err)
	}

	return project.Name, nil
}

func (p agentProjects) List(ctx context.Context, userID string) ([]taskapp.ProjectSummary, error) {
	projects, err := p.projects.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list projects for Orchestrator: %w", err)
	}
	result := make([]taskapp.ProjectSummary, 0, len(projects))
	for _, project := range projects {
		result = append(result, taskapp.ProjectSummary{ID: project.ID, Name: project.Name})
	}
	return result, nil
}
