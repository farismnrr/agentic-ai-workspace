package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
)

func (a *agentTools) projectCommit(projectID string) string {
	if a.folders == nil {
		return ""
	}
	dir, err := a.folders.Dir(projectID)
	if err != nil {
		return ""
	}
	headFile := filepath.Join(dir, ".git", "HEAD")
	data, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(data))
	if strings.HasPrefix(ref, "ref: ") {
		refPath := strings.TrimSpace(strings.TrimPrefix(ref, "ref: "))
		refFile := filepath.Join(dir, ".git", refPath)
		if commitData, err := os.ReadFile(refFile); err == nil {
			return strings.TrimSpace(string(commitData))
		}
		packedFile := filepath.Join(dir, ".git", "packed-refs")
		if packed, err := os.ReadFile(packedFile); err == nil {
			for _, line := range strings.Split(string(packed), "\n") {
				parts := strings.Fields(line)
				if len(parts) == 2 && parts[1] == refPath {
					return parts[0]
				}
			}
		}
		return refPath
	}
	return ref
}
