package application_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func TestStartsServerCatchesDetachedServersEvenWhenSomethingIsKilled(t *testing.T) {
	t.Parallel()

	for command, want := range map[string]bool{
		`cd app && nohup npx next dev -p 3105 > dev.log 2>&1 &`:              true,
		`pkill -f "next dev"; (nohup npx next dev -p 3105 > dev.log 2>&1 &)`: true,
		`setsid node server.js < /dev/null & disown`:                         true,
		`npx next dev`: true,
		`node server.js & sleep 2; curl -s localhost:3000 > out.txt; kill $!`: false,
		`bun run build`:                                        false,
		`npm run typecheck && npm test`:                        false,
		`sqlite3 data/app.sqlite "select count(*) from items"`: false,
		`bun x vite build`:                                     false,
	} {
		if got := application.StartsServer(command); got != want {
			t.Errorf("StartsServer(%q) = %v, want %v", command, got, want)
		}
	}
}
