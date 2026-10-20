package domain_test

import (
	_ "embed"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

//go:embed testdata/codex-models.json
var codexModels []byte

func TestParseCodexCatalogKeepsListedModelsBestFirst(t *testing.T) {
	t.Parallel()

	printed := append([]byte("WARNING: could not create aliases\n"), codexModels...)

	models, err := domain.ParseCodexCatalog(printed)
	if err != nil {
		t.Fatalf("ParseCodexCatalog: %v", err)
	}

	if len(models) != 2 || models[0].ID != "gpt-big" || models[1].ID != "gpt-small" {
		t.Fatalf("models = %+v, want gpt-big then gpt-small", models)
	}

	if got := models[0].Efforts; len(got) != 3 || got[2] != "high" {
		t.Fatalf("efforts = %v", got)
	}
}

func TestParseCodexCatalogWithoutAListFails(t *testing.T) {
	t.Parallel()

	if _, err := domain.ParseCodexCatalog([]byte("nothing here")); err == nil {
		t.Fatal("want an error")
	}
}

func TestChoice(t *testing.T) {
	t.Parallel()

	models := []domain.Model{{ID: "m", Efforts: []string{"low", "high"}}, {ID: "default", Efforts: []string{"low"}}}
	cases := []struct{ model, effort, wantModel, wantEffort string }{
		{"m", "high", "m", "high"},
		{"m", "max", "m", ""},
		{"default", "low", "", "low"},
		{"default", "high", "", ""},
		{"gone", "low", "", ""},
	}

	for _, c := range cases {
		model, effort := domain.Choice(models, c.model, c.effort)
		if model != c.wantModel || effort != c.wantEffort {
			t.Errorf("Choice(%q, %q) = %q, %q", c.model, c.effort, model, effort)
		}
	}
}

func TestParseCodexCatalogKeepsEachModelsWindow(t *testing.T) {
	t.Parallel()

	models, err := domain.ParseCodexCatalog(codexModels)
	if err != nil || models[0].ContextWindow != 272_000 || models[1].ContextWindow != 0 {
		t.Fatalf("windows = %+v, %v", models, err)
	}
}
