package domain

import (
	"strings"
	"testing"
)

func TestResearchEnoughRequiresSubstantiveSourcedRound(t *testing.T) {
	substance := strings.Repeat("Findings from the project and documented behavior. ", 9)
	withSource := substance + "\nSource: https://example.com/reference"
	cases := []struct {
		name   string
		rounds []string
		want   bool
	}{
		{name: "sourced note", rounds: []string{withSource}, want: true},
		{name: "note without source", rounds: []string{substance}},
		{name: "none"},
		{name: "short sourced placeholder", rounds: []string{"placeholder\nSource: https://example.com"}},
		{name: "later sourced round", rounds: []string{"short", withSource}, want: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, missing := ResearchEnough(test.rounds)
			if got != test.want || (got && len(missing) != 0) || (!got && len(missing) == 0) {
				t.Fatalf("ResearchEnough() = %v, %v", got, missing)
			}
		})
	}
}

func TestResearchEnoughCountsUnicodeRunesAndRejectsPlaceholders(t *testing.T) {
	source := "\nSource: https://example.test/reference"
	sourceRunes := len([]rune(source))
	short := strings.Repeat("é", minResearchRunes-sourceRunes-1) + source
	exact := strings.Repeat("é", minResearchRunes-sourceRunes) + source

	if ready, missing := ResearchEnough([]string{short}); ready || len(missing) != 1 {
		t.Fatalf("399 rune note = %v, %v", ready, missing)
	}
	if ready, missing := ResearchEnough([]string{exact}); !ready || len(missing) != 0 {
		t.Fatalf("400 rune note = %v, %v", ready, missing)
	}

	placeholder := strings.Repeat("verified findings ", 30) + "LOREM IPSUM" + source
	if ready, missing := ResearchEnough([]string{placeholder}); ready || len(missing) != 1 {
		t.Fatalf("placeholder note = %v, %v", ready, missing)
	}
}
