package mcp

type submitPlanInput struct {
	WorkTarget
	Summary      string `json:"summary"`
	RequestID    string `json:"request_id,omitempty"`
	BaseRevision int64  `json:"base_revision"`
	SmallTask    bool   `json:"small_task,omitempty"`
}

type askQuestionInput struct {
	WorkTarget
	RequestID string          `json:"request_id" jsonschema:"required"`
	Questions []QuestionInput `json:"questions" jsonschema:"required"`
}

type finishResearchInput struct {
	WorkTarget
	Summary string `json:"summary,omitempty"`
}
