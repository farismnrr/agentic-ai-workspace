package runner

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"strconv"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// DBTunnel says which database a project's sandbox may reach: one host and port, or none. Whoever
// answers owns the rule; the runner only passes the answer on to the proxy and the start script.
type DBTunnel interface {
	Target(ctx context.Context, projectID string) (host string, port int, ok bool)
}

// WithDBTunnel lets this project's commands reach the one database its connection was made from.
func (r *Runner) WithDBTunnel(tunnel DBTunnel) *Runner {
	r.tunnel = tunnel

	return r
}

// egressHosts lists what this run's proxy may reach besides the registries: GitHub for an owner who
// connected it, and the project's database as an exact host:port.
func (r *Runner) egressHosts(ctx context.Context, hasGit bool) (hosts []string, tunnel string) {
	if hasGit {
		hosts = append(hosts, githubHosts...)
	}

	if tunnel = TunnelTarget(ctx, r.tunnel, r.projectID); tunnel != "" {
		hosts = append(hosts, tunnel)
	}

	return hosts, tunnel
}

// TunnelTarget returns "host:port" for the project's database, or "" when there is none.
func TunnelTarget(ctx context.Context, tunnel DBTunnel, projectID string) string {
	if tunnel == nil {
		return ""
	}

	host, port, ok := tunnel.Target(ctx, projectID)
	if !ok {
		return ""
	}

	return net.JoinHostPort(host, strconv.Itoa(port))
}

// UseTunnel opens the local door in spec for the database, but only when the proxy socket is there to
// carry it. It returns the project's values, with the database's address made local when its host is
// an IP address, which cannot be given a local name.
func UseTunnel(spec *sandbox.Spec, tunnel string, values map[string]string) map[string]string {
	if tunnel == "" || spec.Egress == "" {
		return values
	}

	spec.Tunnel = tunnel

	return localize(values, tunnel)
}

// localize points URLs at the database's IP address to this computer's loopback, where the start script listens.
func localize(values map[string]string, tunnel string) map[string]string {
	host, port, err := net.SplitHostPort(tunnel)
	if err != nil {
		return values
	}

	if _, err = netip.ParseAddr(host); err != nil {
		return values
	}

	out := make(map[string]string, len(values))

	for name, value := range values {
		out[name] = value

		if u, perr := url.Parse(value); perr == nil && u.Host != "" && u.Hostname() == host && u.Port() == port {
			u.Host = net.JoinHostPort("127.0.0.1", port)
			out[name] = u.String()
		}
	}

	return out
}
