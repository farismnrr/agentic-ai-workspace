package egressproxy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

// client is one connection through a task socket after its CONNECT was answered.
type client struct {
	net.Conn
	br     *bufio.Reader
	status int
}

func (r *rig) open(scope Scope, target string) *client {
	r.t.Helper()

	dir, err := r.hub.TaskDir(context.Background(), "ws", scope)
	if err != nil {
		r.t.Fatal(err)
	}

	return r.openAt(dir, target)
}

func (r *rig) openAt(dir, target string) *client {
	r.t.Helper()

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unix", filepath.Join(dir, socketName))
	if err != nil {
		r.t.Fatal(err)
	}

	r.t.Cleanup(func() { _ = conn.Close() })

	if _, err = io.WriteString(conn, "CONNECT "+target+" HTTP/1.1\r\nHost: "+target+"\r\n\r\n"); err != nil {
		r.t.Fatal(err)
	}

	br := bufio.NewReader(conn)

	res, err := http.ReadResponse(br, nil)
	if err != nil {
		r.t.Fatal(err)
	}

	_ = res.Body.Close()

	return &client{Conn: conn, br: br, status: res.StatusCode}
}

// closedWithin reports whether the far side closes the connection (rather than sending data or staying silent).
func (c *client) closedWithin(d time.Duration) bool {
	_ = c.SetReadDeadline(time.Now().Add(d))

	_, err := c.br.ReadByte()
	if err == nil {
		return false
	}

	var timeout net.Error

	return !errors.As(err, &timeout) || !timeout.Timeout()
}

// echoes sends data and reports whether the same bytes come back.
func (c *client) echoes(data []byte) bool {
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := c.Write(data); err != nil {
		return false
	}

	back := make([]byte, len(data))
	_, err := io.ReadFull(c.br, back)

	return err == nil && bytes.Equal(back, data)
}
