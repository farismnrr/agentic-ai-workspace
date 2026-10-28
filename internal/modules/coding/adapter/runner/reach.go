package runner

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"slices"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// ProgramHosts combines a vendor program's registries with project grants and owner-connected GitHub access.
func ProgramHosts(own []string, granted Granted, hasGit bool) []string {
	hosts := append(slices.Clone(own), granted.Targets...)
	if hasGit {
		hosts = append(hosts, githubHosts...)
	}
	return hosts
}

// SamePort reports whether a granted target already uses target's port.
func SamePort(targets []string, target string) bool {
	_, port, _ := net.SplitHostPort(target)
	for _, existing := range targets {
		if _, other, _ := net.SplitHostPort(existing); other == port {
			return true
		}
	}
	return false
}

// Granted is what the server lets one project's sandbox reach besides the registries: exact "host:port" targets,
// such as the project's database and its model gateway, and the environment that points the code at them.
type Granted struct {
	Targets []string
	Env     []string
}

// Reach decides what a project's sandbox may reach. Whoever answers owns the rules; the runner only passes the
// answer on to the proxy and the start script.
type Reach interface {
	For(ctx context.Context, projectID string) Granted
}

// WithReach lets this project's commands reach what the server granted it.
func (r *Runner) WithReach(reach Reach) *Runner {
	r.reach = reach

	return r
}

// GrantedFor asks reach, which may be nil, what the project may reach.
func GrantedFor(ctx context.Context, reach Reach, projectID string) Granted {
	if reach == nil {
		return Granted{}
	}

	return reach.For(ctx, projectID)
}

// egressHosts lists what this run's proxy may reach besides the registries: GitHub for an owner who connected it,
// and everything the project was granted.
func (r *Runner) egressHosts(ctx context.Context, hasGit bool) (hosts []string, granted Granted) {
	if hasGit {
		hosts = append(hosts, githubHosts...)
	}

	granted = GrantedFor(ctx, r.reach, r.projectID)

	return append(hosts, granted.Targets...), granted
}

// Open sets the targets in spec, but only when the proxy socket is there to carry them.
func (g Granted) Open(spec *sandbox.Spec) {
	if spec.Egress != "" {
		spec.Tunnels = g.Targets
	}
}

// Apply opens the local doors in spec and returns the project's values, with the address of a target given as an
// IP made local (an IP cannot be given a local name), and the environment to put in the base before the project's
// own values, so a variable the server sets wins over one the person typed.
func (g Granted) Apply(spec *sandbox.Spec, values map[string]string) (map[string]string, []string) {
	g.Open(spec)

	if spec.Egress == "" {
		return values, nil
	}

	for _, target := range g.Targets {
		values = localize(values, target)
	}

	return values, g.Env
}

// localize points URLs at a target's IP address to this computer's loopback, where the start script listens.
func localize(values map[string]string, target string) map[string]string {
	host, port, err := net.SplitHostPort(target)
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
