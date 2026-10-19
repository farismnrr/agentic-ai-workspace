package domain

import (
	"slices"
	"testing"
)

func TestTheDocumentsRoleRunsCommandsAndDoesNotBrowse(t *testing.T) {
	t.Parallel()

	for _, member := range Presets().Members() {
		if member.Key == "documents" {
			if !slices.Contains(member.Access, AccessExec) || slices.Contains(member.Access, AccessWeb) {
				t.Fatalf("access = %v", member.Access)
			}

			return
		}
	}

	t.Fatal("the Documents role is missing")
}

func TestTheDocumentsRoleHasOneStandingPlaybookAndOneSkillPerFormat(t *testing.T) {
	t.Parallel()

	always, loadable := []string{}, []string{}

	for _, skill := range SkillsFor("documents") {
		if skill.Always {
			always = append(always, skill.Name)
		} else {
			loadable = append(loadable, skill.Name)
		}
	}

	slices.Sort(loadable)

	if !slices.Equal(always, []string{"document-workflow"}) || !slices.Equal(loadable, []string{"docx", "pdf", "pptx"}) {
		t.Fatalf("always %v, loadable %v", always, loadable)
	}
}
