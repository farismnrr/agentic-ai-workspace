package bootstrap

import (
	"bufio"
	"context"
	"database/sql"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	egressdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

func egressDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	for _, q := range []string{
		`INSERT INTO users (id, name, email, password_hash, created_at) VALUES ('u1', 'A', 'a@example.com', 'x', 1)`,
		`INSERT INTO projects (id, user_id, name, slug, created_at) VALUES ('p1', 'u1', 'one', 'one', 1)`,
		`INSERT INTO tasks (id, project_id, user_id, title, status, mode, provider_id, model, created_at, updated_at)
			VALUES ('t1', 'p1', 'u1', 'x', 'working', 'risky', 'p', 'm', 1, 1),
			('t2', 'p1', 'u1', 'x', 'working', 'risky', 'p', 'm', 1, 1)`,
	} {
		if _, err = db.ExecContext(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}

	return db
}

func egressHub(t *testing.T) *egressproxy.Hub {
	t.Helper()

	// Unix socket paths are short; the test's own folder names are not.
	base, err := os.MkdirTemp("", "e")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(base) })

	hub := egressproxy.NewHub(base, egressproxy.New([]string{"pypi.org"}))
	t.Cleanup(hub.Close)

	return hub
}

func startEgress(t *testing.T, db *sql.DB, hub *egressproxy.Hub, dataDir string) *egressRuntime {
	t.Helper()

	rt, err := startEgressGrants(db, hub, dataDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(rt.stop)

	return rt
}

func approved(t *testing.T, rt *egressRuntime, task, host string) egressdomain.Grant {
	t.Helper()

	g, err := rt.service.Request(context.Background(), "p1", task, host, "need it", egressAgent)
	if err != nil {
		t.Fatal(err)
	}

	if g, err = rt.service.Approve(context.Background(), g.ID, egressOwner, time.Hour); err != nil {
		t.Fatal(err)
	}

	return g
}

func connectVia(t *testing.T, socket, target string) int {
	t.Helper()

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = conn.Close() }()

	_, _ = conn.Write([]byte("CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n\r\n"))

	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}

	_ = res.Body.Close()

	return res.StatusCode
}
