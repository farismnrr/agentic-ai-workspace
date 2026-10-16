package application

import (
	"strings"
	"testing"
)

func TestPromptNamesTheInstalledToolchainsOnlyWhenKnown(t *testing.T) {
	with := systemPrompt("Site", Persona{}, []string{"Python 3.13", "Go 1.27"}, "")
	wanted := []string{"run_command", "Python 3.13, Go 1.27", "package registries", "start_site", "relative links"}
	for _, want := range wanted {
		if !strings.Contains(with, want) {
			t.Errorf("prompt should mention %q", want)
		}
	}

	if without := systemPrompt("Site", Persona{}, nil, ""); strings.Contains(without, "Installed there") {
		t.Error("a prompt must not promise toolchains it does not know about")
	}
}

func TestSmallWorkNeedsNoPlanButBiggerWorkDoes(t *testing.T) {
	prompt := systemPrompt("Site", Persona{}, nil, "")

	for _, want := range []string{"needs no plan", "one outcome you can check", "Keep exactly one step active"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the workflow should say %q", want)
		}
	}
}
