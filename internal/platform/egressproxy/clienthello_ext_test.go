package egressproxy

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/cryptobyte"
)

type hello struct {
	sessionID []byte
	suites    []byte
	compress  []byte
	exts      [][2][]byte
	trailing  []byte
	version   uint16
}

func goodHello() hello {
	return hello{
		version: 0x0303, sessionID: make([]byte, 32), suites: []byte{0x13, 0x01}, compress: []byte{0},
		exts: [][2][]byte{{{0, 0}, sni("sb.example.com")}, {{0, 0x2b}, {2, 3, 4}}},
	}
}

// sni encodes a server_name extension holding one host_name.
func sni(name string) []byte { return sniEntries(entry{kind: 0, name: name}) }

type entry struct {
	name string
	kind byte
}

func sniEntries(entries ...entry) []byte {
	var b cryptobyte.Builder

	b.AddUint16LengthPrefixed(func(list *cryptobyte.Builder) {
		for _, e := range entries {
			list.AddUint8(e.kind)
			list.AddUint16LengthPrefixed(func(n *cryptobyte.Builder) { n.AddBytes([]byte(e.name)) })
		}
	})

	return b.BytesOrPanic()
}

func (h hello) bytes() []byte {
	var body cryptobyte.Builder

	body.AddUint16(h.version)
	body.AddBytes(make([]byte, 32))
	body.AddUint8LengthPrefixed(func(b *cryptobyte.Builder) { b.AddBytes(h.sessionID) })
	body.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) { b.AddBytes(h.suites) })
	body.AddUint8LengthPrefixed(func(b *cryptobyte.Builder) { b.AddBytes(h.compress) })
	body.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		for _, e := range h.exts {
			b.AddBytes(e[0])
			b.AddUint16LengthPrefixed(func(d *cryptobyte.Builder) { d.AddBytes(e[1]) })
		}
	})
	body.AddBytes(h.trailing)

	var msg cryptobyte.Builder

	msg.AddUint8(1)
	msg.AddUint24LengthPrefixed(func(b *cryptobyte.Builder) { b.AddBytes(body.BytesOrPanic()) })

	var rec cryptobyte.Builder

	rec.AddBytes([]byte{22, 3, 1})
	rec.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) { b.AddBytes(msg.BytesOrPanic()) })

	return rec.BytesOrPanic()
}

func TestTheBuilderMakesAnAcceptableHello(t *testing.T) {
	if _, name, err := readClientHello(bytes.NewReader(goodHello().bytes())); err != nil || name != "sb.example.com" {
		t.Fatalf("%q %v", name, err)
	}
}

func TestMalformedHellosAreRefused(t *testing.T) {
	mutations := map[string]func(*hello){
		"old version":     func(h *hello) { h.version = 0x0200 },
		"long session id": func(h *hello) { h.sessionID = make([]byte, 33) },
		"no suites":       func(h *hello) { h.suites = nil },
		"odd suites":      func(h *hello) { h.suites = []byte{1, 2, 3} },
		"no compression":  func(h *hello) { h.compress = nil },
		"no extensions":   func(h *hello) { h.exts = nil },
		"no server name":  func(h *hello) { h.exts = h.exts[1:] },
		"trailing bytes":  func(h *hello) { h.trailing = []byte{0} },
		"duplicate sni":   func(h *hello) { h.exts = append(h.exts, h.exts[0]) },
		"duplicate other": func(h *hello) { h.exts = append(h.exts, h.exts[1]) },
		"ech":             func(h *hello) { h.exts = append(h.exts, [2][]byte{{0xfe, 0x0d}, {0}}) },
		"empty name":      func(h *hello) { h.exts[0][1] = sni("") },
		"ip name":         func(h *hello) { h.exts[0][1] = sni("127.0.0.1") },
		"wildcard name":   func(h *hello) { h.exts[0][1] = sni("*.example.com") },
		"port in name":    func(h *hello) { h.exts[0][1] = sni("sb.example.com:443") },
		"single label":    func(h *hello) { h.exts[0][1] = sni("localhost") },
		"two names": func(h *hello) {
			h.exts[0][1] = sniEntries(entry{kind: 0, name: "sb.example.com"}, entry{kind: 0, name: "other.example.com"})
		},
		"name type not 0":   func(h *hello) { h.exts[0][1] = sniEntries(entry{kind: 1, name: "sb.example.com"}) },
		"sni list too long": func(h *hello) { h.exts[0][1][0] = 0xff },
		"empty sni data":    func(h *hello) { h.exts[0][1] = nil },
	}

	for name, mutate := range mutations {
		h := goodHello()
		h.exts = append([][2][]byte{}, h.exts...)
		mutate(&h)

		if _, got, err := readClientHello(bytes.NewReader(h.bytes())); err == nil {
			t.Errorf("%s was accepted as %q", name, got)
		}
	}
}

func TestAHelloLongerThanOneRecordIsRefused(t *testing.T) {
	h := goodHello()
	h.exts = append(h.exts, [2][]byte{{0xaa, 0xaa}, make([]byte, maxRecordLen)})

	if _, _, err := readClientHello(bytes.NewReader(h.bytes())); err == nil {
		t.Fatal("a ClientHello split over records (or too big for one) must be refused")
	}
}
