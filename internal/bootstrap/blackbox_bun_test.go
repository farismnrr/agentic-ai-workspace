package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

// bunSteps is what the Developer does for a new app in the real toolbox, one run_command per step like in a real run
// (each step is its own container, so each has its own memory): start it from the starter, prove it with the checks
// QA and Security run, build it, then run the built server and use its login.
var bunSteps = []string{
	`bun --version
new-nuxt-app > /tmp/new.log 2>&1 || { tail -25 /tmp/new.log; echo STARTER_FAILED; exit 1; }
echo STARTER_OK`,
	`bun --bun run typecheck > /tmp/t.log 2>&1 || { tail -15 /tmp/t.log; echo TYPECHECK_FAILED; exit 1; }
echo TYPECHECK_OK
bun --bun run lint > /tmp/l.log 2>&1 || { tail -15 /tmp/l.log; echo LINT_FAILED; exit 1; }
echo LINT_OK
bun test 2>&1 | tail -3`,
	`bun --bun run build > /tmp/b.log 2>&1 || { tail -15 /tmp/b.log; echo BUILD_FAILED; exit 1; }
echo BUILD_OK
find . \( -path ./node_modules -o -path ./.nuxt -o -path ./.output \) -prune -o \
  \( -name '*.js' -o -name '*.mjs' -o -name '*.cjs' \) -print | head -3`,
	`PORT=4999 bun .output/server/index.mjs > /tmp/s.log 2>&1 &
sleep 8
B=http://127.0.0.1:4999
curl -s -X POST $B/api/auth/sign-up/email -H "content-type: application/json" -H "origin: $B" -c /tmp/jar \
  -d '{"email":"a@b.co","password":"correct horse battery","name":"Ann"}' | grep -q '"token"' && echo SIGNUP_OK
curl -s -b /tmp/jar $B/api/me | grep -q '"name":"Ann"' && echo SESSION_OK
test "$(curl -s -o /dev/null -w '%{http_code}' $B/api/me)" = 401 && echo PROTECTED_OK
kill %1 2>/dev/null || true`,
}

// TestLiveANewAppIsBuiltOnBunInTheRealSandbox runs a new Nuxt app from the toolbox's starter through the app's own
// run_command tool, the egress proxy and the per-person cache, exactly as the Developer would: install, type-check,
// lint, test, build, then run it and use its login. It needs Docker, the toolbox image with Bun and the internet:
//
//	BLACKBOX_LIVE=1 BLACKBOX_SANDBOX=<image> go test ./internal/bootstrap -run LiveANewApp
func TestLiveANewAppIsBuiltOnBunInTheRealSandbox(t *testing.T) {
	image := os.Getenv("BLACKBOX_SANDBOX")
	if os.Getenv("BLACKBOX_LIVE") == "" || image == "" {
		t.Skip("set BLACKBOX_LIVE=1 and BLACKBOX_SANDBOX=<toolbox image with Bun> to run this")
	}

	env := newEnvWith(t, func(c *config.Config) {
		c.SandboxImage = image
		c.SandboxEgress = []string{"registry.npmjs.org"}
	})

	steps := make([]string, 0, len(bunSteps)+1)
	for i, command := range bunSteps {
		id := "c" + string(rune('1'+i))
		steps = append(steps, toolCall(id, "run_command", map[string]any{"command": command, "timeout_seconds": 600}))
	}

	a := rigOn(t, env, append(steps, finishCall("Done"))...)
	id := a.start(t, "Build a new app")
	finishLive(t, a, id)

	out := strings.Join(a.toolResults(t), "\n")
	for _, want := range []string{
		"STARTER_OK", "TYPECHECK_OK", "LINT_OK", "BUILD_OK", "SIGNUP_OK", "SESSION_OK", "PROTECTED_OK",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the sandbox output lacks %q; its end: %s", want, out[max(0, len(out)-700):])
		}
	}

	for _, file := range []string{"bun.lock", "app/pages/index.vue", "server/infrastructure/db.ts", ".env"} {
		if _, err := os.Stat(a.file(file)); err != nil {
			t.Errorf("the project folder lacks %s: %v", file, err)
		}
	}
}
