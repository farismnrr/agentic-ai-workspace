package application

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var ErrMissingChatCredentials = errors.New("the latest MoLab chat request has no valid URL and token")
var ErrInvalidMoLabURL = errors.New("MoLab URL must be HTTPS under sb-*.sb.molab.run")
var molabHost = regexp.MustCompile(`^sb-[a-z0-9-]+\.sb\.molab\.run$`)
var molabURL = regexp.MustCompile(`https://[^\s"'<>]+`)
var molabToken = regexp.MustCompile(`(?i)(?:--token\s+|(?:auth\s+)?token\s*(?:is|:|=)\s*["]?)([a-zA-Z0-9._~+/-]{8,512})`)

// Credentials are read exclusively from user chat requests in this task.
// A URL and token must occur in the SAME user message, never mixed across turns.
// No separate pairing, database, auto-approval, or persistent credential state.
type Credentials struct {
	URL   string
	Host  string
	Token string
}

func ValidateURL(raw string) (string, string, error) {
	if len(raw) > 512 {
		return "", "", ErrInvalidMoLabURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.RawQuery != "" ||
		u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Port() != "" && u.Port() != "443") {
		return "", "", ErrInvalidMoLabURL
	}
	host := strings.ToLower(u.Hostname())
	if !molabHost.MatchString(host) {
		return "", "", ErrInvalidMoLabURL
	}
	return "https://" + host + "/", host, nil
}

func FromChat(events []domain.Event) (Credentials, error) {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind != domain.EventRequest {
			continue
		}
		message := events[i].Text
		urlMatch := strings.TrimRight(molabURL.FindString(message), ".,;)]}"+string(rune(96)))
		tokenMatch := molabToken.FindStringSubmatch(message)
		if urlMatch == "" || len(tokenMatch) < 2 {
			continue
		}
		url, host, err := ValidateURL(urlMatch)
		if err != nil {
			return Credentials{}, err
		}
		token := tokenMatch[1]
		if len(token) > 512 || strings.ContainsAny(token, "\r\n\t ") {
			return Credentials{}, ErrMissingChatCredentials
		}
		return Credentials{URL: url, Host: host, Token: token}, nil
	}
	return Credentials{}, ErrMissingChatCredentials
}
