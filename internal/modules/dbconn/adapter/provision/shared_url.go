package provision

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

// SharedAdminURL builds an escaped PostgreSQL URL from a person's saved settings.
func SharedAdminURL(settings domain.SharedSettings) (string, error) {
	if err := settings.Validate(); err != nil {
		return "", fmt.Errorf("validate shared database settings: %w", err)
	}
	sslMode := "verify-full"
	if settings.TLS == "none" {
		sslMode = "disable"
	}
	connection := url.URL{
		Scheme: "postgres", User: url.UserPassword(settings.User, settings.Password),
		Host: net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port)), Path: settings.Database,
		RawQuery: url.Values{"sslmode": []string{sslMode}}.Encode(),
	}
	return connection.String(), nil
}
