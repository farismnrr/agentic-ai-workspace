package sites_test

import (
	"testing"

	siteapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

func TestSiteStartRejectsArbitraryShellAndChecks(t *testing.T) {
	for _, command := range []string{
		"go test ./...", "go test ./... && python3 -m http.server 8123",
		"npm test", "npm run test", "bash -c python3",
		"python3 -c print(1)", "node -e console.log(1)",
		"python3 -m http.server 8123; go test ./...",
		"npm start $(go test ./...)", "npm start | tee /tmp/log",
		"node server.js\ngo test ./...",
	} {
		if _, err := siteapp.SiteArgv(command); err == nil {
			t.Errorf("start command must reject %q", command)
		}
	}
	for _, command := range []string{
		"npm start", "npm run preview", "pnpm run start",
		"node server.js", "python3 -m http.server 8123 --bind 127.0.0.1",
		"go run ./cmd/web", "npx vite preview",
	} {
		if argv, err := siteapp.SiteArgv(command); err != nil || len(argv) < 2 {
			t.Errorf("legitimate server start must work: %q (%v)", command, err)
		}
	}
}
