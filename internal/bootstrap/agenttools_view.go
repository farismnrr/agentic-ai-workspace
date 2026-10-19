package bootstrap

import (
	"encoding/json"
	"errors"
	"net/http"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var toolViewImage = codingdomain.ToolViewImage.Name

func viewImageDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name: toolViewImage, Description: codingdomain.ToolViewImage.Description,
		Schema: json.RawMessage(`{"type":"object","properties":{` +
			`"path":{"type":"string","description":"The picture, such as documents/preview/page-1.png"}},` +
			`"required":["path"]}`),
	}
}

func describeView(in toolArgs) taskapp.Described {
	return said("Looked at a picture", "", "view_image "+in.Path)
}

// executeView shows the model a PNG of the project. The tool result carries PNG only, so a JPEG is refused with
// the command that turns it into one.
func (a *agentTools) executeView(projectID string, call taskapp.Call) taskapp.ToolResult {
	var in toolArgs
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return problem("The tool arguments were not valid JSON.")
	}

	services, err := a.services(projectID)
	if err != nil {
		return problem("The project folder is not available.")
	}

	file, err := services.Reader.ReadRaw(in.Path, codingdomain.MaxViewBytes)
	if err != nil {
		if errors.Is(err, codingdomain.ErrTooLarge) {
			return problem("The picture is larger than 4 MB. Make it smaller, for example with a lower -r in pdftoppm.")
		}

		return failure(err)
	}

	if http.DetectContentType(file.Data) != "image/png" {
		return problem("Only PNG pictures can be shown. Turn it into one with: convert in.jpg out.png")
	}

	return taskapp.ToolResult{Output: "Picture " + file.Path + ".", Image: file.Data}
}
