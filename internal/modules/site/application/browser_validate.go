package application

import "strings"

func validRelativeSitePath(path string) bool {
	if strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\\?#%") || strings.Contains(path, ":") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	return true
}

func validLocator(kind, value string) bool {
	if value == "" || kind == "" {
		return false
	}
	switch kind {
	case "css", "text", "label", "placeholder", "role", "testid":
		return true
	default:
		return false
	}
}
