package domain

// IsWebTool reports whether the tool looks something up outside the project.
func IsWebTool(name string) bool {
	return name == ToolWebSearch.Name || name == ToolFetchPage.Name || name == ToolSearchImages.Name ||
		name == ToolDownloadAsset.Name
}
