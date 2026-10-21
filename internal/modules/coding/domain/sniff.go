package domain

import (
	"bytes"
	"strings"
)

const (
	sniffBytes              = 1024
	ebmlSizeMask            = 0x7f
	isoBaseMediaHeaderBytes = 12
)

func classifyBinary(data []byte) (Upload, bool) {
	head := data[:min(len(data), sniffBytes)]
	if upload, ok := classifyImage(head); ok {
		return upload, true
	}
	if bytes.HasPrefix(head, []byte("%PDF-")) {
		return Upload{Kind: "pdf", Ext: ".pdf", Media: "application/pdf", Limit: MaxUploadBytes}, true
	}
	if upload, ok := classifyVideo(head); ok {
		return upload, true
	}
	return Upload{}, false
}

func classifyImage(head []byte) (Upload, bool) {
	switch {
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return imageUpload(".png", "image/png", MaxUploadBytes), true
	case bytes.HasPrefix(head, []byte("\xff\xd8\xff")):
		return imageUpload(".jpg", "image/jpeg", MaxUploadBytes), true
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return imageUpload(".gif", "image/gif", MaxUploadBytes), true
	case len(head) >= isoBaseMediaHeaderBytes && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return imageUpload(".webp", "image/webp", MaxUploadBytes), true
	case bytes.Contains(bytes.ToLower(head), []byte("<svg")):
		return imageUpload(".svg", "image/svg+xml", MaxSVGBytes), true
	default:
		return Upload{}, false
	}
}

func classifyVideo(head []byte) (Upload, bool) {
	if hasWebMType(head) {
		return Upload{Kind: "video", Ext: ".webm", Media: "video/webm", Limit: MaxVideoBytes}, true
	}
	if isISOBaseMedia(head) {
		return isoBaseMediaUpload(head)
	}
	return Upload{}, false
}

func imageUpload(ext, media string, limit int64) Upload {
	return Upload{Kind: "image", Ext: ext, Media: media, Limit: limit}
}

func hasWebMType(head []byte) bool {
	if !bytes.HasPrefix(head, []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return false
	}
	header := head[4:]
	marker := bytes.Index(header, []byte{0x42, 0x82})
	if marker < 0 || marker+3 > len(header) {
		return false
	}
	size := int(header[marker+2] & ebmlSizeMask)
	return size == 4 && marker+3+size <= len(header) && string(header[marker+3:marker+3+size]) == "webm"
}

func isISOBaseMedia(head []byte) bool {
	if len(head) < isoBaseMediaHeaderBytes || string(head[4:8]) != "ftyp" {
		return false
	}
	boxSize := int(head[0])<<24 | int(head[1])<<16 | int(head[2])<<8 | int(head[3])
	return boxSize >= isoBaseMediaHeaderBytes
}

func isoBaseMediaUpload(head []byte) (Upload, bool) {
	major := string(head[8:12])
	if major == "qt  " {
		return Upload{Kind: "video", Ext: ".mov", Media: "video/quicktime", Limit: MaxVideoBytes}, true
	}
	compatible, blocked := compatibleBrands(head)
	if blocked || isUnsupportedBrand(major) {
		return Upload{}, false
	}
	if isMP4Brand(major) || compatible {
		return Upload{Kind: "video", Ext: ".mp4", Media: "video/mp4", Limit: MaxVideoBytes}, true
	}
	return Upload{}, false
}

func isMP4Brand(brand string) bool {
	switch brand {
	case "isom", "iso2", "mp41", "mp42", "avc1", "M4V ", "dash":
		return true
	default:
		return false
	}
}

func compatibleBrands(head []byte) (bool, bool) {
	boxSize := int(head[0])<<24 | int(head[1])<<16 | int(head[2])<<8 | int(head[3])
	end := min(min(len(head), boxSize), sniffBytes)
	allowed, blocked := false, false
	for offset := 16; offset+4 <= end; offset += 4 {
		brand := string(head[offset : offset+4])
		allowed = allowed || isMP4Brand(brand)
		blocked = blocked || isUnsupportedBrand(brand)
	}
	return allowed, blocked
}

func isUnsupportedBrand(brand string) bool {
	return strings.HasPrefix(brand, "3gp") || strings.HasPrefix(brand, "heic") ||
		strings.HasPrefix(brand, "heix") || strings.HasPrefix(brand, "hevc") ||
		strings.HasPrefix(brand, "avif") || strings.HasPrefix(brand, "avis") ||
		brand == "mif1" || brand == "msf1"
}
