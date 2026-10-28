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
	BaseRevision int64          `json:"base_revision" jsonschema:"base plan revision to guard against concurrent edits; required"`
}

type markdownInput struct {
	WorkTarget
	Markdown string `json:"markdown" jsonschema:"markdown content to save (at most 12000 chars); required"`
}

type researchInput struct {
	WorkTarget
	Markdown string   `json:"markdown" jsonschema:"research notes and findings in markdown (at least 400 runes for full tasks, at most 12000 chars); required"`
	Sources  []string `json:"sources,omitempty" jsonschema:"list of absolute http or https source URLs used for research (at most 30)"`
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
