package agentsession

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

func TestEveryScopeIdentityPartIsolated(t *testing.T) {
	registry := testRegistry(t, 5, time.Hour, 2*time.Hour, time.Second, time.Now)
	base := application.AgentSessionScope{UserID: "u", ProjectID: "p", TaskID: "t", ContainerID: "c"}
	scopes := []application.AgentSessionScope{
		base,
		{UserID: "other", ProjectID: "p", TaskID: "t", ContainerID: "c"},
		{UserID: "u", ProjectID: "other", TaskID: "t", ContainerID: "c"},
		{UserID: "u", ProjectID: "p", TaskID: "other", ContainerID: "c"},
		{UserID: "u", ProjectID: "p", TaskID: "t", ContainerID: "other"},
	}
	target := sessionURL(t)
	for i, scope := range scopes {
		setCookie(t, registry, scope, target, string(rune('a'+i)))
	}
	for i, scope := range scopes {
		assertCookie(t, registry, scope, target, string(rune('a'+i)))
	}
}

func TestIncompleteScopeIsRejected(t *testing.T) {
	registry := testRegistry(t, 1, time.Hour, time.Hour, time.Second, time.Now)
	err := registry.Do(t.Context(), application.AgentSessionScope{UserID: "u"},
		func(context.Context, http.CookieJar) error { return nil })
	if !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("incomplete scope error = %v", err)
	}
}
