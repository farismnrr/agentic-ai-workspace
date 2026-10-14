package application

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// ErrSkipDir tells a Walk to skip the directory it is visiting.
var ErrSkipDir = errors.New("skip directory")

var errEnough = errors.New("enough matches")

// SearchResult holds matching lines, possibly cut short.
type SearchResult struct {
	Matches   []domain.Match
	Truncated bool
}

// Search finds lines containing query (a literal, never a regex, so it cannot
// be made to run away) in text files under a directory.
func (s *Service) Search(query, raw string, caseSensitive bool) (SearchResult, error) {
	if query == "" || utf8.RuneCountInString(query) > domain.MaxQueryLength {
		return SearchResult{}, domain.ErrEmptyQuery
	}

	rel, err := domain.CleanPath(raw)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search: %w", err)
	}

	var result SearchResult

	find := newMatcher(query, caseSensitive)

	err = s.files.Walk(rel, func(entry domain.Entry) error { return s.visit(entry, find, &result) })
	if err != nil && !errors.Is(err, errEnough) {
		return SearchResult{}, fmt.Errorf("search %s: %w", rel, err)
	}

	return result, nil
}

func (s *Service) visit(entry domain.Entry, find matcher, result *SearchResult) error {
	if domain.Denied(entry.Path) || (entry.Dir && skippedDir(entry.Path)) {
		return skipFor(entry)
	}

	if entry.Dir || entry.Size > domain.MaxSearchFileBytes {
		return nil
	}

	return s.scan(entry.Path, find, result)
}

type matcher struct {
	needle string
	fold   bool
}

func newMatcher(query string, caseSensitive bool) matcher {
	if caseSensitive {
		return matcher{needle: query}
	}

	return matcher{needle: strings.ToLower(query), fold: true}
}

func (m matcher) matches(line string) bool {
	if m.fold {
		line = strings.ToLower(line)
	}

	return strings.Contains(line, m.needle)
}

func isEnough(err error) bool {
	return errors.Is(err, errEnough)
}

func skipFor(entry domain.Entry) error {
	if entry.Dir {
		return ErrSkipDir
	}

	return nil
}

func skippedDir(rel string) bool {
	name := rel[strings.LastIndex(rel, "/")+1:]

	return name == "node_modules" || name == "vendor"
}

func (s *Service) scan(rel string, find matcher, result *SearchResult) error {
	data, _, err := s.files.ReadFile(rel, domain.MaxSearchFileBytes)
	if err != nil || isBinary(data) {
		return nil //nolint:nilerr // unreadable files are skipped, not fatal
	}

	for number, line := range strings.Split(string(data), "\n") {
		if !find.matches(line) {
			continue
		}

		if len(result.Matches) == domain.MaxSearchMatches {
			result.Truncated = true

			return errEnough
		}

		result.Matches = append(result.Matches, domain.Match{Path: rel, Line: number + 1, Text: clip(line)})
	}

	return nil
}

func clip(line string) string {
	line = strings.TrimRight(line, "\r")
	if len(line) <= domain.MaxMatchLineLength {
		return strings.ToValidUTF8(line, "�")
	}

	return strings.ToValidUTF8(line[:domain.MaxMatchLineLength], "") + "…"
}
