package mcp

type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Preview     string `json:"preview,omitempty"`
}

type QuestionForm struct {
	Header  string           `json:"header,omitempty"`
	Text    string           `json:"text"`
	Type    string           `json:"type"`
	Options []QuestionOption `json:"options,omitempty"`
}

type QuestionInput struct {
	Header   string           `json:"header,omitempty"`
	Question string           `json:"question"`
	Type     string           `json:"type"`
	Options  []QuestionOption `json:"options,omitempty"`
}

type QuestionResult struct {
	QuestionID string `json:"question_id"`
	EventID    string `json:"event_id"`
	State      string `json:"state"`
}

type ResearchResult struct {
	Summary string   `json:"summary,omitempty"`
	Missing []string `json:"missing"`
	Ready   bool     `json:"ready"`
}

type QuestionGroups struct {
	Open     []WorkQuestion `json:"open"`
	Answered []WorkQuestion `json:"answered"`
}

type WorkQuestion struct {
	ID      string         `json:"id"`
	EventID string         `json:"event_id"`
	State   string         `json:"state"`
	Summary string         `json:"summary,omitempty"`
	Form    []QuestionForm `json:"form"`
	Answers [][]string     `json:"answers,omitempty"`
	AskedAt int64          `json:"asked_at"`
}

type PlanReviewInfo struct {
	Summary      string `json:"summary"`
	Status       string `json:"status"`
	Decision     string `json:"decision,omitempty"`
	HandoffID    string `json:"handoff_id,omitempty"`
	HandoffState string `json:"handoff_state,omitempty"`
	LinkedTaskID string `json:"linked_task_id,omitempty"`
	Revision     int64  `json:"revision"`
	SmallTask    bool   `json:"small_task,omitempty"`
}
