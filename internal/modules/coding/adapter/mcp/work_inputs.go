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
	RequestID    string         `json:"request_id,omitempty" jsonschema:"optional idempotency key (16-64 chars)"`
	Steps        []PlanWorkStep `json:"steps" jsonschema:"ordered steps of the execution plan; required"`
	BaseRevision int64          `json:"base_revision" jsonschema:"required plan revision for concurrent edit protection"`
}

type markdownInput struct {
	WorkTarget
	Markdown string `json:"markdown" jsonschema:"markdown content to save (at most 12000 chars); required"`
}

type researchInput struct {
	WorkTarget
	Markdown string   `json:"markdown" jsonschema:"research markdown (400+ runes, max 12000 chars); required"`
	Sources  []string `json:"sources,omitempty" jsonschema:"up to 30 absolute http(s) research source URLs"`
}

type progressInput struct {
	WorkTarget
	Message string   `json:"message" jsonschema:"progress update message; required"`
	Claims  []string `json:"claims,omitempty" jsonschema:"factual completion claims; at most 20"`
}

type completeInput struct {
	WorkTarget
	Outcome string   `json:"outcome" jsonschema:"outcome status, e.g. completed or blocked; required"`
	Summary string   `json:"summary" jsonschema:"concise summary of work outcome; required"`
	Claims  []string `json:"claims,omitempty" jsonschema:"factual claims verifying completion"`
}

type workIDInput struct{ WorkTarget }
