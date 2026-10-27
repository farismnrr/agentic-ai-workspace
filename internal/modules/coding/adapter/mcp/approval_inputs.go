package mcp

type approvePlanInput struct {
	WorkTarget
	IdempotencyKey string `json:"idempotency_key" jsonschema:"unique key reused to safely retry this decision; required"`
	Revision       int64  `json:"revision" jsonschema:"exact submitted plan revision to approve; required"`
}
