package provision

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestBuildTLSConfigForSharedModes(t *testing.T) {
	for _, mode := range []string{"none", "ca", "full"} {
		t.Run(mode, func(t *testing.T) { assertTLSMode(t, mode) })
	}
}

func assertTLSMode(t *testing.T, mode string) {
	t.Helper()
	config, err := BuildTLSConfig(mode, "db.example", TLSMaterial{})
	if err != nil {
		t.Fatal(err)
	}
	if mode == "none" {
		if config != nil {
			t.Fatal("none mode returned a TLS config")
		}
		return
	}
	if config == nil || config.ServerName != "db.example" || config.RootCAs == nil ||
		config.InsecureSkipVerify || len(config.Certificates) != 0 {
		t.Fatalf("unexpected TLS config for %s: %+v", mode, config)
	}
}

func TestBuildFullTLSConfigAddsClientCertificate(t *testing.T) {
	material := clientTLSMaterial(t)
	material.CACert = material.ClientCert
	caConfig, err := BuildTLSConfig("ca", "db.example", material)
	if err != nil || caConfig.RootCAs == nil || len(caConfig.Certificates) != 0 {
		t.Fatalf("CA mode did not install only the supplied CA: config=%+v err=%v", caConfig, err)
	}
	config, err := BuildTLSConfig("full", "db.example", material)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Certificates) != 1 || config.ServerName != "db.example" || config.InsecureSkipVerify {
		t.Fatalf("full TLS config did not add verified client certificate: %+v", config)
	}
}

func clientTLSMaterial(t *testing.T) TLSMaterial {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "client"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return TLSMaterial{
		ClientCert: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		ClientKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKey})),
	}
}
