package application

import (
	"strings"
	"testing"
)

func TestAFailedFontDownloadIsToldToUseNpm(t *testing.T) {
	t.Parallel()

	failed := ToolResult{IsError: true, Output: "curl: (56) CONNECT tunnel failed fonts.gstatic.com"}
	if !strings.Contains(withHint(failed), "@fontsource-variable") {
		t.Fatal("a font download that failed must point to Fontsource")
	}

	if got := withHint(ToolResult{IsError: true, Output: "boom"}); got != "boom" {
		t.Fatalf("other failures stay as they are: %q", got)
	}

	if got := withHint(ToolResult{Output: "fonts.googleapis.com ok"}); got != "fonts.googleapis.com ok" {
		t.Fatalf("a success is never changed: %q", got)
	}
}
