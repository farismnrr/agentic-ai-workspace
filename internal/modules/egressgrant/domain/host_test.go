package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeHostGivesOneSpelling(t *testing.T) {
	want := map[string]string{
		"sb-f6c7943aa05fb36c.sb.molab.run": "sb-f6c7943aa05fb36c.sb.molab.run",
		"SB-F6C7943AA05FB36C.SB.MOLAB.RUN": "sb-f6c7943aa05fb36c.sb.molab.run",
		"example.com.":                     "example.com",
		"Example.COM":                      "example.com",
		"bücher.example":                   "xn--bcher-kva.example",
		"xn--bcher-kva.example":            "xn--bcher-kva.example",
		"example。com":                      "example.com",
	}

	for in, out := range want {
		got, err := NormalizeHost(in)
		if err != nil || got != out {
			t.Errorf("%q: got %q, %v; want %q", in, got, err, out)
		}
	}
}

func TestNormalizeHostRefusesAnythingButAPublicName(t *testing.T) {
	bad := []string{
		"", " ", ".", "..", "example.com..", ".example.com", "a..example.com", "*.example.com", "example.*",
		"example.com:443", "example.com:80", "user@example.com", "https://example.com", "example.com/path",
		"example.com?x=1", "example.com#x", "exa mple.com", "example.com\n", "ex\\ample.com",
		"localhost", "localhost.", "app.localhost", "printer.local", "db.internal", "nas.lan", "router.home.arpa",
		"host.localdomain", "singlelabel", "-bad.example.com", "bad-.example.com", "under_score.example.com",
		"127.0.0.1", "127.1", "2130706433", "0x7f.0x0.0x0.0x1", "0x7f.1", "010.1.1.1", "1.2.3.4.", "10.0.0.5",
		"::1", "[::1]", "[::ffff:127.0.0.1]", "fe80::1%eth0", "::ffff:7f00:1", " example.com", "example.com ",
		strings.Repeat("a", 64) + ".example.com", strings.Repeat("a.", 130) + "com", "xn--.example.com",
	}

	for _, in := range bad {
		if got, err := NormalizeHost(in); !errors.Is(err, ErrInvalidHost) {
			t.Errorf("%q must be refused, got %q, %v", in, got, err)
		}
	}
}

func TestNormalizeHostIsStableWhenRepeated(t *testing.T) {
	once, err := NormalizeHost("Bücher.Example.")
	if err != nil {
		t.Fatal(err)
	}

	twice, err := NormalizeHost(once)
	if err != nil || twice != once {
		t.Fatalf("second pass changed %q into %q, %v", once, twice, err)
	}
}
