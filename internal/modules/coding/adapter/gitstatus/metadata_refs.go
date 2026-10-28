package gitstatus

import (
	"io/fs"
	"strings"
)

const maxGitRefDepth = 32

func copyReferenceDirectory(copier *metadataCopier, path, refPath, format string) error {
	if refPath != "refs" && !validRefName(refPath) {
		return errGitMetadataEntry
	}
	entries, err := copier.copyDirectory(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entryErr := copyReferenceEntry(copier, path, refPath, entry, format); entryErr != nil {
			return entryErr
		}
	}
	return nil
}

func copyReferenceEntry(copier *metadataCopier, path, refPath string, entry fs.DirEntry, format string) error {
	childPath := path + "/" + entry.Name()
	childRef := refPath + "/" + entry.Name()
	if entry.IsDir() {
		if strings.Count(childRef, "/") > maxGitRefDepth {
			return errGitSnapshotLimit
		}
		return copyReferenceDirectory(copier, childPath, childRef, format)
	}
	if !validRefName(childRef) {
		return errGitMetadataEntry
	}
	contents, err := copier.copyFile(childPath, maxGitMetadataFileBytes)
	if err != nil {
		return err
	}
	if !validRefContents(contents, format) {
		return errGitMetadataEntry
	}
	return nil
}

func validRefContents(contents []byte, format string) bool {
	value := strings.TrimSuffix(string(contents), "\n")
	if strings.HasPrefix(value, "ref: ") {
		return validRefName(strings.TrimPrefix(value, "ref: "))
	}
	return len(value) == objectIDLength(format) && lowerHex(value)
}

func validRefName(name string) bool {
	if !strings.HasPrefix(name, "refs/") || strings.HasSuffix(name, "/") ||
		strings.Contains(name, "//") || strings.Contains(name, "..") || strings.Contains(name, "@{") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if !validRefPart(part) {
			return false
		}
	}
	return true
}

func validRefPart(part string) bool {
	if part == "" || part == "." || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".") ||
		strings.HasSuffix(part, ".lock") {
		return false
	}
	return validRefCharacters(part)
}

func validRefCharacters(part string) bool {
	for _, char := range part {
		if invalidRefCharacter(char) {
			return false
		}
	}
	return true
}

func invalidRefCharacter(char rune) bool {
	return char < 0x20 || char == 0x7f || char == ' ' || strings.ContainsRune("~^:?*[\\", char)
}
