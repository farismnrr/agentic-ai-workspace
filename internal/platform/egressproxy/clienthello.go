package egressproxy

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"golang.org/x/crypto/cryptobyte"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// errHello means the first bytes of a tunnel are not a TLS ClientHello this proxy accepts. Every reason gets the
// same answer: the tunnel is closed.
var errHello = errors.New("not an acceptable TLS ClientHello")

const (
	recordHandshake   = 22
	recordHeaderLen   = 5
	maxRecordLen      = 1 << 14
	handshakeHello    = 1
	extServerName     = 0
	extECH            = 0xfe0d
	nameTypeHost      = 0
	maxSessionIDLen   = 32
	randomLen         = 32
	minLegacyVersion  = 0x0301
	recordVersionHigh = 3
	byteShift         = 8
	maxByteLen        = 255
)

// readClientHello reads exactly the first TLS record from r. It returns those bytes unchanged, so they can be
// forwarded, and the server name the client asked for in canonical form. It accepts only one unfragmented
// ClientHello in one record with exactly one host_name, no encrypted_client_hello and no repeated extension. That is
// all it proves: the destination is the name that was approved. It cannot tell that the rest of the tunnel is TLS.
func readClientHello(r io.Reader) (raw []byte, serverName string, err error) {
	header := make([]byte, recordHeaderLen)
	if _, err = io.ReadFull(r, header); err != nil {
		return nil, "", fmt.Errorf("%w: %w", errHello, err)
	}

	length := int(header[3])<<byteShift | int(header[4])
	if header[0] != recordHandshake || header[1] != recordVersionHigh || header[2] < 1 || header[2] > 3 ||
		length == 0 || length > maxRecordLen {
		return nil, "", errHello
	}

	body := make([]byte, length)
	if _, err = io.ReadFull(r, body); err != nil {
		return nil, "", fmt.Errorf("%w: %w", errHello, err)
	}

	name, err := serverNameOf(body)
	if err != nil {
		return nil, "", err
	}

	return slices.Concat(header, body), name, nil
}

// serverNameOf checks a ClientHello handshake message and returns its canonical server name.
func serverNameOf(body []byte) (string, error) {
	msg := cryptobyte.String(body)

	var (
		kind  uint8
		hello cryptobyte.String
	)

	if !msg.ReadUint8(&kind) || kind != handshakeHello || !msg.ReadUint24LengthPrefixed(&hello) || !msg.Empty() ||
		!skipHelloFields(&hello) {
		return "", errHello
	}

	var extensions cryptobyte.String
	if !hello.ReadUint16LengthPrefixed(&extensions) || !hello.Empty() || extensions.Empty() {
		return "", errHello
	}

	return serverNameIn(extensions)
}

// skipHelloFields consumes version, random, session id, cipher suites and compression methods, checking their sizes.
func skipHelloFields(hello *cryptobyte.String) bool {
	var version uint16

	return hello.ReadUint16(&version) && version >= minLegacyVersion && hello.Skip(randomLen) &&
		skipPrefixed8(hello, 0, maxSessionIDLen) && skipSuites(hello) && skipPrefixed8(hello, 1, maxByteLen)
}

func skipPrefixed8(s *cryptobyte.String, minLen, maxLen int) bool {
	var v cryptobyte.String

	return s.ReadUint8LengthPrefixed(&v) && len(v) >= minLen && len(v) <= maxLen
}

func skipSuites(s *cryptobyte.String) bool {
	var suites cryptobyte.String

	return s.ReadUint16LengthPrefixed(&suites) && len(suites) >= 2 && len(suites)%2 == 0
}

// serverNameIn walks the extensions once, refusing a repeated type and encrypted_client_hello.
func serverNameIn(extensions cryptobyte.String) (string, error) {
	var (
		name string
		seen = map[uint16]bool{}
	)

	for !extensions.Empty() {
		var (
			kind uint16
			data cryptobyte.String
		)

		if !extensions.ReadUint16(&kind) || !extensions.ReadUint16LengthPrefixed(&data) || seen[kind] || kind == extECH {
			return "", errHello
		}

		seen[kind] = true

		if kind == extServerName {
			var ok bool
			if name, ok = parseServerName(data); !ok {
				return "", errHello
			}
		}
	}

	if name == "" {
		return "", errHello
	}

	return name, nil
}

// parseServerName accepts a list of exactly one host_name.
func parseServerName(data cryptobyte.String) (string, bool) {
	var (
		list cryptobyte.String
		kind uint8
		raw  cryptobyte.String
	)

	if !data.ReadUint16LengthPrefixed(&list) || !data.Empty() || !list.ReadUint8(&kind) || kind != nameTypeHost ||
		!list.ReadUint16LengthPrefixed(&raw) || !list.Empty() || len(raw) == 0 {
		return "", false
	}

	return egress.CanonicalHost(string(raw))
}
