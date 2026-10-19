package application

import "testing"

func TestTheDocumentsFolderNeedsNeitherSecurityNorDesignReview(t *testing.T) {
	t.Parallel()

	files := []string{"documents/src/deck.py", "documents/index.html", "documents/preview/deck-pptx/page-1.png"}

	if hasCode(files) || hasScreen(files) {
		t.Fatalf("documents/ is output, not an app: code %v, screen %v", hasCode(files), hasScreen(files))
	}

	if !hasCode([]string{"documents/x.pdf", "app/server.py"}) || !hasScreen([]string{"index.html"}) {
		t.Fatal("anything outside documents/ still counts")
	}
}
