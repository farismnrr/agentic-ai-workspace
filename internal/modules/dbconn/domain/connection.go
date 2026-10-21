package domain

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Kind string

const (
	maxDatabaseName = 128
	maxUserName     = 256
	maxPassword     = 1024
	maxEnvName      = 64

	Postgres Kind = "postgres"
	MySQL    Kind = "mysql"
	MongoDB  Kind = "mongodb"
)

var (
	ErrInvalid = errors.New("invalid database connection")
	ErrMissing = errors.New("database connection not found")
)

type FailureCategory string

const (
	HostNotFound FailureCategory = "host_not_found"
	Refused      FailureCategory = "refused"
	TimedOut     FailureCategory = "timeout"
	AuthFailed   FailureCategory = "auth_failed"
	TLSFailed    FailureCategory = "tls_failed"
	Blocked      FailureCategory = "blocked_address"
	Other        FailureCategory = "other"
)

type ProbeError struct{ Category FailureCategory }

func (e ProbeError) Error() string { return string(e.Category) }

// Connection contains the user's connection settings, including its secret password.
type Connection struct {
	Kind     Kind
	Host     string
	Database string
	User     string
	Password string
	EnvName  string
	Port     int
	TLS      bool
}

// Metadata is the safe subset returned by read APIs.
type Metadata struct {
	Kind       Kind   `json:"type"`
	Host       string `json:"host"`
	Database   string `json:"database"`
	User       string `json:"user"`
	EnvName    string `json:"envName"`
	Port       int    `json:"port"`
	TLS        bool   `json:"tls"`
	Configured bool   `json:"configured"`
}

func (c Connection) Validate() error {
	if !validKind(c.Kind) || !validAddress(c) || !validCredentials(c) || !validConnectionEnv(c) {
		return ErrInvalid
	}
	return nil
}

func validKind(kind Kind) bool { return kind == Postgres || kind == MySQL || kind == MongoDB }

func validAddress(c Connection) bool {
	return strings.TrimSpace(c.Host) != "" && len(c.Host) <= 253 &&
		!strings.ContainsAny(c.Host, " /\\[]\t\r\n@?#,") &&
		(!strings.Contains(c.Host, ":") || net.ParseIP(c.Host) != nil) && c.Port > 0 && c.Port <= 65535
}

func validCredentials(c Connection) bool {
	return c.Database != "" && len(c.Database) <= maxDatabaseName && !strings.ContainsAny(c.Database, "/?#") &&
		c.User != "" && len(c.User) <= maxUserName && len(c.Password) <= maxPassword
}

func validConnectionEnv(c Connection) bool {
	name := c.EnvName
	if name == "" {
		name = DefaultEnvName(c.Kind)
	}
	return validEnvName(name)
}

func DefaultEnvName(kind Kind) string {
	if kind == MongoDB {
		return "MONGODB_URL"
	}
	return "DATABASE_URL"
}

func (c Connection) URL() string {
	scheme := string(c.Kind)
	if c.Kind == Postgres {
		scheme = "postgres"
	}
	query := connectionQuery(c.Kind, c.TLS)
	path := "/" + c.Database
	result := url.URL{
		Scheme: scheme, User: url.UserPassword(c.User, c.Password),
		Host: net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), Path: path, RawPath: "/" + url.PathEscape(c.Database),
	}
	result.RawQuery = query.Encode()
	return result.String()
}

func connectionQuery(kind Kind, tls bool) url.Values {
	query := url.Values{}
	if kind == Postgres {
		mode := "disable"
		if tls {
			mode = "require"
		}
		query.Set("sslmode", mode)
	} else {
		query.Set("tls", strconv.FormatBool(tls))
	}
	return query
}

func (c Connection) Meta() Metadata {
	return Metadata{
		Kind: c.Kind, Host: c.Host, Port: c.Port, Database: c.Database,
		User: c.User, TLS: c.TLS, EnvName: c.EnvName, Configured: true,
	}
}

func validEnvName(name string) bool {
	if len(name) == 0 || len(name) > maxEnvName || !validFirstNameByte(name[0]) {
		return false
	}
	for _, r := range name[1:] {
		if !validNextNameRune(r) {
			return false
		}
	}
	return !reservedEnvName(name)
}

func validFirstNameByte(char byte) bool { return char == '_' || char >= 'A' && char <= 'Z' }

func validNextNameRune(char rune) bool {
	return char == '_' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9'
}

func reservedEnvName(name string) bool {
	return dangerousGoName(name) || strings.HasPrefix(name, "LD_") ||
		strings.HasPrefix(name, "DOCKER_") || name == "PATH" || name == "HOME" ||
		name == "HTTP_PROXY" || name == "HTTPS_PROXY" || name == "NO_PROXY" || name == "ALL_PROXY"
}

func dangerousGoName(name string) bool {
	if strings.HasPrefix(name, "CGO_") {
		return true
	}
	switch name {
	case "GOFLAGS", "GOPATH", "GOROOT", "GOPROXY", "GOENV", "GOMODCACHE", "GOCACHE",
		"GOTOOLCHAIN", "GONOSUMDB", "GONOPROXY", "GOPRIVATE", "GOINSECURE", "GOSUMDB",
		"GOWORK", "GOEXPERIMENT", "GODEBUG", "GOOS", "GOARCH":
		return true
	default:
		return false
	}
}

func (c Connection) Address() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.Port)) }
