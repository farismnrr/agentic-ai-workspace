package application

import (
	"regexp"
	"strings"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

// maxOutput stops a program that prints without end from growing the buffer.
const maxOutput = 16 << 10

var (
	escapes    = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	claudeURL  = regexp.MustCompile(`https://claude\.com/\S+`)
	openaiURL  = regexp.MustCompile(`https://auth\.openai\.com/\S+`)
	deviceCode = regexp.MustCompile(`\b[A-Z0-9]{4}-[A-Z0-9]{4,6}\b`)
)

// scanner reads what a program prints while signing in and picks out the address (and code) the
// person must use. Only addresses on the vendor's own sign-in host are ever kept.
type scanner struct {
	found  chan struct{}
	prompt domain.Prompt
	rt     domain.Runtime
	text   strings.Builder
	mu     sync.Mutex
	once   sync.Once
}

func newScanner(rt domain.Runtime) *scanner {
	return &scanner{rt: rt, found: make(chan struct{})}
}

// Write implements io.Writer.
func (s *scanner) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.text.Len() < maxOutput {
		s.text.Write(p)
	}

	s.look()

	return len(p), nil
}

// Prompt is what was found; it is valid once found is closed.
func (s *scanner) Prompt() domain.Prompt {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.prompt
}

// look checks the complete lines so far; a line still being printed could hold half an address.
func (s *scanner) look() {
	whole := s.text.String()
	if i := strings.LastIndex(whole, "\n"); i >= 0 {
		whole = whole[:i]
	} else {
		return
	}

	clean := escapes.ReplaceAllString(whole, "")
	if s.rt == domain.ChatGPT {
		s.prompt = domain.Prompt{URL: openaiURL.FindString(clean), Code: deviceCode.FindString(clean)}
	} else {
		s.prompt = domain.Prompt{URL: claudeURL.FindString(clean)}
	}

	if s.prompt.URL != "" && (s.rt != domain.ChatGPT || s.prompt.Code != "") {
		s.once.Do(func() { close(s.found) })
	}
}
