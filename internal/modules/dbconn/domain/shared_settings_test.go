package domain

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestValidateCertificatesAcceptsCAAndMatchingClientPair(t *testing.T) {
	cert, key := testCertificate(t)
	settings := SharedSettings{CACert: cert, ClientCert: cert, ClientKey: key}
	if err := settings.ValidateCertificates(); err != nil {
		t.Fatalf("valid PEM rejected: %v", err)
	}
}

func TestValidateCertificatesRejectsInvalidAndMismatchedPEM(t *testing.T) {
	cert, key := testCertificate(t)
	otherCert, _ := testCertificate(t)
	tooLarge := strings.Repeat("x", MaxPEMBytes+1)
	tests := []SharedSettings{
		{CACert: "not a certificate"},
		{ClientCert: cert},
		{ClientCert: otherCert, ClientKey: key},
		{ClientCert: cert, ClientKey: encryptedKey(key)},
		{CACert: tooLarge},
		{ClientCert: tooLarge},
		{ClientKey: tooLarge},
	}
	for _, settings := range tests {
		if err := settings.ValidateCertificates(); err == nil {
			t.Fatal("invalid PEM accepted")
		}
	}
}

func testCertificate(t *testing.T) (string, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "client"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	key := x509.MarshalPKCS1PrivateKey(privateKey)
	return string(cert), string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: key}))
}

func encryptedKey(key string) string {
	block, _ := pem.Decode([]byte(key))
	return string(pem.EncodeToMemory(&pem.Block{Type: block.Type, Bytes: block.Bytes, Headers: map[string]string{
		"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-256-CBC,0000000000000000",
	}}))
}
