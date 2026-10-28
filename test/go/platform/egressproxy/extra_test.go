package egressproxy_test

import (
	"context"
	"net"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

func fakeDial(hosts []string, opts ...egressproxy.Option) *egressproxy.Proxy {
	opts = append(opts, egressproxy.WithDial(func(context.Context, string) (net.Conn, error) {
		client, server := net.Pipe()

		go func() { _ = server.Close() }()

		return client, nil
	}))

	return egressproxy.New(hosts, opts...)
}
