package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	extensiondomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var toolDeployExtension = codingdomain.ToolDeployExtension.Name

// extensionPage is where the person installs a project's extension.
func extensionPage(projectID string) string { return "/app/extensions/" + projectID }

func extensionDefinition() taskdomain.ToolDef {
	return taskdomain.ToolDef{
		Name: toolDeployExtension, Description: codingdomain.ToolDeployExtension.Description,
		Schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":` +
			`"The folder that holds manifest.json after the build, relative to the project, such as .output/chrome-mv3"}},` +
			`"required":["path"]}`),
	}
}

func describeExtension(in toolArgs) taskapp.Described {
	return said(
		"Packaged your browser extension", "May I package your browser extension so you can install it?",
		"deploy_extension "+in.Path)
}

// executeExtension packages the extension built in a folder of the project.
func (a *agentTools) executeExtension(ctx context.Context, projectID string, call taskapp.Call) taskapp.ToolResult {
	if a.extensions == nil {
		return problem("Browser extensions are not available right now.")
	}

	var in toolArgs
	if err := json.Unmarshal(call.Input, &in); err != nil || strings.TrimSpace(in.Path) == "" {
		return problem("deploy_extension needs the path of the folder that holds manifest.json.")
	}

	result, err := a.extensions.Deploy(ctx, projectID, in.Path)

	switch {
	case errors.Is(err, extensiondomain.ErrNoManifest):
		return problem("There is no manifest.json in " + in.Path + ". Build the extension first and give the folder " +
			"the build wrote (for example .output/chrome-mv3 or dist), not the source folder.")
	case errors.Is(err, extensiondomain.ErrBadFolder):
		return problem("That folder is not inside the project. Give a folder of the project, such as .output/chrome-mv3.")
	case errors.Is(err, extensiondomain.ErrTooLarge):
		return problem("The folder is too large for an extension (over 3000 files or 50 MB). Package only the build.")
	case err != nil:
		return problem("The extension could not be packaged.")
	case len(result.Problems) > 0:
		return problem("The extension cannot be installed yet. Fix these, build again and deploy again:\n- " +
			strings.Join(result.Problems, "\n- "))
	}

	extension := result.Extension
	text := fmt.Sprintf("%s %s is packaged (%d files). The person installs it from Extensions in the sidebar: it "+
		"downloads the package and shows the steps. Say so in plain words; do not paste an address.",
		extension.Name, extension.Version, extension.Files)

	if len(result.Notes) > 0 {
		text += "\nNote: " + strings.Join(result.Notes, "; ") + "."
	}

	return taskapp.ToolResult{Output: text, ExtensionLink: extensionPage(projectID)}
}
