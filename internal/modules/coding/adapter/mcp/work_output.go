package mcp

//nolint:govet // fieldalignment: field order keeps the inferred MCP output schema readable.
type saveOutput struct {
	WorkID   string   `json:"work_id"`
	Revision int64    `json:"revision"`
	State    string   `json:"state"`
	Saved    bool     `json:"saved"`
	Path     string   `json:"path"`
	Warnings []string `json:"warnings"`
}

func savedOutput(result WorkMutation, path string, err error) (saveOutput, report, error) {
	if err != nil {
		return saveOutput{}, report{}, err
	}
	if result.Path == "" {
		result.Path = path
	}
	return saveOutput{
		WorkID: result.WorkID, Revision: result.Revision, State: result.State,
		Saved: true, Path: result.Path, Warnings: result.Warnings,
	}, report{}, nil
}
