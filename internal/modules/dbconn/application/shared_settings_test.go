package application

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func TestSharedSettingsSaveSealsAndRoundTripsReservedPassword(t *testing.T) {
	store := &sharedSettingsMemory{missing: true}
	cipher := sharedSettingsCipher{}
	service := NewSharedSettingsService(store, cipher)
	want := domain.SharedSettings{
		Host: "db.example", Port: 5432, Database: "workspace", User: "admin",
		Password: strings.Join([]string{"p@", "ss:/?% ", `"雪`}, ""), SSLMode: "verify-full",
	}
	view, err := service.Save(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	assertSharedSettingsSaved(t, view, store.record, want.Password)
	got, configured, err := service.AdminSettings(context.Background())
	assertAdminSettings(t, got, configured, err, want.Password)
	view, err = service.Get(context.Background())
	assertPasswordFlag(t, view, err)
}

func assertSharedSettingsSaved(t *testing.T, view SharedSettingsView, saved StoredSharedSettings, password string) {
	t.Helper()
	if !view.PasswordSet || !view.Configured {
		t.Fatalf("save response does not report configured password: %+v", view)
	}
	if saved.PasswordSealed == password || saved.Settings.Password != "" {
		t.Fatal("password was persisted in plaintext")
	}
}

func assertAdminSettings(t *testing.T, got domain.SharedSettings, configured bool, err error, password string) {
	t.Helper()
	if err != nil || !configured {
		t.Fatalf("load saved config: configured=%v err=%v", configured, err)
	}
	if got.Password != password {
		t.Fatalf("password round trip = %q", got.Password)
	}
}

func assertPasswordFlag(t *testing.T, view SharedSettingsView, err error) {
	t.Helper()
	if err != nil || !view.PasswordSet {
		t.Fatalf("GET view: %+v, %v", view, err)
	}
}

func TestSharedSettingsMissingAndPasswordCanBeKept(t *testing.T) {
	store := &sharedSettingsMemory{missing: true}
	service := NewSharedSettingsService(store, sharedSettingsCipher{})
	if _, err := service.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.AdminSettings(context.Background()); err != nil {
		t.Fatal(err)
	}
	input := domain.SharedSettings{
		Host: "db", Port: 5432, Database: "db", User: "admin",
		Password: strings.Join([]string{"sec", "ret"}, ""), SSLMode: "disable",
	}
	if _, err := service.Save(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	input.Password = ""
	if _, err := service.Save(context.Background(), input); err != nil {
		t.Fatal(err)
	}
}

func TestSharedSettingsKeepsAndRemovesSealedCertificates(t *testing.T) {
	cert, key := settingsTestCertificate(t)
	store := &sharedSettingsMemory{missing: true}
	service := NewSharedSettingsService(store, sharedSettingsCipher{})
	input := certificateSettings(cert, key)
	saveCertificates(t, service, input, false, false)
	assertCertificatesSealed(t, store.record, cert, key)
	input.CACert, input.ClientCert, input.ClientKey, input.Password = "", "", "", ""
	saveCertificates(t, service, input, false, false)
	assertCertificateFlags(t, service, true)
	saveCertificates(t, service, input, true, true)
	assertCertificateFlags(t, service, false)
}

func certificateSettings(cert, key string) domain.SharedSettings {
	return domain.SharedSettings{
		Host: "db", Port: 5432, Database: "db", User: "admin", Password: "secret",
		SSLMode: "verify-full", CACert: cert, ClientCert: cert, ClientKey: key,
	}
}

func saveCertificates(
	t *testing.T, service *SharedSettingsService, input domain.SharedSettings, removeCA, removeClient bool,
) {
	t.Helper()
	if _, err := service.SaveWithRemovals(context.Background(), input, removeCA, removeClient); err != nil {
		t.Fatal(err)
	}
}

func assertCertificatesSealed(t *testing.T, stored StoredSharedSettings, cert, key string) {
	t.Helper()
	if stored.CACertSealed == cert || stored.ClientKeySealed == key {
		t.Fatal("PEM stored unsealed")
	}
}

func assertCertificateFlags(t *testing.T, service *SharedSettingsService, wantSet bool) {
	t.Helper()
	view, err := service.Get(context.Background())
	if err != nil || view.CACertSet != wantSet || view.ClientCertSet != wantSet || view.ClientKeySet != wantSet {
		t.Fatalf("certificate flags = %+v, err=%v", view, err)
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

type sharedSettingsMemory struct {
	record  StoredSharedSettings
	missing bool
}

func (s *sharedSettingsMemory) Get(context.Context) (StoredSharedSettings, error) {
	if s.missing {
		return StoredSharedSettings{}, ErrSharedSettingsMissing
	}
	return s.record, nil
}

func (s *sharedSettingsMemory) Save(_ context.Context, record StoredSharedSettings) error {
	s.record = record
	s.missing = false
	return nil
}

type sharedSettingsCipher struct{}

func (sharedSettingsCipher) Encrypt(plain []byte) (string, error) {
	return "sealed:" + string(plain), nil
}

func (sharedSettingsCipher) Decrypt(sealed string) ([]byte, error) {
	return []byte(strings.TrimPrefix(sealed, "sealed:")), nil
}
