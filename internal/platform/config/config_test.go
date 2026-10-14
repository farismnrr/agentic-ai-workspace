package config

import (
	"reflect"
	"testing"
)

func TestOriginsDefaultToLocalhostAndLoopback(t *testing.T) {
	t.Setenv("PORT", "8081")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ORIGINS", "")

	cfg := Load()
	want := []string{"http://localhost:8081", "http://127.0.0.1:8081"}

	if !reflect.DeepEqual(cfg.AllowedOrigins, want) || !reflect.DeepEqual(cfg.CORSOrigins, want) {
		t.Fatalf("defaults wrong: %v %v", cfg.AllowedOrigins, cfg.CORSOrigins)
	}
}

func TestOriginsCanBeOverriddenAndCORSFollowsAllowed(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", " https://a.example/ , http://10.0.0.5:8081 ,,")
	t.Setenv("CORS_ORIGINS", "")

	cfg := Load()
	want := []string{"https://a.example", "http://10.0.0.5:8081"}

	if !reflect.DeepEqual(cfg.AllowedOrigins, want) || !reflect.DeepEqual(cfg.CORSOrigins, want) {
		t.Fatalf("parsing wrong: %v %v", cfg.AllowedOrigins, cfg.CORSOrigins)
	}

	t.Setenv("CORS_ORIGINS", "*")

	if got := Load().CORSOrigins; !reflect.DeepEqual(got, []string{"*"}) {
		t.Fatalf("CORS_ORIGINS must be independent when set, got %v", got)
	}
}
