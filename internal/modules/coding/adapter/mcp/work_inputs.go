package mcp

import "strings"

type beginWorkInput struct {
	Project        string   `json:"project,omitempty"`
	Summary        string   `json:"summary"`
	Role           string   `json:"role" jsonschema:"enabled non-lead role key to use first; required"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	PlannedRoles   []string `json:"planned_roles,omitempty"`
}

func (in beginWorkInput) wanted() string { return strings.TrimSpace(in.Project) }

type roleWorkInput struct {
	WorkTarget
	Role  string `json:"role"`
	Brief string `json:"brief,omitempty"`
}

type workPlanInput struct {
	WorkTarget
	RequestID    string         `json:"request_id,omitempty"`
	Steps        []PlanWorkStep `json:"steps"`
	BaseRevision int64          `json:"base_revision"`
}

type markdownInput struct {
	WorkTarget
	Markdown string `json:"markdown"`
}

type researchInput struct {
	WorkTarget
	Markdown string   `json:"markdown"`
	Sources  []string `json:"sources,omitempty"`
}

type progressInput struct {
	WorkTarget
	Message string   `json:"message"`
	Claims  []string `json:"claims,omitempty"`
}

type completeInput struct {
	WorkTarget
	Outcome string   `json:"outcome"`
	Summary string   `json:"summary"`
	Claims  []string `json:"claims,omitempty"`
}

type workIDInput struct{ WorkTarget }
