// Package githubapi talks to GitHub's sign-in and REST API. The addresses are fixed, never chosen
// by a person, so the calls do not need the egress guard.
package githubapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

const (
	webBase    = "https://github.com"
	apiBase    = "https://api.github.com"
	apiVersion = "2022-11-28"
	timeout    = 15 * time.Second
	maxBody    = 1 << 20
)

var errUnavailable = errors.New("github is unavailable")

// Client is the GitHub App the server signs people in with.
type Client struct {
	http     *http.Client
	now      func() time.Time
	id       string
	secret   string
	callback string
	web      string
	api      string
}

// New builds a client; callback is the address GitHub sends people back to.
func New(clientID, clientSecret, callback string, now func() time.Time) *Client {
	return &Client{
		id: clientID, secret: clientSecret, callback: callback, web: webBase, api: apiBase, now: now,
		http: &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
	}
}

// AuthorizeURL is where the person approves the connection.
func (c *Client) AuthorizeURL(state, challenge string) string {
	q := url.Values{
		"client_id": {c.id}, "redirect_uri": {c.callback}, "state": {state},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}

	return c.web + "/login/oauth/authorize?" + q.Encode()
}

type tokenReply struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshIn    int64  `json:"refresh_token_expires_in"`
}

// Exchange trades the code GitHub sent back for tokens.
func (c *Client) Exchange(ctx context.Context, code, verifier string) (domain.Tokens, error) {
	return c.token(ctx, url.Values{"code": {code}, "redirect_uri": {c.callback}, "code_verifier": {verifier}})
}

// Refresh trades a refresh token for new tokens; the old pair stops working.
func (c *Client) Refresh(ctx context.Context, refresh string) (domain.Tokens, error) {
	return c.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}})
}

func (c *Client) token(ctx context.Context, form url.Values) (domain.Tokens, error) {
	form.Set("client_id", c.id)
	form.Set("client_secret", c.secret)

	body := strings.NewReader(form.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.web+"/login/oauth/access_token", body)
	if err != nil {
		return domain.Tokens{}, fmt.Errorf("build token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var reply tokenReply

	status, err := c.do(req, &reply)
	if err != nil {
		return domain.Tokens{}, err
	}

	if status >= http.StatusInternalServerError {
		return domain.Tokens{}, fmt.Errorf("%w: status %d", errUnavailable, status)
	}

	if reply.Error != "" || status >= http.StatusBadRequest || reply.AccessToken == "" {
		return domain.Tokens{}, fmt.Errorf("%w: %s", domain.ErrRejected, reply.Error)
	}

	return c.tokens(reply), nil
}

func (c *Client) tokens(r tokenReply) domain.Tokens {
	t := domain.Tokens{Access: r.AccessToken, Refresh: r.RefreshToken}
	if r.ExpiresIn > 0 {
		t.AccessExpires = c.now().Add(time.Duration(r.ExpiresIn) * time.Second)
	}

	if r.RefreshIn > 0 {
		t.RefreshExpires = c.now().Add(time.Duration(r.RefreshIn) * time.Second)
	}

	return t
}

// Profile reads the account a token belongs to.
func (c *Client) Profile(ctx context.Context, access string) (domain.Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.api+"/user", nil)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("build profile request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+access)

	var user struct {
		Login string `json:"login"`
		Name  string `json:"name"`
		ID    int64  `json:"id"`
	}

	status, err := c.do(req, &user)
	if err != nil {
		return domain.Profile{}, err
	}

	if status == http.StatusUnauthorized || status == http.StatusForbidden || user.ID == 0 {
		return domain.Profile{}, domain.ErrRejected
	}

	return domain.Profile{ID: user.ID, Login: user.Login, Name: user.Name}, nil
}

// Revoke asks GitHub to forget the person's authorization of the app.
func (c *Client) Revoke(ctx context.Context, access string) error {
	body, _ := json.Marshal(map[string]string{"access_token": access})

	target := c.api + "/applications/" + url.PathEscape(c.id) + "/grant"

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("build revoke request: %w", err)
	}

	req.SetBasicAuth(c.id, c.secret)

	_, err = c.do(req, nil)

	return err
}

// do sends the request with GitHub's headers and decodes a JSON answer into out.
func (c *Client) do(req *http.Request, out any) (int, error) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("User-Agent", "agentic-ai-workspace")

	res, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call github: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return res.StatusCode, fmt.Errorf("read github answer: %w", err)
	}

	if out != nil {
		_ = json.Unmarshal(raw, out)
	}

	return res.StatusCode, nil
}
