package application

import (
	"path"
	"slices"
	"strings"
)

// cleanScope normalizes relative file or folder prefixes and drops root or parent-traversal entries.
func cleanScope(raw []string) []string {
	var out []string
	for _, item := range raw {
		cleaned, ok := cleanScopePath(item)
		if ok && !slices.Contains(out, cleaned) {
			out = append(out, cleaned)
		}
	}
	return out
}

func cleanScopePath(raw string) (string, bool) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if strings.HasPrefix(trimmed, "/") || trimmed == "" {
		return "", false
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func validScope(raw []string) bool {
	for _, item := range raw {
		if _, ok := cleanScopePath(item); !ok {
			return false
		}
	}
	return true
}

// pathsOverlap reports whether two normalized paths name the same file/folder or one is inside the other.
func pathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// scopesOverlap reports whether any path prefix in a overlaps with any path prefix in b.
func scopesOverlap(a, b []string) bool {
	for _, left := range a {
		for _, right := range b {
			if pathsOverlap(left, right) {
				return true
			}
		}
	}
	return false
}

// inScope reports whether target lies inside at least one prefix of scope.
func inScope(target string, scope []string) bool {
	cleaned, ok := cleanScopePath(target)
	if !ok {
		return false
	}
	for _, prefix := range scope {
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
			return true
		}
	}
	return false
}
