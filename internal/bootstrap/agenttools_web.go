package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	webdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

// executeWeb runs a web tool for a project.
func (a *agentTools) executeWeb(ctx context.Context, projectID string, call taskapp.Call) taskapp.ToolResult {
	if a.web == nil {
		return problem("Looking things up on the web is not available right now.")
	}

	var in toolArgs
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return problem("The tool arguments were not valid JSON.")
	}

	switch call.Name {
	case toolWebSearch:
		return a.search(ctx, in)
	case toolFetchPage:
		return a.readPage(ctx, in)
	case toolSearchImages:
		return a.findImages(ctx, projectID, in)
	}

	return a.saveAsset(ctx, projectID, in)
}

func (a *agentTools) search(ctx context.Context, in toolArgs) taskapp.ToolResult {
	results, err := a.web.Search(ctx, in.Query)
	if err != nil {
		return problem(webFailure(err))
	}

	if len(results) == 0 {
		return taskapp.ToolResult{Output: "No results. Try other words."}
	}

	var out strings.Builder

	out.WriteString(webdomain.MarkUntrusted(""))

	for i, hit := range results {
		fmt.Fprintf(&out, "%d. %s\n   %s\n   %s\n", i+1, hit.Title, hit.URL, hit.Snippet)
	}

	return taskapp.ToolResult{Output: out.String()}
}

func (a *agentTools) readPage(ctx context.Context, in toolArgs) taskapp.ToolResult {
	page, err := a.web.Read(ctx, in.URL)
	if err != nil {
		return problem(webFailure(err))
	}

	note := ""
	if page.Cut {
		note = "\n[The page is longer; only the start is shown.]"
	}

	return taskapp.ToolResult{
		Output: webdomain.MarkUntrusted(fmt.Sprintf("Page: %s (%s)\n---\n%s%s",
			page.Title, page.URL, page.Text, note)),
	}
}

func (a *agentTools) findImages(ctx context.Context, projectID string, in toolArgs) taskapp.ToolResult {
	images, err := a.web.SearchImages(ctx, projectID, in.Query)
	if err != nil {
		return problem(webFailure(err))
	}

	if len(images) == 0 {
		return taskapp.ToolResult{Output: "No pictures with a usable licence. Try other words."}
	}

	var out strings.Builder

	out.WriteString(webdomain.MarkUntrusted("Pictures you may use (save one with download_asset, using its url):\n"))

	for i, image := range images {
		fmt.Fprintf(&out, "%d. %s by %s, %s, %dx%d, from %s\n   url: %s\n", i+1, image.Title,
			orUnknown(image.Creator), image.License, image.Width, image.Height, image.Source, image.URL)
	}

	return taskapp.ToolResult{Output: out.String()}
}

func orUnknown(name string) string {
	if name == "" {
		return "an unknown author"
	}

	return name
}

func (a *agentTools) saveAsset(ctx context.Context, projectID string, in toolArgs) taskapp.ToolResult {
	asset, err := a.web.Fetch(ctx, projectID, in.URL)
	if err != nil {
		return problem(webFailure(err))
	}

	services, err := a.services(projectID)
	if err != nil {
		return problem("The project folder is not available.")
	}

	target := assetTarget(in.Path, asset.Ext(), asset.Image.Title)

	applied, err := services.Editor.WriteAsset(target, asset.Data)
	if err != nil {
		return failure(err)
	}

	result := changed(applied, nil, target)
	result.Output = fmt.Sprintf("Saved %s (%d bytes). Add to assets/CREDITS.md (create it if missing):\n- %s: %s",
		target, len(asset.Data), target, asset.Image.Credit())

	return result
}

// maxNameLength caps a file name made from a picture title.
const maxNameLength = 40

var notNameChars = regexp.MustCompile(`[^a-z0-9]+`)

// assetTarget decides where a saved picture goes: the path the model gave, in assets/ unless it names a folder,
// with the extension that matches what the file really is.
func assetTarget(given, ext, title string) string {
	given = strings.TrimSpace(given)
	if given == "" {
		name := strings.Trim(notNameChars.ReplaceAllString(strings.ToLower(title), "-"), "-")
		if name == "" {
			name = "picture"
		}

		given = name[:min(len(name), maxNameLength)]
	}

	if !strings.Contains(given, "/") {
		given = "assets/" + given
	}

	return strings.TrimSuffix(given, path.Ext(given)) + ext
}

func webFailure(err error) string {
	for _, known := range []error{
		webdomain.ErrNotConfigured, webdomain.ErrBadQuery, webdomain.ErrBadURL, webdomain.ErrTooLarge,
		webdomain.ErrNotAnImage, webdomain.ErrNotAPage, webdomain.ErrNotListed, webdomain.ErrBusy, webdomain.ErrBlocked,
	} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}

	return webdomain.ErrUnavailable.Error()
}
