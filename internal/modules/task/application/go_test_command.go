package application

import "strings"

const goTestArgumentStart = 2

// GoTestExecutionEvidenceAllowed accepts literal go test commands that run tests
// directly and fail closed when shell syntax or a replacement executor appears.
func GoTestExecutionEvidenceAllowed(command string) bool {
	words, ok := literalCommandWords(command)
	return ok && isGoTestWords(words) && goTestFlagsAllowExecution(words)
}

func isGoTestWords(words []string) bool {
	return len(words) >= 2 && words[0] == "go" && words[1] == "test"
}

func goTestFlagsAllowExecution(words []string) bool {
	for index, word := range words[goTestArgumentStart:] {
		if noExecutionFlag(word) || isReplacementExecutor(word) || zeroCount(words, index+goTestArgumentStart) {
			return false
		}
	}
	return true
}

func isReplacementExecutor(word string) bool {
	flag := strings.TrimLeft(word, "-")
	return flag == "exec" || strings.HasPrefix(flag, "exec=")
}

func zeroCount(words []string, index int) bool {
	if words[index] != "-count" && words[index] != "-test.count" {
		return false
	}
	return index+1 < len(words) && words[index+1] == "0"
}

func literalCommandWords(command string) ([]string, bool) {
	lexer := commandLexer{}
	if !lexer.scan(command) {
		return nil, false
	}
	return lexer.words, true
}

type commandLexer struct {
	words   []string
	word    strings.Builder
	quote   rune
	escaped bool
	started bool
}

func (l *commandLexer) scan(command string) bool {
	for _, char := range command {
		if !l.consume(char) {
			return false
		}
	}
	if l.escaped || l.quote != 0 {
		return false
	}
	l.flush()
	return true
}

func (l *commandLexer) consume(char rune) bool {
	if l.escaped {
		return l.consumeEscaped(char)
	}
	if l.quote != 0 {
		return l.consumeQuoted(char)
	}
	return l.consumeUnquoted(char)
}

func (l *commandLexer) consumeEscaped(char rune) bool {
	if char == '\n' || char == '\r' {
		return false
	}
	l.word.WriteRune(char)
	l.escaped, l.started = false, true
	return true
}

func (l *commandLexer) consumeQuoted(char rune) bool {
	if char == l.quote {
		l.quote = 0
		return true
	}
	if l.quote == '"' && strings.ContainsRune("$`()", char) {
		return false
	}
	l.word.WriteRune(char)
	return true
}

func (l *commandLexer) consumeUnquoted(char rune) bool {
	switch {
	case char == '\\':
		l.escaped, l.started = true, true
	case char == '\'' || char == '"':
		l.quote, l.started = char, true
	case strings.ContainsRune(";&|<>$`(){}", char):
		return false
	case isCommandSpace(char):
		l.flush()
	default:
		l.word.WriteRune(char)
		l.started = true
	}
	return true
}

func isCommandSpace(char rune) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

func (l *commandLexer) flush() {
	if !l.started {
		return
	}
	l.words = append(l.words, l.word.String())
	l.word.Reset()
	l.started = false
}

func noExecutionFlag(word string) bool {
	return word == "-list" || strings.HasPrefix(word, "-list=") || word == "-n" || word == "-c" ||
		word == "-count=0" || word == "-test.count=0" || word == "-test.list" ||
		strings.HasPrefix(word, "-test.list=")
}
