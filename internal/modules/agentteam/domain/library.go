package domain

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
)

// libraryRoot holds what roles can look up while they work: craft rules, style families and page templates.
// A role sees the groups listed in its LIBRARY file.
const libraryRoot = "skills/library"

var referenceName = regexp.MustCompile(`^[a-z0-9-]+/[a-z0-9-]+$`)

// Reference is one item of the library a role may read, named group/name such as style/minimal.
type Reference struct {
	Name    string
	Summary string
}

// libraryGroups are the library groups a built-in role may read.
func libraryGroups(key string) []string {
	data, err := skillFiles.ReadFile("skills/" + key + "/LIBRARY")
	if err != nil {
		return nil
	}

	return strings.Fields(string(data))
}

// LibraryFor lists what a built-in role can look up, from the INDEX.txt of each of its groups.
func LibraryFor(key string) []Reference {
	var refs []Reference

	for _, group := range libraryGroups(key) {
		data, err := skillFiles.ReadFile(libraryRoot + "/" + group + "/INDEX.txt")
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(data), "\n") {
			if name, summary, ok := strings.Cut(line, ": "); ok {
				refs = append(refs, Reference{Name: group + "/" + name, Summary: summary})
			}
		}
	}

	return refs
}

// ReferenceText is the text of a library item a built-in role may read: a single file, or for a folder its
// markdown, CSS and example files one after another.
func ReferenceText(key, name string) (string, bool) {
	group, item, found := strings.Cut(name, "/")
	if !found || !referenceName.MatchString(name) || !contains(libraryGroups(key), group) {
		return "", false
	}

	base := path.Join(libraryRoot, group, item)
	if data, err := skillFiles.ReadFile(base + ".md"); err == nil {
		return strings.TrimSpace(string(data)), true
	}

	var out strings.Builder

	_ = fs.WalkDir(skillFiles, base, func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !readable(strings.TrimPrefix(file, base+"/")) {
			return nil //nolint:nilerr // an unreadable file is left out of the text
		}

		if data, readErr := skillFiles.ReadFile(file); readErr == nil {
			out.WriteString("### " + strings.TrimPrefix(file, base+"/") + "\n\n" + strings.TrimSpace(string(data)) + "\n\n")
		}

		return nil
	})

	return strings.TrimSpace(out.String()), out.Len() > 0
}

// readable is whether a file of a library folder is part of its text: prose, styles and the example page, not the
// seed templates or licences.
func readable(file string) bool {
	if strings.HasPrefix(file, "assets/") {
		return false
	}

	switch path.Ext(file) {
	case ".md", ".css", ".html":
		return true
	}

	return false
}

func contains(list []string, item string) bool {
	for _, entry := range list {
		if entry == item {
			return true
		}
	}

	return false
}
