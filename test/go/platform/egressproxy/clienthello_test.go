package egressproxy_test

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"
)

// helloFor makes the bytes a real Go TLS client sends first, which is what sandboxes will send.
func helloFor(t *testing.T, serverName string) []byte {
	t.Helper()

	client, server := net.Pipe()

	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	go func() {
		cfg := &tls.Config{ServerName: serverName, InsecureSkipVerify: true} //nolint:gosec // only the first flight is wanted
		conn := tls.Client(client, cfg)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		defer cancel()

		_ = conn.HandshakeContext(ctx)
	}()

	_ = server.SetReadDeadline(time.Now().Add(time.Second))

	buf := make([]byte, 1<<15)

	n, err := server.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	return buf[:n]
}

// trailing adds a byte inside the record after the hello, with both lengths left saying otherwise fixed up.

// withLength sets the record length in the header, leaving the body as it is.
