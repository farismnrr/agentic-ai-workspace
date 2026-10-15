package runner

import "context"

// githubHosts are what git and gh need besides the registries; only a workspace whose owner
// connected GitHub gets them.
var githubHosts = []string{"github.com", "api.github.com", "codeload.github.com"}

// Git is who a command acts as on GitHub: the person who owns the workspace.
type Git struct {
	Token string
	Name  string
	Email string
}

// GitSource returns the owner's GitHub credentials, or false when they have none. It is bound to
// one workspace's owner when the runner is made, so a command never chooses whose it gets.
type GitSource func(ctx context.Context) (Git, bool)

// helper makes git answer a password prompt for github.com with the token from the environment,
// so nothing is written to disk and the token never appears in a remote address.
const helper = `!f() { echo username=x-access-token; echo "password=$GH_TOKEN"; }; f`

// gitEnvironment is what makes git, gh and commits work as the owner. Git reads its configuration
// from GIT_CONFIG_* here, so no file in the workspace or the image ever holds the token.
func gitEnvironment(g Git) []string {
	return []string{
		"GH_TOKEN=" + g.Token, "GIT_TERMINAL_PROMPT=0", "GH_PROMPT_DISABLED=1",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=credential.https://github.com.helper", "GIT_CONFIG_VALUE_0=" + helper,
		"GIT_AUTHOR_NAME=" + g.Name, "GIT_AUTHOR_EMAIL=" + g.Email,
		"GIT_COMMITTER_NAME=" + g.Name, "GIT_COMMITTER_EMAIL=" + g.Email,
	}
}
