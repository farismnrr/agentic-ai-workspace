package application

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func TestSharedSettingsKeepAndRemoveCertificatesForEveryKind(t *testing.T) {
	cert, key := settingsTestCertificate(t)
	store := &sharedSettingsMemory{missing: true}
	service := NewSharedSettingsService(store, sharedSettingsCipher{})
	for _, kind := range domain.SharedKinds {
		input := domain.SharedSettings{
			Host: "db", Database: "app", User: "admin", Password: "secret",
			TLS: "full", CACert: cert, ClientCert: cert, ClientKey: key,
		}
		if _, err := service.Save(context.Background(), kind, input); err != nil {
			t.Fatalf("save %s certificate material: %v", kind, err)
		}
		assertKindCertificates(t, service, kind, true)
		input.Password, input.CACert, input.ClientCert, input.ClientKey = "", "", "", ""
		if _, err := service.Save(context.Background(), kind, input); err != nil {
			t.Fatalf("keep %s certificate material: %v", kind, err)
		}
		assertKindCertificates(t, service, kind, true)
		if _, err := service.SaveWithRemovals(context.Background(), kind, input, true, true); err != nil {
			t.Fatalf("remove %s certificate material: %v", kind, err)
		}
		assertKindCertificates(t, service, kind, false)
	}
}

func TestInvalidPEMIsRejectedForEveryKind(t *testing.T) {
	cert, key := settingsTestCertificate(t)
	otherCert, _ := settingsTestCertificate(t)
	block, _ := pem.Decode([]byte(key))
	encryptedKey := string(pem.EncodeToMemory(&pem.Block{
		Type: block.Type, Bytes: block.Bytes,
		Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-256-CBC,00"},
	}))
	for _, kind := range domain.SharedKinds {
		for _, material := range []struct{ ca, cert, key string }{
			{ca: "bad PEM"},
			{cert: otherCert, key: key},
			{cert: cert, key: encryptedKey},
			{ca: strings.Repeat("x", domain.MaxPEMBytes+1)},
		} {
			service := NewSharedSettingsService(&sharedSettingsMemory{missing: true}, sharedSettingsCipher{})
			_, err := service.Save(context.Background(), kind, domain.SharedSettings{
				Host: "db", Database: "app", User: "admin", Password: "secret", TLS: "full",
				CACert: material.ca, ClientCert: material.cert, ClientKey: material.key,
			})
			if !errors.Is(err, domain.ErrInvalidPEM) {
				t.Fatalf("%s accepted invalid PEM %+v: %v", kind, material, err)
			}
		}
	}
}

func assertKindCertificates(t *testing.T, service *SharedSettingsService, kind string, want bool) {
	t.Helper()
	view, err := service.Get(context.Background(), kind)
	if err != nil || view.CACertSet != want || view.ClientCertSet != want || view.ClientKeySet != want {
		t.Fatalf("%s certificate flags = %+v, err=%v", kind, view, err)
	}
}

func settingsTestCertificate(t *testing.T) (string, string) {
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
