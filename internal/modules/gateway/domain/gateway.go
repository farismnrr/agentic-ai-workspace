// Package domain holds the rules for letting a project's code reach the person's own model gateway.
package domain

import (
	"errors"
	"net/url"
	"strconv"
)

// Errors of the gateway use cases.
var (
	ErrNotFound   = errors.New("project not found")
	ErrIneligible = errors.New("that provider cannot be used by a project")
)

// Choice is a provider a project may be given: an OpenAI-compatible gateway on this computer.
type Choice struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

const (
	httpPort  = 80
	httpsPort = 443
	maxPort   = 65535
)

// Target returns the host and port of a gateway's base URL, and whether it is a gateway on this computer that the
// sandbox can be given. The tunnel keeps the name and the port, so the saved address works as it is inside.
func Target(baseURL string) (host string, port int, ok bool) {
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", 0, false
	}

	host = u.Hostname()
	if host != "localhost" && host != "127.0.0.1" {
		return "", 0, false
	}

	port, ok = portOf(u)

	return host, port, ok
}

// portOf is the URL's port, or the scheme's default.
func portOf(u *url.URL) (int, bool) {
	switch {
	case u.Port() != "":
		port, err := strconv.Atoi(u.Port())

		return port, err == nil && port > 0 && port <= maxPort
	case u.Scheme == "https":
		return httpsPort, true
	default:
		return httpPort, true
	}
}

// Environment is what the project's code is given to use the gateway: the address as saved, and the key when the
// gateway has one. The names are what OpenAI-compatible libraries read.
func Environment(baseURL, apiKey string) []string {
	env := []string{"OPENAI_BASE_URL=" + baseURL}
	if apiKey != "" {
		env = append(env, "OPENAI_API_KEY="+apiKey)
	}

	return env
}
