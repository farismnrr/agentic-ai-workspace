// Package egressproxy lets sandboxed commands reach package registries and nothing else.
// A sandbox has no network of its own; it talks to this proxy through a Unix socket,
// so the server still listens on one TCP port only. The proxy tunnels HTTPS (CONNECT on
// port 443) to an allowlist of hostnames, and the address each name resolves to is
// checked when the connection opens, so the host, the LAN and loopback stay unreachable.
package egressproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const (
	httpsPort    = "443"
	dialTimeout  = 10 * time.Second
	headerWait   = 10 * time.Second
	blockedText  = "Egress policy blocked this HTTPS host or port. Check SANDBOX_EGRESS_HOSTS."
	connectReply = "HTTP/1.1 200 Connection Established\r\n\r\n"
)

type dialFunc func(ctx context.Context, address string) (net.Conn, error)

// Proxy tunnels HTTPS connections to allowed hostnames.
type Proxy struct {
	hosts map[string]struct{}
	dial  dialFunc
}

// New allows the given hostnames. Names resolving to non-public addresses are still refused.
func New(hosts []string) *Proxy {
	allowed := make(map[string]struct{}, len(hosts))

	for _, host := range hosts {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			allowed[host] = struct{}{}
		}
	}

	dialer := &net.Dialer{
		Timeout: dialTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			return egress.Check(egress.Policy{Rule: egress.PublicOnly}, address)
		},
	}

	return &Proxy{hosts: allowed, dial: func(ctx context.Context, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", address)
	}}
}

// WithHosts returns a proxy that allows the extra hostnames as well; the original is unchanged.
func (p *Proxy) WithHosts(extra []string) *Proxy {
	allowed := make(map[string]struct{}, len(p.hosts)+len(extra))
	for host := range p.hosts {
		allowed[host] = struct{}{}
	}

	for _, host := range extra {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			allowed[host] = struct{}{}
		}
	}

	return &Proxy{hosts: allowed, dial: p.dial}
}

// Serve answers connections on l until it is closed.
func (p *Proxy) Serve(l net.Listener) error {
	server := &http.Server{Handler: p, ReadHeaderTimeout: headerWait}

	return server.Serve(l) //nolint:wrapcheck // the caller only needs to know that serving ended
}

// ServeHTTP handles one CONNECT request.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, port, err := net.SplitHostPort(r.Host)

	if r.Method != http.MethodConnect || err != nil || port != httpsPort || !p.allowed(host) {
		http.Error(w, blockedText, http.StatusForbidden)

		return
	}

	upstream, err := p.dial(r.Context(), net.JoinHostPort(host, port))
	if err != nil {
		http.Error(w, "Could not reach "+host+".", http.StatusBadGateway)

		return
	}

	defer func() { _ = upstream.Close() }()

	tunnel(w, upstream)
}

func (p *Proxy) allowed(host string) bool {
	_, ok := p.hosts[strings.ToLower(host)]

	return ok
}

// tunnel copies bytes both ways until either side closes.
func tunnel(w http.ResponseWriter, upstream net.Conn) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "tunnels are not supported", http.StatusInternalServerError)

		return
	}

	client, _, err := hijacker.Hijack()
	if err != nil {
		return
	}

	defer func() { _ = client.Close() }()

	if _, err = io.WriteString(client, connectReply); err != nil {
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		_, _ = io.Copy(upstream, client)

		_ = upstream.Close()
	}()

	_, _ = io.Copy(client, upstream)

	_ = client.Close()

	wg.Wait()
}
