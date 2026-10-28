package application

import (
	"path"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func pathImpact(name string) verificationImpact {
	clean := strings.ToLower(path.Clean(name))
	base := path.Base(clean)
	if inDocuments(clean) {
		return verificationImpact{}
	}
	if apiBoundaryPath(clean) {
		return verificationImpact{
			security: true, architecture: true, unknown: true,
			reason: name + " is a server/API boundary whose behavior needs review",
		}
	}
	if securityPath(clean, base) {
		reason := name + " touches a sensitive boundary or configuration"
		if dependencyFile(base) {
			reason = name + " changes dependencies; include affected callers and dependants"
		}
		return verificationImpact{
			security: true, architecture: architecturePath(clean, base) || dependencyFile(base), reason: reason,
		}
	}
	if sourcePath(base) {
		return verificationImpact{
			security: true, architecture: true, unknown: true,
			reason: name + " has executable behavior whose impact is not evident from its path",
		}
	}
	if architecturePath(clean, base) {
		return verificationImpact{architecture: true, reason: name + " changes a module contract or shared configuration"}
	}
	if presentationPath(clean, base) {
		return verificationImpact{reason: name + " is a presentation change"}
	}
	if knownProjectPath(clean, base) {
		return verificationImpact{reason: name + " is an application change"}
	}
	return verificationImpact{
		security: true, architecture: true, unknown: true,
		reason: name + " has unclassified impact; applying conservative reviews",
	}
}

func apiBoundaryPath(name string) bool {
	return strings.Contains(name, "/server/api/") || strings.HasPrefix(name, "server/api/")
}

func presentationPath(name, base string) bool {
	return (strings.HasPrefix(name, "web/src/") || strings.HasPrefix(name, "web/")) &&
		slices.Contains([]string{".css", ".scss"}, path.Ext(base))
}

func sourcePath(base string) bool {
	return slices.Contains([]string{
		".ts", ".tsx", ".js", ".mjs", ".cjs", ".py", ".go", ".rs", ".java", ".kt", ".rb", ".php", ".sql",
		".vue", ".svelte", ".html", ".htm",
	}, path.Ext(base))
}

func changedPaths(events []domain.Event) []string {
	seen := make(map[string]struct{})
	var files []string
	for _, event := range events {
		if event.Undone {
			continue
		}
		files = appendNewPaths(files, seen, event.Files)
		if event.Run != nil && event.Run.EffectsKnown {
			files = appendNewPaths(files, seen, event.Run.EffectPaths)
		}
	}
	return files
}

func appendNewPaths(files []string, seen map[string]struct{}, paths []string) []string {
	for _, name := range paths {
		clean := path.Clean(name)
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		files = append(files, clean)
	}
	return files
}

func securityPath(name, base string) bool {
	for _, marker := range []string{
		"auth", "security", "permission", "credential", "secret", "token", "session", "oauth", "egress",
		"sandbox", "guard", "migration", "migrations", "database", "sqlite", "release", "workflow",
	} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return base == "dockerfile" || base == "makefile" || dependencyFile(base) || path.Ext(base) == ".sql" ||
		strings.HasPrefix(base, ".env") || strings.Contains(name, "config") ||
		slices.Contains([]string{".yaml", ".yml", ".toml"}, path.Ext(base))
}

func architecturePath(name, base string) bool {
	for _, marker := range []string{"/domain/", "/application/", "/ports/", "/contracts/", "/shared/"} {
		if strings.Contains("/"+name, marker) {
			return true
		}
	}
	return dependencyFile(base) || base == "tsconfig.json" || strings.Contains(name, "config")
}

func dependencyFile(base string) bool {
	return slices.Contains([]string{
		"go.mod", "go.sum", "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock",
		"bun.lockb", "cargo.toml", "cargo.lock", "pyproject.toml", "requirements.txt", "poetry.lock", "gemfile",
		"gemfile.lock", "composer.json", "composer.lock", "pom.xml", "build.gradle", "build.gradle.kts",
		"gradle.lockfile", "deno.lock",
	}, base)
}

func knownProjectPath(name, base string) bool {
	ext := path.Ext(base)
	knownExtensions := []string{
		".ts", ".tsx", ".js", ".mjs", ".cjs", ".vue", ".py", ".go", ".rs", ".java", ".kt", ".rb", ".php",
		".sql", ".sh", ".css", ".scss", ".html", ".htm", ".svelte", ".md", ".yaml", ".yml", ".json",
		".toml", ".mod", ".sum", ".lock", ".txt",
	}
	if slices.Contains(knownExtensions, ext) {
		return true
	}
	return base == "makefile" || base == "dockerfile" || base == ".gitignore" ||
		base == ".dockerignore" || strings.HasPrefix(name, "documents/")
}
