package domain_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

func TestNamesThatMakeTheSameAddressCollide(t *testing.T) {
	existing := []domain.Named{{ID: "a", Name: "Bakery Site"}, {ID: "b", Name: "Bakery 2"}}

	cases := map[string]bool{"bakery-site": true, "  BAKERY   site ": true, "Bakery": false, "Bakery 3": false}
	for name, want := range cases {
		if got := domain.Taken(name, "", existing); got != want {
			t.Errorf("%q: taken %v, want %v", name, got, want)
		}
	}

	if domain.Taken("Bakery Site", "a", existing) {
		t.Error("a project does not collide with itself")
	}
}

func TestSuggestFindsTheFirstFreeNumber(t *testing.T) {
	existing := []domain.Named{{ID: "a", Name: "Bakery"}, {ID: "b", Name: "Bakery 2"}}

	if got := domain.Suggest("Bakery", "", existing); got != "Bakery 3" {
		t.Errorf("got %q", got)
	}

	if got := domain.Suggest("Cafe", "", existing); got != "Cafe" {
		t.Errorf("a free name is kept: %q", got)
	}
}
