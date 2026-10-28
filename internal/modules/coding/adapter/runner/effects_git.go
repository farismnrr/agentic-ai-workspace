package runner

import (
	"fmt"
	"io/fs"
	"strings"
)

func skipInertGitPath(name string, entry fs.DirEntry) (bool, error) {
	if name == ".git/objects/info" {
		if entry.IsDir() {
			return false, nil
		}
		return true, fmt.Errorf("%w: object metadata is not a directory", errGitMetadataOutside)
	}
	return skipGitContents(name, entry)
}

func skipGitContents(name string, entry fs.DirEntry) (bool, error) {
	if strings.HasPrefix(name, ".git/objects/info/") {
		return false, nil
	}
	if strings.HasPrefix(name, ".git/logs/") || name == ".git/objects/pack" ||
		strings.HasPrefix(name, ".git/objects/pack/") {
		return skipGitEntry(entry)
	}
	if strings.HasPrefix(name, ".git/objects/") {
		return skipGitEntry(entry)
	}
	if name == ".git/logs" {
		return skipGitEntry(entry)
	}
	return false, nil
}

func skipGitEntry(entry fs.DirEntry) (bool, error) {
	if entry.IsDir() {
		return true, fs.SkipDir
	}
	return true, nil
}
