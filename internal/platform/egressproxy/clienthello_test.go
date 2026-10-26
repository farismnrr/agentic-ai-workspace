package egressproxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// helloFor makes the bytes a real Go TLS client sends first, which is what sandboxes will send.
func helloFor(t *testing.T, serverName string) []byte {
	t.Helper()

	client, server := net.Pipe()

	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	go func() {
		cfg := &tls.Config{ServerName: serverName, InsecureSkipVerify: true} //nolint:gosec // only the first flight is wanted
		conn := tls.Client(client, cfg)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		defer cancel()

		_ = conn.HandshakeContext(ctx)
	}()

	_ = server.SetReadDeadline(time.Now().Add(time.Second))

	buf := make([]byte, 1<<15)

	n, err := server.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	return buf[:n]
}

func TestAClientHelloYieldsItsCanonicalServerNameAndTheSameBytes(t *testing.T) {
	for in, want := range map[string]string{
		"sb.example.com": "sb.example.com", "Sb.Example.COM": "sb.example.com", "bücher.example": "xn--bcher-kva.example",
	} {
		hello := helloFor(t, in)

		raw, name, err := readClientHello(bytes.NewReader(append(append([]byte{}, hello...), "tail"...)))
		if err != nil || name != want || !bytes.Equal(raw, hello) {
			t.Errorf("%q: name %q err %v same %v", in, name, err, bytes.Equal(raw, hello))
		}
	}
}

func TestOnlyTheFirstRecordIsConsumed(t *testing.T) {
	hello := helloFor(t, "a.example.com")
	r := bytes.NewReader(append(append([]byte{}, hello...), "after"...))

	if _, _, err := readClientHello(r); err != nil || r.Len() != len("after") {
		t.Fatalf("err %v, %d bytes left", err, r.Len())
	}
}

func TestNoNameOrAnAddressAsNameIsRefused(t *testing.T) {
	for _, name := range []string{"", "127.0.0.1", "10.0.0.5", "::1"} {
		if _, got, err := readClientHello(bytes.NewReader(helloFor(t, name))); err == nil {
			t.Errorf("%q gave %q", name, got)
		}
	}
}

func TestNotATLSHelloIsRefused(t *testing.T) {
	good := helloFor(t, "a.example.com")

	cases := map[string][]byte{
		"empty": nil, "plaintext http": []byte("GET / HTTP/1.1\r\nHost: a.example.com\r\n\r\n"),
		"ssh": []byte("SSH-2.0-OpenSSH_9.6\r\n"), "short header": good[:3], "truncated body": good[:len(good)-1],
		"application data": withByte(good, 0, 23), "alert": withByte(good, 0, 21),
		"ssl2 version": withByte(good, 1, 2), "major 4": withByte(good, 1, 4), "minor 0": withByte(good, 2, 0),
		"length zero": withLength(good, 0), "length too big": withLength(good, maxRecordLen+1),
		"not a hello": withByte(good, 5, 2), "handshake longer": withByte(good, 8, good[8]+1),
		"handshake shorter": withByte(good, 8, good[8]-1), "trailing bytes in record": trailing(good),
	}

	for name, in := range cases {
		if _, got, err := readClientHello(bytes.NewReader(in)); err == nil {
			t.Errorf("%s was accepted as %q", name, got)
		}
	}
}

func withByte(in []byte, at int, value byte) []byte {
	out := append([]byte{}, in...)
	out[at] = value

	return out
}

// trailing adds a byte inside the record after the hello, with both lengths left saying otherwise fixed up.
func trailing(in []byte) []byte {
	return withLength(append(append([]byte{}, in...), 0), len(in)+1-recordHeaderLen)
}

// withLength sets the record length in the header, leaving the body as it is.
func withLength(in []byte, length int) []byte {
	out := append([]byte{}, in...)
	binary.BigEndian.PutUint16(out[3:5], uint16(length)) //nolint:gosec // lengths in these tests fit

	return out
}

func FuzzReadClientHello(f *testing.F) {
	f.Add([]byte("GET / HTTP/1.1\r\n\r\n"))
	f.Add([]byte{22, 3, 1, 0, 4, 1, 0, 0, 0})
	f.Add([]byte{22, 3, 3, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, in []byte) {
		raw, name, err := readClientHello(bytes.NewReader(in))
		if err == nil && (len(name) == 0 || !bytes.Equal(raw, in[:len(raw)])) {
			t.Fatalf("accepted %q %q", name, raw)
		}
	})
}
