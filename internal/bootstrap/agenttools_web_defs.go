package bootstrap

import (
	"encoding/json"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// The web tools look things up outside the project. What they return is written by strangers, so each answer
// says so, and only roles that may search the web get them (never the orchestrator).
var (
	toolWebSearch     = codingdomain.ToolWebSearch.Name
	toolFetchPage     = codingdomain.ToolFetchPage.Name
	toolSearchImages  = codingdomain.ToolSearchImages.Name
	toolDownloadAsset = codingdomain.ToolDownloadAsset.Name
)

// maxQueryShown is how much of a search the thread shows.
const maxQueryShown = 60

func schema(properties, required string) json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` + properties + `},"required":[` + required + `]}`)
}

// webDefinitions describes the web tools. Search is left out when the server has no search engine to ask.
func (a *agentTools) webDefinitions() []taskdomain.ToolDef {
	if a.web == nil {
		return nil
	}

	defs := []taskdomain.ToolDef{
		{
			Name: toolFetchPage, Description: codingdomain.ToolFetchPage.Description,
			Schema: schema(`"url":{"type":"string","description":"Full http or https address"}`, `"url"`),
		},
		{
			Name: toolSearchImages, Description: codingdomain.ToolSearchImages.Description,
			Schema: schema(`"query":{"type":"string","description":"What the picture shows, in a few words"}`, `"query"`),
		},
		{
			Name: toolDownloadAsset, Description: codingdomain.ToolDownloadAsset.Description,
			Schema: schema(`"url":{"type":"string","description":"The picture's url exactly as search_images gave it"},`+
				`"path":{"type":"string","description":"Where to save it, such as assets/hero.jpg; `+
				`default is assets/ plus a name"}`, `"url"`),
		},
	}

	if a.web.CanSearch() {
		defs = append([]taskdomain.ToolDef{{
			Name: toolWebSearch, Description: codingdomain.ToolWebSearch.Description,
			Schema: schema(`"query":{"type":"string","description":"The search"}`, `"query"`),
		}}, defs...)
	}

	return defs
}

func describeWeb(tool string, in toolArgs) taskapp.Described {
	switch tool {
	case toolWebSearch:
		return said("Searched the web for "+clipWords(in.Query), "", "web_search "+in.Query)
	case toolFetchPage:
		return said("Read a web page", "", "fetch_page "+in.URL)
	case toolSearchImages:
		return said("Looked for pictures of "+clipWords(in.Query), "", "search_images "+in.Query)
	}

	return said("Saved a picture to "+assetTarget(in.Path, ".jpg", ""),
		"May I save a picture from the web into your project?", "download_asset "+in.URL+" → "+in.Path)
}

func clipWords(text string) string {
	if runes := []rune(strings.TrimSpace(text)); len(runes) > maxQueryShown {
		return string(runes[:maxQueryShown]) + "…"
	}

	return strings.TrimSpace(text)
}
