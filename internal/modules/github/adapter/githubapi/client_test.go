package githubapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := New("cid", "csecret", "https://app.test/cb", func() time.Time { return time.Unix(1000, 0) })
	c.web, c.api = srv.URL, srv.URL

	return c
}

func TestAuthorizeURLUsesPKCEAndState(t *testing.T) {
	c := New("cid", "s", "https://app.test/cb", time.Now)
	u, _ := url.Parse(c.AuthorizeURL("st", "ch"))
	q := u.Query()

	if u.Host != "github.com" || q.Get("client_id") != "cid" || q.Get("state") != "st" ||
		q.Get("code_challenge") != "ch" || q.Get("code_challenge_method") != "S256" ||
		q.Get("redirect_uri") != "https://app.test/cb" {
		t.Fatalf("authorize url: %s", u)
	}
}

func TestExchangeReadsExpiringTokens(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code_verifier") != "ver" || r.Form.Get("client_secret") != "csecret" {
			http.Error(w, "bad", http.StatusBadRequest)

			return
		}

		_, _ = w.Write([]byte(`{"access_token":"ghu_a","refresh_token":"ghr_r",` +
			`"expires_in":28800,"refresh_token_expires_in":15897600}`))
	})

	got, err := c.Exchange(context.Background(), "code", "ver")
	if err != nil || got.Access != "ghu_a" || got.Refresh != "ghr_r" ||
		!got.AccessExpires.Equal(time.Unix(1000+28800, 0)) {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestRefreshErrorsAreRejectedButOutagesAreNot(t *testing.T) {
	bad := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"bad_refresh_token"}`))
	})
	if _, err := bad.Refresh(context.Background(), "x"); !errors.Is(err, domain.ErrRejected) {
		t.Fatalf("want rejected, got %v", err)
	}

	down := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "oops", http.StatusBadGateway)
	})
	if _, err := down.Refresh(context.Background(), "x"); err == nil || errors.Is(err, domain.ErrRejected) {
		t.Fatalf("an outage must not look like a revoked grant: %v", err)
	}
}

func TestProfileSendsBearerAndMaps401(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer good") {
			http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)

			return
		}

		_, _ = w.Write([]byte(`{"id":42,"login":"ada","name":null}`))
	})

	p, err := c.Profile(context.Background(), "good")
	if err != nil || p.ID != 42 || p.Login != "ada" || p.NoreplyEmail() != "42+ada@users.noreply.github.com" {
		t.Fatalf("%+v %v", p, err)
	}

	if _, err = c.Profile(context.Background(), "bad"); !errors.Is(err, domain.ErrRejected) {
		t.Fatalf("want rejected, got %v", err)
	}
}
