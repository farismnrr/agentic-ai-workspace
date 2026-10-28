package gitstatus

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const maxGitConfigBytes = 1 << 20

func validateGitMetadata(root string) error {
	gitDir := filepath.Join(root, ".git")
	metadata, err := os.Lstat(gitDir)
	if err != nil {
		return err
	}
	if !metadata.IsDir() {
		return errors.New("unsupported git metadata")
	}

	configPath := filepath.Join(gitDir, "config")
	configInfo, err := os.Lstat(configPath)
	if err == nil && (!configInfo.Mode().IsRegular() || configInfo.Size() > maxGitConfigBytes) {
		return errors.New("unsupported git config")
	}
	if err == nil {
		config, readErr := os.ReadFile(configPath)
		if readErr != nil || unsafeGitConfig(config) {
			return errors.New("unsafe git config")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("unsupported git config")
	}

	if _, err := os.Lstat(filepath.Join(gitDir, "objects", "info", "alternates")); err == nil {
		return errors.New("unsupported git alternates")
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("unsupported git alternates")
	}
	return nil
}

func unsafeGitConfig(data []byte) bool {
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			end := strings.IndexByte(line, ']')
			if end < 0 || strings.TrimSpace(line[end+1:]) != "" {
				return true
			}
			sectionParts := strings.Fields(strings.TrimSpace(line[1:end]))
			if len(sectionParts) == 0 {
				return true
			}
			section = strings.ToLower(sectionParts[0])
			if blockedGitSection(section) {
				return true
			}
			continue
		}
		if section == "" || strings.Contains(line, "\\") {
			return true
		}
		key := line
		if index := strings.IndexAny(key, "= \t"); index >= 0 {
			key = key[:index]
		}
		if blockedGitKey(section, strings.ToLower(key)) {
			return true
		}
	}
	return false
}

func blockedGitSection(section string) bool {
	switch section {
	case "filter", "diff", "include", "includeif", "pager", "alias", "credential", "difftool", "mergetool":
		return true
	default:
		return false
	}
}

func blockedGitKey(section, key string) bool {
	if section == "include" || section == "includeif" {
		return true
	}
	if section == "pager" || section == "filter" || section == "diff" || section == "difftool" || section == "mergetool" {
		return true
	}
	if section != "core" {
		return false
	}
	switch key {
	case "worktree", "fsmonitor", "hookspath", "pager", "sshcommand", "attributesfile", "editor", "askpass":
		return true
	default:
		return false
	}
}

func gitCommandArgs(root string, args ...string) []string {
	gitDir := filepath.Join(root, ".git")
	commandArgs := []string{"--git-dir=" + gitDir, "--work-tree=" + root, "-c", "core.fsmonitor=false",
		"-c", "core.attributesFile=/dev/null"}
	return append(commandArgs, args...)
}

func gitEnvironment() []string {
	return []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0", "GIT_ATTR_NOSYSTEM=1"}
}
