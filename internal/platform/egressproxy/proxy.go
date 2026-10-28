// Package egressproxy lets sandboxed commands reach package registries and nothing else.
// A sandbox has no network of its own; it talks to this proxy through a Unix socket,
// so the server still listens on one TCP port only. The proxy tunnels HTTPS (CONNECT on
// port 443) to an allowlist of hostnames, and the address each name resolves to is
// checked when the connection opens, so the host, the LAN and loopback stay unreachable.
// The one exception is an exact host:port target, which a run is given for the project's
// database; only that address and port are reachable, and only through that run's socket.
package egressproxy

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
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

// Option configures a proxy's network dependencies.
type Option func(*Proxy)

// WithLookup replaces DNS lookup for dynamically granted hosts.
func WithLookup(lookup func(context.Context, string) ([]netip.Addr, error)) Option {
	return func(proxy *Proxy) { proxy.task.lookupIP = lookup }
}

// WithDialIP replaces dialing the already-validated address of a dynamically granted host.
func WithDialIP(dial func(context.Context, netip.Addr) (net.Conn, error)) Option {
	return func(proxy *Proxy) { proxy.task.dialIP = dial }
}

// WithDial replaces dialing for registry hosts.
func WithDial(dial func(context.Context, string) (net.Conn, error)) Option {
	return func(proxy *Proxy) { proxy.dial = dial }
}

// Proxy tunnels HTTPS connections to allowed hostnames, and any TCP connection to an exact target.
type Proxy struct {
	hosts   map[string]struct{}
	targets map[string]struct{}
	dial    dialFunc
	// dialTarget connects to an exact target, which may be on the LAN or this computer.
	dialTarget dialFunc
	// task is set only on the proxy of a task socket (Hub.TaskDir); see grantpath.go.
	task taskPath
}

// New allows the given hostnames. Names resolving to non-public addresses are still refused.
func New(hosts []string, opts ...Option) *Proxy {
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

	dial := func(ctx context.Context, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", address)
	}

	proxy := &Proxy{hosts: allowed, dial: dial, dialTarget: targetDial(""), task: defaultTaskPath()}
	for _, opt := range opts {
		opt(proxy)
	}

	return proxy
}

// Serve answers connections on l until it is closed.
func (p *Proxy) Serve(l net.Listener) error {
	server := &http.Server{Handler: p, ReadHeaderTimeout: headerWait}

	return server.Serve(l) //nolint:wrapcheck // the caller only needs to know that serving ended
}

// ServeHTTP handles one CONNECT request. The older rules (registry list, exact targets) are tried first and are
// unchanged; only a proxy made for a task socket goes on to consult dynamic grants.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, port, err := net.SplitHostPort(r.Host)
	if r.Method != http.MethodConnect || err != nil {
		http.Error(w, blockedText, http.StatusForbidden)

		return
	}

	target := p.isTarget(host, port)

	switch {
	case target || (port == httpsPort && p.allowed(host)):
		p.tunnelTo(w, r, host, port, target)
	case p.task.grants != nil:
		p.serveGrant(w, r, host, port)
	default:
		// The reply only reaches the tool inside the sandbox, which often shows a bare 403, so the server
		// records which host was refused: this is how a missing allowlist entry is found.
		slog.Warn("egress refused a host", "host", host, "port", port)
		http.Error(w, blockedText, http.StatusForbidden)
	}
}

func (p *Proxy) tunnelTo(w http.ResponseWriter, r *http.Request, host, port string, target bool) {
	dial := p.dial
	if target {
		dial = p.dialTarget
	}

	upstream, err := dial(r.Context(), net.JoinHostPort(host, port))
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
