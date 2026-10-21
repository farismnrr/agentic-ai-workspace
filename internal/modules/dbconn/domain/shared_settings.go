package domain

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
)

var (
	ErrInvalidSharedSettings = errors.New("invalid shared database settings")
	ErrInvalidPEM            = errors.New("invalid database certificate material")
)

const MaxPEMBytes = 16 * 1024

const (
	postgresPort = 5432
	mysqlPort    = 3306
	mongoPort    = 27017
)

var (
	SSLModes = [...]string{
		"disable", "allow", "prefer", "require", "verify-ca", "verify-full",
	}
	SharedKinds = [...]string{
		"postgres", "mysql", "mongodb",
	}
	TLSModes = [...]string{"none", "ca", "full"}
)

func IsSharedKind(kind string) bool { return validSharedKind(kind) }

func DefaultSharedPort(kind string) int {
	switch kind {
	case "postgres":
		return postgresPort
	case "mysql":
		return mysqlPort
	case "mongodb":
		return mongoPort
	default:
		return 0
	}
}

// SharedSettings contains the admin connection used to provision project schemas.
type SharedSettings struct {
	Kind, Host, Database, User, Password, SSLMode, TLS string
	CACert, ClientCert, ClientKey                      string
	Port                                               int
}

func (s SharedSettings) Validate() error {
	if !validSharedKind(s.Kind) || !validSharedAddress(s) || !validTLSMode(s.TLS) {
		return ErrInvalidSharedSettings
	}
	if s.Kind != "postgres" {
		return nil
	}
	return validatePostgresSSLMode(s.SSLMode)
}

func validSharedAddress(settings SharedSettings) bool {
	return settings.Host != "" && settings.Database != "" && settings.User != "" &&
		settings.Port >= 1 && settings.Port <= 65535
}

func validatePostgresSSLMode(sslMode string) error {
	for _, candidate := range SSLModes {
		if sslMode == candidate {
			return nil
		}
	}
	return ErrInvalidSharedSettings
}

func validSharedKind(kind string) bool {
	for _, candidate := range SharedKinds {
		if kind == candidate {
			return true
		}
	}
	return false
}

func validTLSMode(mode string) bool {
	for _, candidate := range TLSModes {
		if mode == candidate {
			return true
		}
	}
	return false
}

// ValidateCertificates checks the optional CA bundle and matching client pair.
func (s SharedSettings) ValidateCertificates() error {
	if !validPEMSizes(s) || !validOptionalCABundle(s.CACert) || !validClientPair(s.ClientCert, s.ClientKey) {
		return ErrInvalidPEM
	}
	return nil
}

func validPEMSizes(settings SharedSettings) bool {
	return len(settings.CACert) <= MaxPEMBytes && len(settings.ClientCert) <= MaxPEMBytes &&
		len(settings.ClientKey) <= MaxPEMBytes
}

func validOptionalCABundle(value string) bool {
	return value == "" || validCABundle(value)
}

func validClientPair(cert, key string) bool {
	if (cert == "") != (key == "") || encryptedPEMKey(key) {
		return false
	}
	if cert == "" {
		return true
	}
	_, err := tls.X509KeyPair([]byte(cert), []byte(key))
	return err == nil
}

func validCABundle(value string) bool {
	remaining := []byte(value)
	count := 0
	for len(remaining) > 0 {
		remaining = []byte(strings.TrimSpace(string(remaining)))
		if len(remaining) == 0 {
			break
		}
		if !strings.HasPrefix(string(remaining), "-----BEGIN ") {
			return false
		}
		block, rest := pem.Decode(remaining)
		if block == nil {
			return false
		}
		if block.Type != "CERTIFICATE" {
			return false
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return false
		}
		count++
		remaining = rest
	}
	pool := x509.NewCertPool()
	return count > 0 && pool.AppendCertsFromPEM([]byte(value))
}

func encryptedPEMKey(value string) bool {
	for rest := []byte(value); len(rest) > 0; {
		block, remaining := pem.Decode(rest)
		if block == nil {
			return false
		}
		if strings.Contains(block.Type, "ENCRYPTED") || block.Headers["Proc-Type"] == "4,ENCRYPTED" {
			return true
		}
		rest = remaining
	}
	return false
}
