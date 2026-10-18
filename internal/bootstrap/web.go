package bootstrap

import (
	"time"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/adapter/commons"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/adapter/fetch"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/adapter/openverse"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/adapter/searxng"
	webapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/web/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// newWeb builds what the agent uses to look things up. Pictures come from two free libraries that need no key; web
// search needs a SearXNG server (SEARXNG_URL) and is left off without one. Every address the agent reads goes
// through the egress guard, so only the public internet can be reached.
func newWeb(cfg config.Config) *webapp.Service {
	client := fetch.NewClient(egress.Policy{Rule: egress.PublicOnly})
	sources := []webapp.ImageSource{openverse.New(openverse.DefaultBase, client), commons.New(commons.DefaultBase, client)}

	var searcher webapp.Searcher
	if cfg.SearxngURL != "" {
		searcher = searxng.New(cfg.SearxngURL)
	}

	return webapp.NewService(searcher, sources, fetch.New(client), time.Now)
}

// newToolbox builds the tools the agent works with, web lookups included.
func newToolbox(folders *folder.Folders, cfg config.Config, engine *codingrunner.Engine) (*agentTools, error) {
	tools, err := newAgentTools(folders, cfg.DataDir, cfg.PublicURL, engine)
	if err != nil {
		return nil, err
	}

	tools.web = newWeb(cfg)

	return tools, nil
}
