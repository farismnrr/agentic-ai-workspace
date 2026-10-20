package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// ErrNoCatalog means the program printed no list of models.
var ErrNoCatalog = errors.New("the program printed no list of models")

// listed is the visibility Codex gives a model it shows people.
const listed = "list"

type codexCatalog struct {
	Models []struct {
		Slug       string `json:"slug"`
		Name       string `json:"display_name"`
		Visibility string `json:"visibility"`
		Levels     []struct {
			Effort string `json:"effort"`
		} `json:"supported_reasoning_levels"`
		Priority int   `json:"priority"`
		Window   int64 `json:"context_window"`
	} `json:"models"`
}

// ParseCodexCatalog reads what `codex debug models` prints while signed in: the models Codex offers the account,
// best first, each with the effort levels it takes. Anything printed before the list (a warning) is skipped.
func ParseCodexCatalog(printed []byte) ([]Model, error) {
	start := bytes.Index(printed, []byte(`{"models"`))
	if start < 0 {
		return nil, ErrNoCatalog
	}

	var catalog codexCatalog
	if err := json.NewDecoder(bytes.NewReader(printed[start:])).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("read model list: %w", err)
	}

	items := catalog.Models
	sort.SliceStable(items, func(i, j int) bool { return items[i].Priority < items[j].Priority })

	models := []Model{}

	for _, item := range items {
		if item.Visibility != listed || item.Slug == "" {
			continue
		}

		efforts := make([]string, 0, len(item.Levels))
		for _, level := range item.Levels {
			efforts = append(efforts, level.Effort)
		}

		models = append(models, Model{ID: item.Slug, Name: item.Name, Efforts: efforts, ContextWindow: item.Window})
	}

	if len(models) == 0 {
		return nil, ErrNoCatalog
	}

	return models, nil
}
