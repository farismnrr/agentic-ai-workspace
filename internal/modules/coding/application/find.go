package application

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// FindResult lists matching file paths, possibly cut short.
type FindResult struct {
	Paths     []string
	Truncated bool
}

// Find lists files whose path (relative to dir) matches a glob. "**" matches
// any number of folders, so "**/*.go" finds Go files at every depth.
func (s *Service) Find(pattern, raw string) (FindResult, error) {
	segments, err := globSegments(pattern)
	if err != nil {
		return FindResult{}, err
	}

	base, err := domain.CleanPath(raw)
	if err != nil {
		return FindResult{}, fmt.Errorf("find: %w", err)
	}

	var result FindResult

	err = s.files.Walk(base, func(entry domain.Entry) error { return collect(entry, base, segments, &result) })
	if err != nil && !isEnough(err) {
		return FindResult{}, fmt.Errorf("find %s: %w", base, err)
	}

	sort.Strings(result.Paths)

	return result, nil
}

func globSegments(pattern string) ([]string, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, domain.ErrBadPattern
	}

	segments := strings.Split(strings.Trim(pattern, "/"), "/")
	for _, segment := range segments {
		if _, err := path.Match(segment, ""); err != nil {
			return nil, domain.ErrBadPattern
		}
	}

	return segments, nil
}

func collect(entry domain.Entry, base string, segments []string, result *FindResult) error {
	if domain.Denied(entry.Path) || (entry.Dir && skippedDir(entry.Path)) {
		return skipFor(entry)
	}

	if entry.Dir || !globMatch(segments, relativeTo(base, entry.Path)) {
		return nil
	}

	if len(result.Paths) == domain.MaxFindResults {
		result.Truncated = true

		return errEnough
	}

	result.Paths = append(result.Paths, entry.Path)

	return nil
}

func relativeTo(base, full string) []string {
	if base != "." {
		full = strings.TrimPrefix(full, base+"/")
	}

	return strings.Split(full, "/")
}

// globMatch matches path segments against pattern segments, where "**" spans folders.
func globMatch(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}

	if pattern[0] == "**" {
		for skip := 0; skip <= len(name); skip++ {
			if globMatch(pattern[1:], name[skip:]) {
				return true
			}
		}

		return false
	}

	if len(name) == 0 {
		return false
	}

	ok, err := path.Match(pattern[0], name[0])

	return err == nil && ok && globMatch(pattern[1:], name[1:])
}
