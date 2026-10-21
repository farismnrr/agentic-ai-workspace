package provision

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestTLSConfigModesAndFallbacks(t *testing.T) {
	for _, mode := range []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"} {
		t.Run(mode, func(t *testing.T) {
			cfg := parsedConfig(t, mode)
			if err := applyTLS(cfg, TLSMaterial{}, mode); err != nil {
				t.Fatal(err)
			}
			assertModeConfig(t, cfg, mode)
		})
	}
}

func TestKeywordDSNWithColonValuesKeepsVerifyFull(t *testing.T) {
	raw := "host=2001:db8::1 port=5432 dbname=app user=admin password=secret:part sslmode=verify-full"
	cfg, err := parseAdminURL(raw)
	if err != nil {
		t.Fatalf("parse keyword DSN: %v", err)
	}
	mode, err := sslMode(raw)
	if err != nil {
		t.Fatalf("read keyword DSN sslmode: %v", err)
	}
	if err := applyTLS(cfg, TLSMaterial{}, mode); err != nil {
		t.Fatalf("apply keyword DSN TLS mode: %v", err)
	}
	if mode != "verify-full" || cfg.TLSConfig == nil || cfg.TLSConfig.InsecureSkipVerify ||
		cfg.TLSConfig.ServerName != "2001:db8::1" {
		t.Fatalf("keyword DSN did not retain verify-full: mode=%q tls=%#v", mode, cfg.TLSConfig)
	}
}

func assertModeConfig(t *testing.T, cfg *pgx.ConnConfig, mode string) {
	t.Helper()
	switch mode {
	case "disable":
		assertTLSDisabled(t, cfg)
	case "allow":
		assertPlainFirst(t, cfg)
	case "prefer":
		assertTLSFirst(t, cfg)
	case "require":
		assertUnverifiedTLS(t, cfg, "require must encrypt without peer verification")
	case "verify-ca":
		assertChainOnlyTLS(t, cfg)
	case "verify-full":
		assertHostnameTLS(t, cfg)
	}
}

func assertChainOnlyTLS(t *testing.T, cfg *pgx.ConnConfig) {
	t.Helper()
	if cfg.TLSConfig == nil || !cfg.TLSConfig.InsecureSkipVerify || cfg.TLSConfig.VerifyConnection == nil {
		t.Fatal("verify-ca must install chain-only verification")
	}
}

func assertHostnameTLS(t *testing.T, cfg *pgx.ConnConfig) {
	t.Helper()
	if cfg.TLSConfig == nil || cfg.TLSConfig.InsecureSkipVerify || cfg.TLSConfig.ServerName != "wrong.example" {
		t.Fatal("verify-full must verify the configured host")
	}
}

func assertTLSDisabled(t *testing.T, cfg *pgx.ConnConfig) {
	t.Helper()
	if cfg.TLSConfig != nil {
		t.Fatal("disable retained primary TLS")
	}
	for _, fallback := range cfg.Fallbacks {
		if fallback.TLSConfig != nil {
			t.Fatal("disable retained fallback TLS")
		}
	}
}

func assertPlainFirst(t *testing.T, cfg *pgx.ConnConfig) {
	t.Helper()
	if cfg.TLSConfig != nil || len(cfg.Fallbacks) == 0 || cfg.Fallbacks[0].TLSConfig == nil {
		t.Fatal("allow must try plaintext before TLS")
	}
}

func assertTLSFirst(t *testing.T, cfg *pgx.ConnConfig) {
	t.Helper()
	if cfg.TLSConfig == nil || len(cfg.Fallbacks) == 0 || cfg.Fallbacks[0].TLSConfig != nil {
		t.Fatal("prefer must try TLS before plaintext")
	}
}

func assertUnverifiedTLS(t *testing.T, cfg *pgx.ConnConfig, message string) {
	t.Helper()
	if cfg.TLSConfig == nil || !cfg.TLSConfig.InsecureSkipVerify {
		t.Fatal(message)
	}
}

func TestVerifyCAAcceptsWrongHostAndVerifyFullRejectsIt(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := handshakeWithMaterial(server.Listener.Addr().String(), "verify-ca", string(certPEM)); err != nil {
		t.Fatalf("chain-only verification rejected trusted wrong host: %v", err)
	}
	if err := handshakeWithMaterial(server.Listener.Addr().String(), "verify-full", string(certPEM)); err == nil {
		t.Fatal("verify-full accepted the wrong hostname")
	}
}

func handshakeWithMaterial(address, mode, ca string) error {
	cfg, err := parseTLSConfig(mode, ca)
	if err != nil {
		return err
	}
	connection, err := handshake(address, cfg)
	if connection != nil {
		defer func() { _ = connection.Close() }()
	}
	return err
}

func parseTLSConfig(mode, ca string) (*tls.Config, error) {
	cfg, err := parsedConfigValue(mode)
	if err != nil {
		return nil, err
	}
	if err := applyTLS(cfg, TLSMaterial{CACert: ca}, mode); err != nil {
		return nil, err
	}
	return cfg.TLSConfig, nil
}

func parsedConfigValue(mode string) (*pgx.ConnConfig, error) {
	parsed, err := url.Parse("postgres://user:secret@wrong.example/db?sslmode=" + mode)
	if err != nil {
		return nil, fmt.Errorf("parse test PostgreSQL URL: %w", err)
	}
	cfg, err := pgx.ParseConfig(parsed.String())
	if err != nil {
		return nil, fmt.Errorf("parse test PostgreSQL config: %w", err)
	}
	return cfg, nil
}

func TestVerifiedModesRejectUntrustedChain(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	for _, mode := range []string{"verify-ca", "verify-full"} {
		cfg := parsedConfig(t, mode)
		if err := applyTLS(cfg, TLSMaterial{}, mode); err != nil {
			t.Fatal(err)
		}
		connection, err := handshake(server.Listener.Addr().String(), cfg.TLSConfig)
		if connection != nil {
			_ = connection.Close()
		}
		if err == nil {
			t.Fatalf("%s accepted an untrusted chain", mode)
		}
	}
}

func parsedConfig(t *testing.T, mode string) *pgx.ConnConfig {
	t.Helper()
	cfg, err := parsedConfigValue(mode)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func handshake(address string, config *tls.Config) (*tls.Conn, error) {
	connection, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial TLS test server: %w", err)
	}
	client := tls.Client(connection, config.Clone())
	if err := client.HandshakeContext(context.Background()); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("complete TLS handshake: %w", err)
	}
	return client, nil
}
