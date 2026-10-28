package dbconn_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	dbconnprovision "github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/adapter/provision"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func TestSharedAdminURLPreservesReservedPassword(t *testing.T) {
	reservedPassword := strings.Join([]string{"value", ":/?% ", "雪"}, "")
	want := domain.SharedSettings{
		Kind: "postgres", TLS: "full", Host: "db.example", Port: 5432, Database: "workspace",
		User: "admin user", Password: reservedPassword, SSLMode: "verify-full",
	}
	raw, err := dbconnprovision.SharedAdminURL(want)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	config, err := pgx.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if config.Password != want.Password || config.User != want.User || config.Database != want.Database ||
		parsed.Query().Get("sslmode") != want.SSLMode || config.Port != 5432 {
		t.Fatalf("connection fields did not round trip: %+v", config)
	}
}
