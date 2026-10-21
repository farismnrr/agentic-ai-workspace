package sandbox

import (
	"net"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// TunnelEnv names the variable the start script reads to open the local door to the database.
const TunnelEnv = "DB_TUNNEL"

// hostName is what may stand in a tunnel target: it ends up in a start script argument, so nothing
// that could add an option, such as a comma or a colon, is allowed.
var hostName = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]{0,251}[a-z0-9])?$`)

// tunnelTarget splits Spec.Tunnel ("host:port") and reports whether it is safe to use.
func tunnelTarget(spec Spec) (host, port string, ok bool) {
	host, port, err := net.SplitHostPort(strings.ToLower(spec.Tunnel))
	if err != nil || !hostName.MatchString(host) {
		return "", "", false
	}

	number, err := strconv.Atoi(port)

	return host, port, err == nil && number > 0 && number < 65536
}

// containerEnv is the command's environment plus, when the server grants a database, the target for the
// start script to open. The container still has no network: the script listens on this computer's loopback
// and passes bytes to the server's proxy socket. A value of the same name set from outside is dropped, so
// only the server decides.
func containerEnv(spec Spec) []string {
	env := slices.DeleteFunc(slices.Clone(spec.Env), func(entry string) bool {
		return strings.HasPrefix(entry, TunnelEnv+"=")
	})

	if host, port, ok := tunnelTarget(spec); ok {
		env = append(env, TunnelEnv+"="+net.JoinHostPort(host, port))
	}

	return env
}

// tunnelHosts lets the database's own name resolve to the local door, so the saved address keeps
// working as it is (and a certificate still matches the name). An IP address needs no entry. Even
// "localhost" gets one, because the container only has a hosts file when it is given an entry.
func tunnelHosts(spec Spec) []string {
	host, _, ok := tunnelTarget(spec)
	if !ok {
		return nil
	}

	if _, err := netip.ParseAddr(host); err == nil {
		return nil
	}

	return []string{host + ":127.0.0.1"}
}
