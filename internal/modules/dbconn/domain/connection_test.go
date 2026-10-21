package domain

import (
	"net/url"
	"strings"
	"testing"
)

func TestConnectionURLEscapesReservedCredentials(t *testing.T) {
	connection := Connection{
		Kind: Postgres, Host: "db.example", Port: 5432, Database: "reporting",
		User: "person@work", Password: strings.Join([]string{"p:", "/?@", "#%&="}, ""), TLS: true,
	}
	parsed, err := url.Parse(connection.URL())
	if err != nil {
		t.Fatal(err)
	}
	user := parsed.User.Username()
	password, hasPassword := parsed.User.Password()
	if !hasPassword || user != connection.User || password != connection.Password {
		t.Fatalf("connection URL changed credentials: got %q and %q", user, password)
	}
	if parsed.Scheme != "postgres" || parsed.Query().Get("sslmode") != "require" {
		t.Fatalf("unexpected URL structure: %#v", parsed)
	}
}

func TestConnectionDefaultsMongoEnvAndRejectsReservedNames(t *testing.T) {
	connection := Connection{
		Kind: MongoDB, Host: "localhost", Port: 27017, Database: "app",
		User: "user", Password: "secret", TLS: true,
	}
	if got := DefaultEnvName(connection.Kind); got != "MONGODB_URL" {
		t.Fatalf("got %q", got)
	}
	connection.EnvName = "DOCKER_HOST"
	if err := connection.Validate(); err == nil {
		t.Fatal("reserved sandbox variable was accepted")
	}
	connection.EnvName = "GOFLAGS"
	if err := connection.Validate(); err == nil {
		t.Fatal("dangerous Go variable was accepted")
	}
	connection.EnvName = "GOOGLE_API_KEY"
	if err := connection.Validate(); err != nil {
		t.Fatalf("unrelated GO-prefixed variable was rejected: %v", err)
	}
}
