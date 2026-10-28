package bootstrap

import (
	"time"

	"github.com/gofiber/fiber/v3"

	accounthttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/http"
	githubhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/http"
	oauthhttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Limits are per client address unless noted. Sign-in counts only failures, so
// people who type their password correctly are never slowed down.
const (
	registerPerHour       = 5
	loginFailuresPerIP    = 30
	loginFailuresPerEmail = 8
	loginWindow           = 10 * time.Minute
	authorizePerMinute    = 60
	tokenPerMinute        = 120
	clientRegisterPerHour = 20
	githubPerMinute       = 20
	signInPerMinute       = 10
)

func accountLimits() accounthttp.Limits {
	return accounthttp.Limits{
		Register:   httpserver.Limit(registerPerHour, time.Hour, nil, false),
		LoginIP:    httpserver.Limit(loginFailuresPerIP, loginWindow, nil, true),
		LoginEmail: httpserver.Limit(loginFailuresPerEmail, loginWindow, httpserver.EmailKey, true),
	}
}

func githubLimits() githubhttp.Limits {
	return githubhttp.Limits{
		Start:    httpserver.Limit(githubPerMinute, time.Minute, nil, false),
		Callback: httpserver.Limit(githubPerMinute, time.Minute, nil, false),
	}
}

func oauthLimits() oauthhttp.Limits {
	return oauthhttp.Limits{
		Authorize: httpserver.Limit(authorizePerMinute, time.Minute, nil, false),
		Token:     httpserver.Limit(tokenPerMinute, time.Minute, nil, false),
		Register:  httpserver.Limit(clientRegisterPerHour, time.Hour, nil, false),
	}
}

// signInLimit guards the sign-in endpoints that each start a container.
func signInLimit() fiber.Handler {
	return httpserver.Limit(signInPerMinute, time.Minute, nil, false)
}
