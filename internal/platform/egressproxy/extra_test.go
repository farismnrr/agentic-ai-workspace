package egressproxy

import (
	"context"
	"net"
)

func fakeDial(proxy *Proxy) *Proxy {
	proxy.dial = func(context.Context, string) (net.Conn, error) {
		client, server := net.Pipe()

		go func() { _ = server.Close() }()

		return client, nil
	}

	return proxy
}
