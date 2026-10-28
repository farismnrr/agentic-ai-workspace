package github_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
	githubapi "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/githubapi"
	githubsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/adapter/sqlite"
	githubapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
	githubdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func withGitHub(c *config.Config) {
	c.GitHub = config.GitHub{ClientID: "cid", ClientSecret: "secret", AppSlug: "my-app"}
}

type fixedOwners map[string]string

func (f fixedOwners) Owner(_ context.Context, id string) (string, string, error) {
	if owner, ok := f[id]; ok {
		return owner, "project", nil
	}
	return "", "", errNoProject
}

var errNoProject = errors.New("no such project")

func TestCommandsActOnlyAsTheProjectOwner(t *testing.T) {
	t.Parallel()

	env := apptest.NewEnvWith(t, withGitHub)
	register := func(name string) string {
		return apptest.Register(t, env.App, name, name+"@example.com", "a sufficiently long password")
	}
	sari, bob := register("sari"), register("bob")
	idOf := func(cookie string) string {
		id, _ := apptest.Do(t, env.App, "GET", "/api/auth/me", "", map[string]string{"Cookie": cookie}).
			JSON(t)["id"].(string)

		return id
	}

	sealer, err := secretbox.Open(env.DataDir)
	if err != nil {
		t.Fatal(err)
	}

	sealed, _ := sealer.Encrypt([]byte("ghu_sari"))
	record := githubapp.Record{
		UserID: idOf(sari), Profile: githubdomain.Profile{ID: 1, Login: "sari"}, AccessSealed: sealed,
		RefreshSealed: sealed, AccessExpires: time.Now().Add(time.Hour),
	}

	gh := githubsqlite.NewStore(env.DB)
	if err = gh.Save(context.Background(), record, time.Now()); err != nil {
		t.Fatal(err)
	}

	client := githubapi.New("cid", "s", "", time.Now)
	service := githubapp.NewService(gh, client, sealer, opaque.Generator{}, time.Now, true)
	git := bootstrap.GitHubSourceForProject(service,
		fixedOwners{"sari-project": idOf(sari), "bob-project": idOf(bob)})

	got, ok := git("sari-project")(context.Background())
	if !ok || got.Token != "ghu_sari" || got.Email != "1+sari@users.noreply.github.com" {
		t.Fatalf("the owner's own project: %+v %v", got, ok)
	}

	for _, project := range []string{"bob-project", "unknown-project"} {
		if leaked, found := git(project)(context.Background()); found || leaked.Token != "" {
			t.Fatalf("%s must not get anyone's token: %+v", project, leaked)
		}
	}
}
