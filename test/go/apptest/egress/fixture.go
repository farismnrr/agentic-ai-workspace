package egress

import (
	"bufio"
	"context"
	"database/sql"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	egressruntime "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/runtime"
	egressdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

var (
	Agent = egressdomain.Actor{Kind: egressdomain.ActorAgent, ID: "agent:t1"}
	Owner = egressdomain.Decider{ID: "u1", ProjectID: "p1", Owner: true}
)

func DB(t *testing.T) *sql.DB {
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

func Hub(t *testing.T) *egressproxy.Hub {
	t.Helper()
	base, err := os.MkdirTemp("", "e")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	hub := egressproxy.NewHub(base, egressproxy.New([]string{"pypi.org"}))
	t.Cleanup(hub.Close)
	return hub
}

func Start(t *testing.T, db *sql.DB, hub *egressproxy.Hub, dataDir string) *egressruntime.Runtime {
	t.Helper()
	rt, err := egressruntime.Start(db, hub, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Close)
	return rt
}

func Approve(t *testing.T, rt *egressruntime.Runtime, task, host string) egressdomain.Grant {
	t.Helper()
	grant, err := rt.Service().Request(context.Background(), "p1", task, host, "need it", Agent)
	if err != nil {
		t.Fatal(err)
	}
	grant, err = rt.Service().Approve(context.Background(), grant.ID, Owner, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return grant
}

func ConnectVia(t *testing.T, socket, target string) int {
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
