package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/secretbox"
)

func TestSharedSettingsPasswordIsSealedInSQLite(t *testing.T) {
	db, openErr := database.Open(t.TempDir())
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(func() { _ = db.Close() })
	cipher, cipherErr := secretbox.Open(t.TempDir())
	if cipherErr != nil {
		t.Fatal(cipherErr)
	}
	password := strings.Join([]string{"p@", "ss:/?% ", `"雪`}, "")
	sealed, encryptErr := cipher.Encrypt([]byte(password))
	if encryptErr != nil {
		t.Fatal(encryptErr)
	}
	store := NewSharedSettingsStore(db)
	want := application.StoredSharedSettings{
		Settings: domain.SharedSettings{
			Kind: "postgres", TLS: "none",
			Host: "db", Port: 5432, Database: "workspace", User: "admin", SSLMode: "prefer",
		},
		PasswordSealed: sealed,
	}
	if err := store.Save(context.Background(), "postgres", want); err != nil {
		t.Fatal(err)
	}
	got, getErr := store.Get(context.Background(), "postgres")
	if getErr != nil {
		t.Fatal(getErr)
	}
	var persisted string
	query := `SELECT password_sealed FROM shared_database_servers WHERE kind = 'postgres'`
	if queryErr := db.QueryRowContext(context.Background(), query).Scan(&persisted); queryErr != nil {
		t.Fatal(queryErr)
	}
	assertStoredSecretIsSealed(t, persisted, password, sealed)
	plain, decryptErr := cipher.Decrypt(got.PasswordSealed)
	assertDecryptedSecret(t, plain, decryptErr, password)
}

func assertStoredSecretIsSealed(t *testing.T, stored, plain, sealed string) {
	t.Helper()
	if stored == plain || stored != sealed {
		t.Fatal("SQLite did not retain the sealed password")
	}
}

func assertDecryptedSecret(t *testing.T, value []byte, err error, want string) {
	t.Helper()
	if err != nil || string(value) != want {
		t.Fatalf("sealed password round trip failed: %v", err)
	}
}
