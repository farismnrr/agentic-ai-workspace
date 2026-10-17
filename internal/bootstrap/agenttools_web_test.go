package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	webapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/web/application"
	webdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

const signURL = "https://img.example/sign.jpg"

type stubSearch struct{}

func (stubSearch) Search(context.Context, string, int) ([]webdomain.Result, error) {
	return []webdomain.Result{{Title: "Kopi Teras", URL: "https://example.com/kopi", Snippet: "Warung kopi"}}, nil
}

type stubPictures struct{}

func (stubPictures) Name() string { return "stub" }

func (stubPictures) Search(context.Context, string, int) ([]webdomain.Image, error) {
	return []webdomain.Image{
		{
			Title: "Coffee Shop Sign", Creator: "pixeljones", License: "CC BY-SA 2.0",
			URL: signURL, Page: "https://example.com/sign", Width: 1024, Height: 758,
		},
		{Title: "Not for sale", License: "CC BY-NC 4.0", URL: "https://img.example/nc.jpg", Page: "https://example.com/nc"},
	}, nil
}

type stubFetch struct{}

func (stubFetch) Page(context.Context, string) (webdomain.Page, error) {
	return webdomain.Page{URL: "https://example.com/kopi", Title: "Kopi Teras", Text: "Buka setiap hari"}, nil
}

func (stubFetch) Image(context.Context, string, int64) ([]byte, string, error) {
	return []byte("\x89PNG\r\n\x1a\nxxxx"), "image/png", nil
}

func webTools(t *testing.T) (*agentTools, string) {
	t.Helper()

	dir := t.TempDir()

	folders, err := folder.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	tools, err := newAgentTools(folders, dir, "http://x", nil)
	if err != nil {
		t.Fatal(err)
	}

	pictures := []webapp.ImageSource{stubPictures{}}
	tools.web = webapp.NewService(stubSearch{}, pictures, stubFetch{}, time.Now)

	const project = "abcdef0123456789"
	if err = folders.Create(project); err != nil {
		t.Fatal(err)
	}

	return tools, project
}

func call(name string, args map[string]string) taskapp.Call {
	input, _ := json.Marshal(args)

	return taskapp.Call{ID: "c1", Name: name, Input: input}
}

func TestTheWebToolsReturnTextMarkedAsFromStrangers(t *testing.T) {
	t.Parallel()

	tools, project := webTools(t)
	calls := []taskapp.Call{
		call("web_search", map[string]string{"query": "warung kopi"}),
		call("fetch_page", map[string]string{"url": "https://example.com/kopi"}),
	}

	for _, c := range calls {
		result := tools.Execute(context.Background(), project, c)
		if result.IsError || !strings.HasPrefix(result.Output, untrusted) || !strings.Contains(result.Output, "Kopi Teras") {
			t.Fatalf("%s: %+v", c.Name, result)
		}
	}
}

func TestOnlyPicturesFromTheLatestSearchOfferedWithTheirLicenceCanBeSaved(t *testing.T) {
	t.Parallel()

	tools, project := webTools(t)
	ctx := context.Background()

	refused := tools.Execute(ctx, project, call("download_asset", map[string]string{"url": signURL}))
	if !refused.IsError {
		t.Fatalf("nothing was searched yet: %+v", refused)
	}

	found := tools.Execute(ctx, project, call("search_images", map[string]string{"query": "coffee shop"}))
	if found.IsError || strings.Contains(found.Output, "nc.jpg") || !strings.Contains(found.Output, "CC BY-SA 2.0") {
		t.Fatalf("only usable licences are offered: %+v", found)
	}
}

func TestASavedPictureKeepsItsTypeAndComesWithItsCredit(t *testing.T) {
	t.Parallel()

	tools, project := webTools(t)
	ctx := context.Background()
	tools.Execute(ctx, project, call("search_images", map[string]string{"query": "coffee shop"}))

	saved := tools.Execute(ctx, project, call("download_asset", map[string]string{"url": signURL, "path": "hero"}))
	if saved.IsError || len(saved.Undo) == 0 || len(saved.Files) != 1 || saved.Files[0] != "assets/hero.png" {
		t.Fatalf("saved in assets/ with the extension the file really has: %+v", saved)
	}

	if !strings.Contains(saved.Output, "\"Coffee Shop Sign\" by pixeljones, CC BY-SA 2.0") {
		t.Fatalf("the credit line comes back: %s", saved.Output)
	}

	root, err := tools.folders.Dir(project)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = os.Stat(filepath.Join(root, "assets", "hero.png")); err != nil {
		t.Fatal(err)
	}

	replace := call("download_asset", map[string]string{"url": signURL, "path": "assets/hero.png"})
	again := tools.Execute(ctx, project, replace)
	if !again.IsError {
		t.Fatalf("a saved asset is never replaced: %+v", again)
	}
}

func TestTargetsAreInAssetsWithTheRightExtension(t *testing.T) {
	t.Parallel()

	for given, want := range map[string]string{
		"": "assets/coffee-shop-sign.jpg", "hero": "assets/hero.jpg", "hero.png": "assets/hero.jpg",
		"assets/original/hero.jpeg": "assets/original/hero.jpg", "img/a": "img/a.jpg",
	} {
		if got := assetTarget(given, ".jpg", "Coffee Shop Sign"); got != want {
			t.Errorf("%q → %q, want %q", given, got, want)
		}
	}
}
