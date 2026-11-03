package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

// ScreenshotReference binds an opaque task screenshot name to its Site start generation.
func ScreenshotReference(id string, siteVersion int64, workID string) string {
	return id + "~" + strconv.FormatInt(siteVersion, 36) + "~" + screenshotWorkTag(workID)
}

// ParseScreenshotReference returns a stored name and Site generation from a browser screenshot reference.
func ParseScreenshotReference(ref, workID string) (string, int64, bool) {
	parts := strings.Split(ref, "~")
	if len(parts) != 3 || len(parts[0]) != 28 || parts[2] != screenshotWorkTag(workID) {
		return "", 0, false
	}
	version, err := strconv.ParseInt(parts[1], 36, 64)
	return parts[0], version, err == nil && version > 0
}

func screenshotWorkTag(workID string) string {
	if workID == "" {
		return "internal"
	}
	digest := sha256.Sum256([]byte(workID))
	return hex.EncodeToString(digest[:4])
}
