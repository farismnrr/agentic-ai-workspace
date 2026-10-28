package application

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// inputDocumentForProject preserves a long request verbatim with a project-file reference.
func (s *Service) inputDocumentForProject(
	ctx context.Context, taskID, projectID, text string,
) (string, []string, error) {
	if s.deps.Notes == nil || utf8.RuneCountInString(text) <= documentThreshold {
		return text, nil, nil
	}
	id, err := s.deps.IDs.ID()
	if err != nil {
		return "", nil, fmt.Errorf("name input document: %w", err)
	}
	base := notesFolder + "tasks/" + taskID + "/input/" + id
	files := inputFiles(base, text)
	if err = s.deps.Notes.Write(ctx, projectID, files); err != nil {
		return "", nil, fmt.Errorf("save input document: %w", err)
	}
	if s.deps.Notify != nil {
		s.deps.Notify.FilesChanged(projectID)
	}
	const previewLength = 1200
	prompt := "The person's full message is saved verbatim in the project document `" + base + ".md`. " +
		"Read the entire document (in sections if needed) before answering, planning or delegating. " +
		"It is the current user request, not a summary.\n\nPreview:\n" + truncate(text, previewLength)
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return prompt, paths, nil
}

// inputFiles keeps an exact original plus parts that fit a read_file result even for a single very long line.
func inputFiles(base, text string) []NoteFile {
	files := []NoteFile{{Path: base + ".txt", Content: text}}
	var index strings.Builder
	index.WriteString("# Full user message\n\nRead every part below, in order. Together they contain the exact message.\n")
	runes := []rune(text)
	for start, part := 0, 1; start < len(runes); start, part = start+documentThreshold, part+1 {
		name := fmt.Sprintf("%s/part-%04d.txt", base, part)
		files = append(files, NoteFile{Path: name, Content: string(runes[start:min(start+documentThreshold, len(runes))])})
		fmt.Fprintf(&index, "\n- %s", name)
	}
	files = append(files, NoteFile{Path: base + ".md", Content: index.String()})
	return files
}
