package egress

import "testing"

func TestCanonicalHostGivesOneSpelling(t *testing.T) {
	want := map[string]string{
		"Example.COM": "example.com", "example.com.": "example.com", "bücher.example": "xn--bcher-kva.example",
		"xn--bcher-kva.example": "xn--bcher-kva.example", "example。com": "example.com",
	}

	for in, out := range want {
		if got, ok := CanonicalHost(in); !ok || got != out {
			t.Errorf("%q: %q %v, want %q", in, got, ok, out)
		}
	}
}

func TestCanonicalHostRefusesAnythingButABareName(t *testing.T) {
	for _, in := range []string{
		"", ".", "..", "a..example.com", ".example.com", "*.example.com", "example.com:443", "user@example.com",
		"example.com/x", "exa mple.com", " example.com", "example.com\n", "singlelabel", "127.0.0.1", "127.1",
		"2130706433", "0x7f.0x0.0x0.0x1", "0x7f.1", "010.1.1.1", "::1", "[::1]", "::ffff:7f00:1", "1.2.3.4.",
		"under_score.example.com", "-bad.example.com", "xn--.example.com",
	} {
		if got, ok := CanonicalHost(in); ok {
			t.Errorf("%q must be refused, got %q", in, got)
		}
	}
}
