package egressproxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const (
	maxAnswers = 16
	directions = 2
)

var (
	errNonPublic = errors.New("the name resolves to a non-public address")
	errNoAnswer  = errors.New("the name did not resolve")
)

// taskPath is what a proxy serving one task's socket adds to the older rules. A proxy with a nil Grants (every
// proxy that Hub.Dir makes) never reaches it, so the older paths behave exactly as before.
type taskPath struct {
	grants *Grants
	// lookupIP and dialIP are the network, replaceable in tests. dialIP always gets an address that was already
	// checked, so the name is resolved once and the checked address is the one connected to.
	lookupIP func(ctx context.Context, host string) ([]netip.Addr, error)
	dialIP   func(ctx context.Context, ip netip.Addr) (net.Conn, error)
	scope    Scope
}

func defaultTaskPath() taskPath {
	dialer := &net.Dialer{
		Timeout: dialTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			return egress.Check(egress.Policy{Rule: egress.PublicOnly}, address)
		},
	}

	return taskPath{
		lookupIP: func(ctx context.Context, host string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		},
		dialIP: func(ctx context.Context, ip netip.Addr) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), httpsPort))
		},
	}
}

// serveGrant handles a CONNECT that no older rule allowed. It answers everything itself: a refusal is always 403
// and a tunnel is only opened for a live grant of this socket's own scope, on port 443, to a public address.
func (p *Proxy) serveGrant(w http.ResponseWriter, r *http.Request, rawHost, port string) {
	host, ok := egress.CanonicalHost(rawHost)
	if !ok || port != httpsPort {
		http.Error(w, blockedText, http.StatusForbidden)

		return
	}

	lease, err := p.task.grants.Lease(p.task.scope, host)
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, ErrTooMany) {
			status = http.StatusServiceUnavailable
		}

		http.Error(w, blockedText, status)

		return
	}

	defer lease.Close()

	if !p.verified(r.Context(), lease, host) {
		http.Error(w, blockedText, http.StatusForbidden)

		return
	}

	upstream, err := p.pinnedDial(r.Context(), host)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errNonPublic) {
			status = http.StatusForbidden
		}

		http.Error(w, blockedText, status)

		return
	}

	lease.Add(upstream)
	p.relay(w, lease, upstream, host)
}

func (p *Proxy) verified(ctx context.Context, lease *Lease, host string) bool {
	v := p.task.grants.verify

	return v == nil || v.Verify(ctx, lease.ID(), p.task.scope, host) == nil
}

// pinnedDial resolves host once, refuses it when any answer is not a public address, and connects to the first
// answer that works by its address, so a second lookup can never lead somewhere else.
func (p *Proxy) pinnedDial(ctx context.Context, host string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, p.task.grants.limits.Dial)
	defer cancel()

	answers, err := p.task.lookupIP(ctx, host)
	if err != nil || len(answers) == 0 || len(answers) > maxAnswers {
		return nil, errNoAnswer
	}

	for i, ip := range answers {
		answers[i] = ip.Unmap().WithZone("")
		if egress.Check(egress.Policy{Rule: egress.PublicOnly}, net.JoinHostPort(answers[i].String(), httpsPort)) != nil {
			return nil, errNonPublic
		}
	}

	for _, ip := range answers {
		if conn, err := p.task.dialIP(ctx, ip); err == nil {
			return conn, nil
		}
	}

	return nil, errNoAnswer
}

// relay answers the CONNECT, reads the first TLS record, and only forwards it when its server name is the granted
// host. After that the tunnel is opaque bytes until it ends, the grant ends, or it sits idle.
func (p *Proxy) relay(w http.ResponseWriter, lease *Lease, upstream net.Conn, host string) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "tunnels are not supported", http.StatusInternalServerError)

		return
	}

	client, rw, err := hijacker.Hijack()
	if err != nil {
		return
	}

	lease.Add(client)

	limits := p.task.grants.limits
	hard := lease.Ends()
	_ = client.SetDeadline(earliest(hard, time.Now().Add(limits.Handshake)))

	if _, err = io.WriteString(client, connectReply); err != nil {
		return
	}

	hello, name, err := readClientHello(rw.Reader)
	if err != nil || name != host {
		return
	}

	_ = client.SetDeadline(time.Time{})

	if _, err = upstream.Write(hello); err != nil {
		return
	}

	done := make(chan struct{}, directions)

	go func() { pipe(upstream, client, rw.Reader, limits.Idle, hard); done <- struct{}{} }()
	go func() { pipe(client, upstream, upstream, limits.Idle, hard); done <- struct{}{} }()

	<-done
	lease.Close()
	<-done
}
