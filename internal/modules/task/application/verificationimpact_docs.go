package application

import (
	"path"
	"strings"
)

// Documentation-only edits are not implementation changes. Keep agent
// playbooks, deployment configuration and runnable scripts out of this list.
func documentationOnlyPath(name string) bool {
	clean := strings.ToLower(path.Clean(name))
	if inDocuments(clean) {
		return true
	}
	if strings.HasPrefix(clean, "docs/") || strings.HasPrefix(clean, "project-notes/") {
		switch path.Ext(clean) {
		case ".md", ".txt", ".pdf", ".png", ".jpg", ".jpeg", ".webp", ".svg":
			return true
		}
	}
	switch path.Base(clean) {
	case "readme.md", "changelog.md", "contributing.md", "license", "license.md":
		return true
	}
	return false
}
