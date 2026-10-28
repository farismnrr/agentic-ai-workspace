package mcp

type submitPlanInput struct {
	WorkTarget
	Summary      string `json:"summary" jsonschema:"summary of the plan submitted for person review; required"`
	RequestID    string `json:"request_id,omitempty" jsonschema:"optional idempotency key (16-64 chars)"`
	BaseRevision int64  `json:"base_revision" jsonschema:"base plan revision to guard against concurrent edits; required"`
	SmallTask    bool   `json:"small_task,omitempty" jsonschema:"set to true only for small tasks where formal research is unnecessary"`
}

type askQuestionInput struct {
	WorkTarget
	RequestID string          `json:"request_id" jsonschema:"required,optional idempotency key"`
	Questions []QuestionInput `json:"questions" jsonschema:"required,1 to 4 questions for the person"`
}

type finishResearchInput struct {
	WorkTarget
	Summary string `json:"summary,omitempty" jsonschema:"optional summary of research completion; checks whether saved research has at least 400 runes and valid sources"`
}
