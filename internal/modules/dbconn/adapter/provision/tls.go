package provision

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	errInvalidCABundle   = errors.New("invalid CA bundle")
	errMissingServerCert = errors.New("missing server certificate")
	errInvalidTLSMode    = errors.New("invalid database TLS mode")
)

func applyTLS(cfg *pgx.ConnConfig, material TLSMaterial, mode string) error {
	if mode == "" {
		mode = "prefer"
	}
	if mode == "disable" {
		return disableTLS(cfg)
	}
	base, err := configuredTLS(cfg, material, mode)
	if err != nil {
		return err
	}
	return configureTLSMode(cfg, base, mode)
}

func disableTLS(cfg *pgx.ConnConfig) error {
	cfg.TLSConfig = nil
	for _, fallback := range cfg.Fallbacks {
		fallback.TLSConfig = nil
	}
	return nil
}

func configuredTLS(cfg *pgx.ConnConfig, material TLSMaterial, mode string) (*tls.Config, error) {
	base, err := BuildTLSConfig("full", cfg.Host, material)
	if err != nil {
		return nil, err
	}
	roots := base.RootCAs
	if mode == "allow" || mode == "prefer" || mode == "require" {
		base.InsecureSkipVerify = true
	}
	if mode == "verify-ca" {
		base.InsecureSkipVerify = true // VerifyConnection checks the chain without DNS matching.
		base.VerifyConnection = func(state tls.ConnectionState) error { return verifyChain(state, roots) }
	}
	return base, nil
}

// BuildTLSConfig creates client TLS settings without reading certificate files.
func BuildTLSConfig(mode, host string, material TLSMaterial) (*tls.Config, error) {
	if mode == "none" {
		return nil, nil
	}
	if mode != "ca" && mode != "full" {
		return nil, errInvalidTLSMode
	}
	roots, err := rootPool(material.CACert)
	if err != nil {
		return nil, err
	}
	certificates := []tls.Certificate(nil)
	if mode == "full" {
		certificates, err = clientCertificates(material)
		if err != nil {
			return nil, err
		}
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12, RootCAs: roots,
		Certificates: certificates, ServerName: host,
	}, nil
}

func clientCertificates(material TLSMaterial) ([]tls.Certificate, error) {
	if material.ClientCert == "" {
		return nil, nil
	}
	cert, err := tls.X509KeyPair([]byte(material.ClientCert), []byte(material.ClientKey))
	if err != nil {
		return nil, fmt.Errorf("parse client certificate pair: %w", err)
	}
	return []tls.Certificate{cert}, nil
}

func configureTLSMode(cfg *pgx.ConnConfig, base *tls.Config, mode string) error {
	switch mode {
	case "allow":
		cfg.TLSConfig = nil
		cfg.Fallbacks = orderedFallbacks(cfg.Fallbacks, base, true)
	case "prefer":
		cfg.TLSConfig = base
		cfg.Fallbacks = orderedFallbacks(cfg.Fallbacks, base, false)
	default:
		cfg.TLSConfig = base
		for _, fallback := range cfg.Fallbacks {
			fallback.TLSConfig = base.Clone()
		}
	}
	return nil
}

func rootPool(ca string) (*x509.CertPool, error) {
	if ca == "" {
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("load system certificate roots: %w", err)
		}
		return roots, nil
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(ca)) {
		return nil, errInvalidCABundle
	}
	return roots, nil
}

func orderedFallbacks(existing []*pgconn.FallbackConfig, secure *tls.Config, plainFirst bool) []*pgconn.FallbackConfig {
	for _, fallback := range existing {
		if plainFirst {
			fallback.TLSConfig = secure.Clone()
		} else {
			fallback.TLSConfig = nil
		}
	}
	return existing
}

func verifyChain(state tls.ConnectionState, roots *x509.CertPool) error {
	if len(state.PeerCertificates) == 0 {
		return errMissingServerCert
	}
	intermediates := x509.NewCertPool()
	for _, cert := range state.PeerCertificates[1:] {
		intermediates.AddCert(cert)
	}
	_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
		Roots: roots, Intermediates: intermediates,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		return fmt.Errorf("verify server certificate chain: %w", err)
	}
	return nil
}

func sslMode(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "postgres://") && !strings.HasPrefix(rawURL, "postgresql://") {
		for _, field := range strings.Fields(rawURL) {
			key, value, ok := strings.Cut(field, "=")
			if ok && key == "sslmode" {
				return strings.Trim(value, "\"'"), nil
			}
		}
		return "prefer", nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse PostgreSQL admin URL: %w", err)
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return "", fmt.Errorf("parse PostgreSQL admin URL query: %w", err)
	}
	mode := query.Get("sslmode")
	if mode == "" {
		return "prefer", nil
	}
	return mode, nil
}
