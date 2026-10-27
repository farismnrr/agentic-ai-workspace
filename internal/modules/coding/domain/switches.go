package domain

// followsRunCommand are the tools that are on exactly when run_command is.
var followsRunCommand = map[string]bool{
	ToolStartSite.Name: true, ToolStopSite.Name: true, ToolScreenshot.Name: true, ToolDeployExtension.Name: true,
	ToolRequestEgress.Name: true,
}

var followsLogMessage = map[string]bool{
	ToolBeginWork.Name: true, ToolSetRole.Name: true, ToolUpdatePlan.Name: true, ToolSaveResearch.Name: true,
	ToolSaveRequirements.Name: true, ToolReportProgress.Name: true,
	ToolCompleteWork.Name: true, ToolGetWorkState.Name: true,
}

// SwitchOf names the switch that governs a tool. Most tools have their own; starting and stopping the web
// app follow run_command.
func SwitchOf(tool string) string {
	if followsLogMessage[tool] {
		return ToolLogMessage.Name
	}

	if followsRunCommand[tool] {
		return ToolRunCommand.Name
	}

	if tool == ToolWebSearch.Name || tool == ToolFetchPage.Name || tool == ToolSearchImages.Name ||
		tool == ToolViewImage.Name {
		return ToolReadFile.Name
	}

	if tool == ToolDownloadAsset.Name {
		return ToolWriteFile.Name
	}

	return tool
}
