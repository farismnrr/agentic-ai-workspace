package gitstatus

import (
	"fmt"
	"strings"
)

var metadataStateFiles = map[string]bool{
	"AUTO_MERGE": true, "BISECT_EXPECTED_REV": true, "BISECT_LOG": true, "BISECT_NAMES": true,
	"BISECT_START": true, "CHERRY_PICK_HEAD": true, "MERGE_HEAD": true, "MERGE_MODE": true,
	"MERGE_MSG": true, "MERGE_RR": true, "REBASE_HEAD": true, "REVERT_HEAD": true,
	"shallow": true, "SQUASH_MSG": true,
}

var rebaseApplyFiles = namesSet(
	"apply-opt", "applying", "author-script", "final-commit", "keep", "last", "message", "next",
	"onto", "orig-head", "patch", "quiet", "quoted-cr", "rebasing", "scissors", "sign", "threeway",
)

var rebaseMergeFiles = namesSet(
	"amend", "autostash", "current-fixups", "done", "drop_redundant_commits", "end", "git-rebase-todo",
	"git-rebase-todo.backup", "head-name", "interactive", "message", "msgnum", "no-reschedule-failed-exec",
	"onto", "orig-head", "patch", "rewritten-list", "squash-onto", "stopped-sha", "update-refs",
)

func namesSet(names ...string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func copyMetadataEntries(copier *metadataCopier, config safeGitConfig) error {
	entries, err := copier.copyDirectory(".")
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		seen[name] = true
		if entryErr := copyMetadataEntry(copier, name, config); entryErr != nil {
			return fmt.Errorf("copy Git metadata entry %q: %w", name, entryErr)
		}
	}
	if !seen["HEAD"] || !seen["objects"] {
		return errGitMetadata
	}
	return nil
}

func copyMetadataEntry(copier *metadataCopier, name string, config safeGitConfig) error {
	switch name {
	case "config":
		return nil
	case "HEAD":
		return copyHeadFile(copier, name, config.objectFormat)
	case "objects":
		return copier.copyObjectStore(config.objectFormat)
	case "refs":
		return copyReferenceDirectory(copier, name, "refs", config.objectFormat)
	case "info":
		return copyGitInfo(copier, name)
	case "packed-refs", "index":
		return copyMetadataFile(copier, name)
	case "rebase-apply", "rebase-merge":
		return copyRebaseState(copier, name)
	default:
		return copyOtherMetadata(copier, name)
	}
}

func copyHeadFile(copier *metadataCopier, name, format string) error {
	data, err := copier.copyFile(name, maxGitConfigBytes)
	if err != nil {
		return err
	}
	if !validHead(data, format) {
		return errGitMetadataEntry
	}
	return nil
}

func copyMetadataFile(copier *metadataCopier, name string) error {
	_, err := copier.copyFile(name, maxGitMetadataFileBytes)
	return err
}

func copyRebaseState(copier *metadataCopier, name string) error {
	if name == "rebase-apply" {
		return copyStateDirectory(copier, name, rebaseApplyFiles)
	}
	return copyStateDirectory(copier, name, rebaseMergeFiles)
}

func copyOtherMetadata(copier *metadataCopier, name string) error {
	if metadataStateFiles[name] {
		return copyMetadataFile(copier, name)
	}
	if ignoredMetadataDirectory(name) {
		_, err := copier.copyDirectory(name)
		return err
	}
	if ignoredMetadataFile(name) {
		return copier.skipRegularFile(name)
	}
	return errGitMetadataEntry
}

func ignoredMetadataDirectory(name string) bool {
	return name == "branches" || name == "hooks" || name == "logs" || name == "modules"
}

func ignoredMetadataFile(name string) bool {
	return name == "COMMIT_EDITMSG" || name == "FETCH_HEAD" || name == "ORIG_HEAD" || name == "description"
}

func copyGitInfo(copier *metadataCopier, path string) error {
	entries, err := copier.copyDirectory(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "exclude":
			if _, copyErr := copier.copyFile(path+"/exclude", maxGitMetadataFileBytes); copyErr != nil {
				return copyErr
			}
		case "refs":
			if skipErr := copier.skipRegularFile(path + "/refs"); skipErr != nil {
				return skipErr
			}
		default:
			return errGitMetadataEntry
		}
	}
	return nil
}

func copyStateDirectory(copier *metadataCopier, path string, allowed map[string]bool) error {
	entries, err := copier.copyDirectory(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !allowed[entry.Name()] {
			return errGitMetadataEntry
		}
		if _, copyErr := copier.copyFile(path+"/"+entry.Name(), maxGitMetadataFileBytes); copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func validHead(data []byte, format string) bool {
	value := strings.TrimSuffix(string(data), "\n")
	if strings.HasPrefix(value, "ref: ") {
		return validRefName(strings.TrimPrefix(value, "ref: "))
	}
	return len(value) == objectIDLength(format) && lowerHex(value)
}
