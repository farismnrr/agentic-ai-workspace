// Package clistream reads what a vendor's coding program prints while it works and turns it into the steps
// of a task. It only reads output; it never touches the program's sign-in.
package clistream

import (
	"bytes"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// maxLine is the longest line kept; a program prints a whole file's text in one line now and then.
const maxLine = 4 << 20

// Limits on how much of a command or an error is shown in one line.
const (
	commandChars = 60
	errorChars   = 200
)

// workspace is where the project's folder appears inside the container.
const workspace = "/workspace/"

// Stream receives a program's output as a writer and reports the steps in it.
type Stream struct {
	emit    func(application.CLIEvent)
	parse   func(line []byte)
	session string
	summary string
	reason  string
	buf     bytes.Buffer
	used    int64
	window  int64
}

// New reads the output of one program: "claude" or "chatgpt".
func New(program string, emit func(application.CLIEvent)) *Stream {
	s := &Stream{emit: emit}
	if program == "chatgpt" {
		s.parse = newCodex(s).line
	} else {
		s.parse = newClaude(s).line
	}

	return s
}

// Write implements io.Writer: it splits the output into lines and reads each whole one.
func (s *Stream) Write(p []byte) (int, error) {
	s.buf.Write(p)

	for {
		i := bytes.IndexByte(s.buf.Bytes(), '\n')
		if i < 0 {
			break
		}

		line := bytes.TrimSpace(s.buf.Next(i + 1))
		if len(line) > 0 && len(line) <= maxLine {
			s.parse(line)
		}
	}

	if s.buf.Len() > maxLine {
		s.buf.Reset()
	}

	return len(p), nil
}

// Result is how the turn ended, once the program has exited.
func (s *Stream) Result() application.CLIResult {
	return application.CLIResult{Session: s.session, Summary: s.summary, Reason: s.reason, Used: s.used, Window: s.window}
}

// relative shows a file the way the project sees it.
func relative(path string) string {
	return strings.TrimPrefix(path, workspace)
}

// brief shortens text to a single readable line.
func brief(text string, limit int) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	if r := []rune(line); len(r) > limit {
		return string(r[:limit]) + "…"
	}

	return line
}
