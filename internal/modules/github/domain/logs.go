package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrLogsDenied means GitHub would not give the logs: the app may lack the Actions read permission, or the
	// repository was not shared with it.
	ErrLogsDenied = errors.New("github did not allow reading the logs")
	// ErrLogsGone means there are no such logs: a wrong number, or logs GitHub already deleted.
	ErrLogsGone = errors.New("those logs do not exist or have expired")
	// ErrBadLogRef means the request names neither a run nor a job, or a repository that cannot exist.
	ErrBadLogRef = errors.New("give the repository as owner/name and either a run number or a job number")
)

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,99}/[A-Za-z0-9._-]{1,100}$`)

// LogRef names the logs wanted: the failed jobs of a run, or one job.
type LogRef struct {
	Repo  string
	RunID int64
	JobID int64
}

// Valid reports whether the reference can only name a real repository and exactly one run or job.
func (r LogRef) Valid() bool {
	return repoPattern.MatchString(r.Repo) && (r.RunID > 0) != (r.JobID > 0)
}

// JobLog is the log of one job of a workflow run.
type JobLog struct {
	Name       string
	Conclusion string
	Text       string
}

// Tail keeps the last lines of a log, and no more than maxBytes of them. A failed step ends a log, so the end is
// where the reason is.
func Tail(text string, lines, maxBytes int) string {
	all := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}

	out := strings.Join(all, "\n")
	if len(out) > maxBytes {
		out = out[len(out)-maxBytes:]
	}

	return out
}
