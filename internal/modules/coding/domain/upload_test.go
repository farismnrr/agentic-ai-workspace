package domain

import (
	"errors"
	"testing"
)

func TestClassifyRecognizesAllowedHeaders(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		kind string
		ext  string
		head []byte
	}{
		{name: "png", head: []byte("\x89PNG\r\n\x1a\n"), kind: "image", ext: ".png"},
		{name: "jpeg", head: []byte("\xff\xd8\xff"), kind: "image", ext: ".jpg"},
		{name: "gif", head: []byte("GIF89a"), kind: "image", ext: ".gif"},
		{name: "webp", head: []byte("RIFFxxxxWEBP"), kind: "image", ext: ".webp"},
		{name: "svg", head: []byte(`<svg xmlns="http://www.w3.org/2000/svg">`), kind: "image", ext: ".svg"},
		{name: "pdf", head: []byte("%PDF-1.7"), kind: "pdf", ext: ".pdf"},
		{
			name: "webm", head: []byte{0x1a, 0x45, 0xdf, 0xa3, 0x88, 0x42, 0x82, 0x84, 'w', 'e', 'b', 'm'},
			kind: "video", ext: ".webm",
		},
		{name: "mp4", head: []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}, kind: "video", ext: ".mp4"},
		{name: "mov", head: []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'q', 't', ' ', ' '}, kind: "video", ext: ".mov"},
		{name: "text", head: []byte("hello\n"), kind: "text", ext: ".txt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upload, _, err := Classify(tc.name+tc.ext, tc.head)
			if err != nil || upload.Kind != tc.kind || upload.Ext != tc.ext {
				t.Fatalf("Classify() = %#v, %v", upload, err)
			}
		})
	}
}

func TestClassifyRejectsWrongOrUnsupportedContent(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"renamed executable.pdf", []byte("MZ executable")},
		{"matroska.webm", []byte{0x1a, 0x45, 0xdf, 0xa3, 'm', 'a', 't', 'r', 'o', 's', 'k', 'a'}},
		{"heic.mp4", []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c'}},
		{"avif.mp4", []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'a', 'v', 'i', 'f'}},
		{"3gp.mp4", []byte{0, 0, 0, 20, 'f', 't', 'y', 'p', '3', 'g', 'p', '4', 0, 0, 0, 0, 'i', 's', 'o', 'm'}},
		{"invalid text.txt", []byte{0xff, 0xfe}},
		{"nul text.txt", []byte{'a', 0, 'b'}},
	} {
		if _, _, err := Classify(tc.name, tc.data); !errors.Is(err, ErrUnsupported) {
			t.Errorf("%s: error = %v", tc.name, err)
		}
	}
	if _, _, err := Classify("empty.txt", nil); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty error = %v", err)
	}
}

func TestStreamingCheckValidatesTextAndSVGAcrossWrites(t *testing.T) {
	t.Parallel()
	_, textCheck, _ := Classify("notes.txt", []byte("ok"))
	_, _ = textCheck.Write([]byte{0xe2})
	_, _ = textCheck.Write([]byte{0x82, 0xac})
	if err := textCheck.Err(); err != nil {
		t.Fatalf("split UTF-8 rune rejected: %v", err)
	}
	_, badText, _ := Classify("notes.txt", []byte("ok"))
	_, _ = badText.Write([]byte{0xff})
	if !errors.Is(badText.Err(), ErrBinary) {
		t.Fatalf("invalid UTF-8 accepted: %v", badText.Err())
	}
	_, svgCheck, _ := Classify("icon.svg", []byte("<svg>"))
	_, _ = svgCheck.Write([]byte("<scr"))
	_, _ = svgCheck.Write([]byte("ipt>alert(1)</script>"))
	if !errors.Is(svgCheck.Err(), ErrBinary) {
		t.Fatalf("split script accepted: %v", svgCheck.Err())
	}
	_, pngCheck, _ := Classify("image.png", []byte("\x89PNG\r\n\x1a\n"))
	_, _ = pngCheck.Write([]byte("<script>"))
	if err := pngCheck.Err(); err != nil {
		t.Fatalf("PNG bytes received SVG-only validation: %v", err)
	}
}

func TestSVGCheckRejectsActiveContentAndExternalReferences(t *testing.T) {
	t.Parallel()
	for _, payload := range []string{
		`<svg>< foreignObject><body>bad</body></foreignObject></svg>`,
		`<svg><SVG:SCRIPT>alert(1)</SVG:SCRIPT></svg>`,
		`<svg><g OnLoad   ="alert(1)"/></svg>`,
		`<svg><a href="jav&#x61;script:alert(1)"/></svg>`,
		`<svg><a href="&#x6a;avascript:alert(1)"/></svg>`,
		`<svg><a href="data:image/svg+xml,bad"/></svg>`,
		`<svg><a xlink:href="https://example.test/x.svg"/></svg>`,
	} {
		_, check, err := Classify("drawing.svg", []byte("<svg>"))
		if err != nil {
			t.Fatalf("classify SVG: %v", err)
		}
		for offset := 0; offset < len(payload); offset += 3 {
			end := min(offset+3, len(payload))
			if _, err := check.Write([]byte(payload[offset:end])); err != nil {
				t.Fatalf("write payload %q: %v", payload, err)
			}
		}
		if !errors.Is(check.Err(), ErrBinary) {
			t.Errorf("unsafe SVG accepted: %s", payload)
		}
	}
}

func TestSVGCheckAllowsFragmentReferences(t *testing.T) {
	t.Parallel()
	_, check, err := Classify("drawing.svg", []byte("<svg>"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = check.Write([]byte(`<svg><use href="#shape"/><a xlink:href="#page"/></svg>`))
	if err := check.Err(); err != nil {
		t.Fatalf("fragment references rejected: %v", err)
	}
}
