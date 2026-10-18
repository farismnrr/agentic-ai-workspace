package domain_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

func TestOnlyLicencesThatAllowEditedCommercialUseAreUsable(t *testing.T) {
	t.Parallel()

	usable := []string{
		"CC0", "Public domain", "pdm", "CC BY 4.0", "CC BY-SA 3.0", "by", "by-sa", "CC-BY-4.0", "cc by 2.0",
	}
	banned := []string{
		"CC BY-NC 4.0", "by-nc", "by-nc-sa", "CC BY-ND 2.0", "by-nd", "All rights reserved", "", "unknown", "Fair use",
	}

	for _, name := range usable {
		if !domain.UsableLicense(name) {
			t.Errorf("%q should be usable", name)
		}
	}

	for _, name := range banned {
		if domain.UsableLicense(name) {
			t.Errorf("%q must not be usable", name)
		}
	}
}

func TestQueriesAndAddressesAreChecked(t *testing.T) {
	t.Parallel()

	if got, err := domain.ValidQuery("  warung   kopi \n bandung "); err != nil || got != "warung kopi bandung" {
		t.Fatalf("query = %q, %v", got, err)
	}

	for _, bad := range []string{"", "a", string(make([]rune, 201))} {
		if _, err := domain.ValidQuery(bad); !errors.Is(err, domain.ErrBadQuery) {
			t.Errorf("%q must be refused: %v", bad, err)
		}
	}

	refused := []string{
		"", "ftp://x.com/a", "file:///etc/passwd", "javascript:alert(1)", "http://", "https://user:pw@x.com/", "x.com",
	}

	for _, bad := range refused {
		if _, err := domain.ValidURL(bad); !errors.Is(err, domain.ErrBadURL) {
			t.Errorf("%q must be refused: %v", bad, err)
		}
	}

	if _, err := domain.ValidURL("https://example.com/a?b=1"); err != nil {
		t.Fatal(err)
	}
}

func TestACreditNamesTheWorkItsAuthorAndTheLicence(t *testing.T) {
	t.Parallel()

	line := domain.Image{
		Title: "Coffee Shop Sign", Creator: "pixeljones", License: "CC BY-SA 2.0",
		LicenseURL: "https://creativecommons.org/licenses/by-sa/2.0/", Page: "https://www.flickr.com/photos/1/2",
	}.Credit()

	want := "\"Coffee Shop Sign\" by pixeljones, CC BY-SA 2.0 " +
		"(https://creativecommons.org/licenses/by-sa/2.0/), from https://www.flickr.com/photos/1/2"
	if line != want {
		t.Fatalf("credit = %q", line)
	}
}

func TestOnlyPublicDomainAndCC0AreFreeToUse(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]bool{
		"CC0": true, "Public domain": true, "pdm": true, "CC BY 2.0": false, "CC BY-SA 4.0": false, "by": false,
		"CC0 NC": false, "": false,
	} {
		if got := domain.FreeToUse(name); got != want {
			t.Errorf("%q free to use = %v, want %v", name, got, want)
		}
	}
}
