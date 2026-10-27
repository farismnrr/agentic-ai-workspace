package domain

import "testing"

func TestWorkToolsUseLogMessageSwitchWithoutJoiningSettingsCatalog(t *testing.T) {
	workTools := []ToolInfo{
		ToolBeginWork, ToolSetRole, ToolUpdatePlan, ToolSaveResearch, ToolSaveRequirements,
		ToolReportProgress, ToolCompleteWork, ToolGetWorkState,
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
