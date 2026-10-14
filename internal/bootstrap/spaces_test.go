package bootstrap

import "testing"

func TestSpacesListNeedsSessionAndDescribesTools(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	if anon := do(t, app, "GET", "/api/spaces", "", nil); anon.Status != 401 {
		t.Fatalf("spaces are private to signed-in people, got %d", anon.Status)
	}

	cookie, _ := signedInClient(t, app)
	spaces := do(t, app, "GET", "/api/spaces", "", map[string]string{"Cookie": cookie})

	mustContain(t, spaces.Body, `"url":"`+publicURL+`/api/coding"`, `"name":"read_file"`, `"readOnly":true`,
		`"id":"coding:read"`, "Read files and search code")
}
