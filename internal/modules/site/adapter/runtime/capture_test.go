package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// A running app is photographed by a browser in a container of its own, which reaches the app through the site's
// socket. The picture is a real PNG and the page facts come from the browser.
func TestARunningAppIsPhotographedThroughItsSocket(t *testing.T) {
	rt := newRuntime(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

	defer cancel()

	id, err := rt.Start(ctx, application.StartSpec{
		ProjectID: "p1", BasePath: "/run/demo/abc", Port: 8000,
		Command: `echo '<!doctype html><title>Kopi</title><h1>Halo dunia</h1><img src="x.png">` +
			`<svg width="1280" height="800"><filter id="n"><feTurbulence baseFrequency=".9"/></filter>` +
			`<rect width="100%" height="100%" filter="url(#n)"/></svg>' > index.html && ` +
			`exec python3 -m http.server "$PORT" --bind 127.0.0.1`,
	})
	if err != nil {
		t.Fatal(err)
	}

	defer rt.Stop(ctx, id)

	output, err := rt.Capture(ctx, application.CaptureRequest{
		ProjectID: "p1", Port: 8000, Spec: domain.CaptureSpec{Path: "/", Mobile: false},
	})
	if err != nil {
		t.Fatal(err)
	}

	shot, err := application.ParseCapture(output)
	if err != nil {
		t.Fatalf("%v\n%.500s", err, output)
	}

	// Noise makes a picture far bigger than a pipe holds, which once cut every desktop screenshot short.
	if len(shot.PNG) < 150_000 {
		t.Fatalf("the test page should make a large picture, got %d bytes", len(shot.PNG))
	}

	checkPicture(t, shot)
}

func checkPicture(t *testing.T, shot domain.Capture) {
	t.Helper()

	if len(shot.PNG) < 8 || string(shot.PNG[1:4]) != "PNG" || shot.Page.Title != "Kopi" || shot.Frame.Width != 1280 {
		t.Fatalf("capture = title %q, frame %+v, %d bytes", shot.Page.Title, shot.Frame, len(shot.PNG))
	}

	if shot.Page.Images.Broken != 1 || shot.Page.Images.NoAlt != 1 || shot.Page.ViewportMeta {
		t.Fatalf("the page facts are read in the browser: %+v", shot.Page)
	}
}
