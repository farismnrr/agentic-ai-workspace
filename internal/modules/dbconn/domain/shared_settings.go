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

var SSLModes = [...]string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

// SharedSettings contains the admin connection used to provision project schemas.
type SharedSettings struct {
	Host, Database, User, Password, SSLMode string
	CACert, ClientCert, ClientKey           string
	Port                                    int
}

func (s SharedSettings) Validate() error {
	if s.Host == "" || s.Database == "" || s.User == "" || s.Port < 1 || s.Port > 65535 {
		return ErrInvalidSharedSettings
	}
	for _, mode := range SSLModes {
		if s.SSLMode == mode {
			return nil
		}
	}
	return ErrInvalidSharedSettings
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
