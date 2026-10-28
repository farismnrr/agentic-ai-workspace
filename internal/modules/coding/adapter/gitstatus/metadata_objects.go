package gitstatus

import (
	"fmt"
	"strings"
)

const (
	sha1ObjectIDLength   = 40
	sha256ObjectIDLength = 64
)

func (c *metadataCopier) copyObjectStore(format string) error {
	entries, err := c.copyDirectory("objects")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		path := "objects/" + name
		switch name {
		case "info":
			err = c.copyObjectInfo(path)
		case "pack":
			err = c.copyPackDirectory(path, format)
		default:
			err = c.copyLooseObjects(path, name, format)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *metadataCopier) copyObjectInfo(path string) error {
	entries, err := c.copyDirectory(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "alternates", "http-alternates":
			return errGitAlternates
		case "packs", "commit-graph", "commit-graph-chain":
			if err := c.skipRegularFile(path + "/" + entry.Name()); err != nil {
				return err
			}
		default:
			return errGitMetadataEntry
		}
	}
	return nil
}

func (c *metadataCopier) copyPackDirectory(path, format string) error {
	entries, err := c.copyDirectory(path)
	if err != nil {
		return err
	}
	packs := make(map[string]map[string]bool)
	for _, entry := range entries {
		if entryErr := c.copyPackEntry(path, entry.Name(), format, packs); entryErr != nil {
			return entryErr
		}
	}
	return requireCompletePacks(packs)
}

func (c *metadataCopier) copyPackEntry(path, name, format string, packs map[string]map[string]bool) error {
	base, extension, ok := splitPackName(name, format)
	if !ok || packs[base][extension] {
		return errGitMetadataEntry
	}
	if packs[base] == nil {
		packs[base] = make(map[string]bool)
	}
	packs[base][extension] = true
	filePath := path + "/" + name
	if extension == "pack" || extension == "idx" {
		_, err := c.copyFile(filePath, maxGitMetadataFileBytes)
		return err
	}
	return c.skipRegularFile(filePath)
}

func requireCompletePacks(packs map[string]map[string]bool) error {
	for _, extensions := range packs {
		if !extensions["pack"] || !extensions["idx"] {
			return fmt.Errorf("incomplete Git pack pair: %w", errGitMetadataEntry)
		}
	}
	return nil
}

func splitPackName(name, format string) (string, string, bool) {
	if !strings.HasPrefix(name, "pack-") {
		return "", "", false
	}
	index := strings.LastIndexByte(name, '.')
	if index <= len("pack-") || index == len(name)-1 {
		return "", "", false
	}
	objectID := name[len("pack-"):index]
	if len(objectID) != objectIDLength(format) || !lowerHex(objectID) {
		return "", "", false
	}
	extension := name[index+1:]
	switch extension {
	case "pack", "idx", "bitmap", "rev", "keep", "mtimes":
		return "pack-" + objectID, extension, true
	default:
		return "", "", false
	}
}

func (c *metadataCopier) copyLooseObjects(path, prefix, format string) error {
	if len(prefix) != 2 || !lowerHex(prefix) {
		return errGitMetadataEntry
	}
	entries, err := c.copyDirectory(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) != objectIDLength(format)-2 || !lowerHex(name) {
			return errGitMetadataEntry
		}
		if _, err := c.copyFile(path+"/"+name, maxGitMetadataFileBytes); err != nil {
			return err
		}
	}
	return nil
}

func objectIDLength(format string) int {
	if format == "sha256" {
		return sha256ObjectIDLength
	}
	return sha1ObjectIDLength
}

func lowerHex(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			if char < 'a' || char > 'f' {
				return false
			}
		}
	}
	return true
}

func (c *metadataCopier) skipRegularFile(path string) error {
	info, err := c.source.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect unused Git metadata: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxGitMetadataFileBytes || linkedFile(info) {
		return errGitMetadataEntry
	}
	if err := c.countEntry(); err != nil {
		return err
	}
	c.files[path] = info
	return nil
}
