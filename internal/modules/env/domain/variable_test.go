package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		want error
		name string
	}{
		{name: "DATABASE_URL"},
		{name: "_A"},
		{name: strings.Repeat("A", MaxNameSize)},
		{name: "", want: ErrInvalidName},
		{name: "1A", want: ErrInvalidName},
		{name: "lower", want: ErrInvalidName},
		{name: strings.Repeat("A", MaxNameSize+1), want: ErrInvalidName},
		{name: "PATH", want: ErrReserved},
		{name: "home", want: ErrReserved},
		{name: "LD_PRELOAD", want: ErrReserved},
		{name: "GOFLAGS", want: ErrReserved},
		{name: "GOOGLE_API_KEY"},
		{name: "CGO_ENABLED", want: ErrReserved},
		{name: "DOCKER_HOST", want: ErrReserved},
		{name: "ALL_PROXY", want: ErrReserved},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateName(test.name)
			if !errors.Is(err, test.want) {
				t.Fatalf("ValidateName(%q) = %v, want %v", test.name, err, test.want)
			}
		})
	}
}

func TestValidateValueRejectsControlCharactersExceptTab(t *testing.T) {
	if err := ValidateValue("line one\tline two"); err != nil {
		t.Fatalf("tab should be allowed: %v", err)
	}
	for _, value := range []string{"line\nbreak", "carriage\rreturn", "nul\x00byte", "delete\x7f"} {
		if !errors.Is(ValidateValue(value), ErrInvalidValue) {
			t.Errorf("control character in %q was accepted", value)
		}
	}
}

func TestValidateValueUsesByteLimit(t *testing.T) {
	if err := ValidateValue(strings.Repeat("x", MaxValueSize)); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ValidateValue(strings.Repeat("x", MaxValueSize+1)), ErrInvalidValue) {
		t.Fatal("oversized value was accepted")
	}
}
