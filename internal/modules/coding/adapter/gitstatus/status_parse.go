package gitstatus

import (
	"bytes"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	statusRecordFields = 2
	ordinaryColumns    = 8
	renameColumns      = 9
	conflictColumns    = 10
	untrackedRecord    = "?"
)

func parse(data []byte) (codingdomain.GitSnapshot, error) {
	parts := bytes.Split(data, []byte{0})
	out := codingdomain.GitSnapshot{Available: true, Files: make([]codingdomain.GitFile, 0)}
	for i := 0; i < len(parts); i++ {
		if err := parseStatusPart(parts, &i, &out); err != nil {
			return codingdomain.GitSnapshot{}, err
		}
	}
	return out, nil
}

func parseStatusPart(parts [][]byte, index *int, snapshot *codingdomain.GitSnapshot) error {
	line := string(parts[*index])
	if line == "" {
		return nil
	}
	if strings.HasPrefix(line, "# ") {
		parseHeader(line, snapshot)
		return nil
	}
	file, rename, err := parseRecord(line)
	if err != nil {
		return err
	}
	if rename {
		if err := parseOriginalPath(parts, index, &file); err != nil {
			return err
		}
	}
	appendVisibleFile(snapshot, file)
	if len(snapshot.Files) > maxFiles {
		return errGitFileLimit
	}
	return nil
}

func parseOriginalPath(parts [][]byte, index *int, file *codingdomain.GitFile) error {
	*index++
	if *index >= len(parts) || len(parts[*index]) == 0 {
		return errGitRenameRecord
	}
	file.Original = string(parts[*index])
	return nil
}

func appendVisibleFile(snapshot *codingdomain.GitSnapshot, file codingdomain.GitFile) {
	if codingdomain.Denied(file.Path) || codingdomain.Denied(file.Original) {
		return
	}
	snapshot.Files = append(snapshot.Files, file)
}

func parseHeader(line string, out *codingdomain.GitSnapshot) {
	key, value, found := strings.Cut(strings.TrimPrefix(line, "# "), " ")
	if !found {
		return
	}
	switch key {
	case "branch.oid":
		if value != "(initial)" {
			out.Revision = value
		} else {
			out.Initial = true
		}
	case "branch.head":
		if value != "(detached)" {
			out.Branch = value
		}
	}
}

func parseRecord(line string) (codingdomain.GitFile, bool, error) {
	fields := strings.SplitN(line, " ", statusRecordFields)
	if len(fields) != statusRecordFields {
		return codingdomain.GitFile{}, false, errGitStatusRecord
	}
	if fields[0] == untrackedRecord {
		return codingdomain.GitFile{Path: strings.TrimPrefix(fields[1], " "), Untracked: true}, false, nil
	}
	columns, err := parseStatusColumns(fields[0], fields[1])
	if err != nil {
		return codingdomain.GitFile{}, false, err
	}
	file := gitFileFromColumns(fields[0], columns)
	return file, fields[0] == "2", nil
}

func parseStatusColumns(kind, record string) ([]string, error) {
	columnCount, valid := statusColumnCount(kind)
	if !valid {
		return nil, errGitStatusRecordKind
	}
	columns := strings.SplitN(record, " ", columnCount)
	if len(columns) < columnCount || len(columns[0]) != 2 {
		return nil, errGitStatusColumns
	}
	return columns, nil
}

func statusColumnCount(kind string) (int, bool) {
	switch kind {
	case "1":
		return ordinaryColumns, true
	case "2":
		return renameColumns, true
	case "u":
		return conflictColumns, true
	default:
		return 0, false
	}
}

func gitFileFromColumns(kind string, columns []string) codingdomain.GitFile {
	return codingdomain.GitFile{
		Path: columns[len(columns)-1], Index: statusCode(columns[0][0]), Worktree: statusCode(columns[0][1]),
		Conflict: kind == "u", Submodule: strings.HasPrefix(columns[1], "S"),
	}
}

func statusCode(value byte) string {
	if value == '.' || value == ' ' {
		return ""
	}
	return string(value)
}
