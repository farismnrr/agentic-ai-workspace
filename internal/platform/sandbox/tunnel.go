package sandbox

import (
	"net"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// TunnelEnv names the variable the start script reads to open a local door for each exact target.
const TunnelEnv = "TUNNEL_TARGETS"

// hostName is what may stand in a tunnel target: it ends up in a start script argument, so nothing
// that could add an option, such as a comma or a colon, is allowed.
var hostName = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]{0,251}[a-z0-9])?$`)

// tunnelTarget splits one "host:port" and reports whether it is safe to use.
func tunnelTarget(target string) (host, port string, ok bool) {
	host, port, err := net.SplitHostPort(strings.ToLower(target))
	if err != nil || !hostName.MatchString(host) {
		return "", "", false
	}

	number, err := strconv.Atoi(port)

	return host, port, err == nil && number > 0 && number < 65536
}

// tunnelTargets returns the usable targets of spec in order. A target that is not safe is dropped, and so is
// one that wants a port an earlier target already uses, because each target gets that port on loopback.
func tunnelTargets(spec Spec) []string {
	var targets, ports []string

	for _, target := range spec.Tunnels {
		host, port, ok := tunnelTarget(target)
		if !ok || slices.Contains(ports, port) {
			continue
		}

		ports = append(ports, port)
		targets = append(targets, net.JoinHostPort(host, port))
	}

	return targets
}

// containerEnv is the command's environment plus, when the server grants any, the targets for the start
// script to open. The container still has no network: the script listens on this computer's loopback and
// passes bytes to the server's proxy socket. A value of the same name set from outside is dropped, so only
// the server decides.
func containerEnv(spec Spec) []string {
	env := slices.DeleteFunc(slices.Clone(spec.Env), func(entry string) bool {
		return strings.HasPrefix(entry, TunnelEnv+"=")
	})

	if targets := tunnelTargets(spec); len(targets) > 0 {
		env = append(env, TunnelEnv+"="+strings.Join(targets, " "))
	}

	return env
}

// tunnelHosts lets each target's own name resolve to the local door, so a saved address keeps working as it
// is (and a certificate still matches the name). An IP address needs no entry. Even "localhost" gets one,
// because the container only has a hosts file when it is given an entry.
func tunnelHosts(spec Spec) []string {
	var hosts []string

	for _, target := range tunnelTargets(spec) {
		host, _, _ := tunnelTarget(target)
		if _, err := netip.ParseAddr(host); err == nil {
			continue
		}

		if entry := host + ":127.0.0.1"; !slices.Contains(hosts, entry) {
			hosts = append(hosts, entry)
		}
	}

	return hosts
}
