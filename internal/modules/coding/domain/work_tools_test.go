package domain

import (
	"slices"
	"testing"
)

func TestWorkToolsUseLogMessageSwitchAndApprovalGetsItsOwnCatalogSwitch(t *testing.T) {
	assertWorkToolSwitches(t)
	assertApprovalToolSwitch(t)
}

func assertWorkToolSwitches(t *testing.T) {
	t.Helper()
	workTools := []ToolInfo{
		ToolBeginWork, ToolSetRole, ToolUpdatePlan, ToolSaveResearch, ToolSaveRequirements,
		ToolReportProgress, ToolCompleteWork, ToolGetWorkState, ToolSubmitPlan,
	}
	for _, tool := range workTools {
		if tool.Scope != ScopeRead || SwitchOf(tool.Name) != ToolLogMessage.Name {
			t.Errorf("%s scope/switch = %q/%q", tool.Name, tool.Scope, SwitchOf(tool.Name))
		}
		for _, listed := range Tools() {
			if listed.Name == tool.Name {
				t.Errorf("%s unexpectedly appears in Settings tools", tool.Name)
			}
		}
	}
}

func assertApprovalToolSwitch(t *testing.T) {
	t.Helper()
	if ToolApprovePlan.Scope != ScopeApprove || ToolApprovePlan.Group != GroupExec ||
		SwitchOf(ToolApprovePlan.Name) != ToolApprovePlan.Name || ToolApprovePlan.DefaultOn {
		t.Fatalf("approval scope/switch/default = %q/%q/%v", ToolApprovePlan.Scope,
			SwitchOf(ToolApprovePlan.Name), ToolApprovePlan.DefaultOn)
	}
	if !slices.ContainsFunc(Tools(), func(tool ToolInfo) bool { return tool.Name == ToolApprovePlan.Name }) {
		t.Fatal("approve_plan must have an independent Settings switch")
	}
}
