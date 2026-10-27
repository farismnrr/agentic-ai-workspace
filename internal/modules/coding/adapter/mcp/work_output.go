package mcp

type saveOutput struct {
	WorkID   string   `json:"work_id"`
	State    string   `json:"state"`
	Path     string   `json:"path"`
	Warnings []string `json:"warnings"`
	Revision int64    `json:"revision"`
	Saved    bool     `json:"saved"`
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
