package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/adapter/runtime"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	requestTimeout = 5 * time.Minute
	maxResponse    = 64 << 20
)

// relayRequest is one request for the app, already allowed: the person's browser sends it through the proxy, and
// the preview the AI looks at sends it through the viewer.
type relayRequest struct {
	Header   http.Header
	Socket   string
	Method   string
	Path     string
	Query    string
	Base     string
	Prefix   string
	ClientIP string
	Body     []byte
	Preview  bool
}

// relayAnswer is what the browser gets: the app's answer, filtered and with its links under the prefix.
type relayAnswer struct {
	Header http.Header
	Body   []byte
	Status int
}

// relayFailure is an answer the proxy makes itself because the app gave none.
type relayFailure struct {
	text   string
	status int
}

// relay sends the request to the app through its socket. The app sees the path without the /run/<name> prefix.
func relay(ctx context.Context, in relayRequest) (relayAnswer, *relayFailure) {
	target := url.URL{Scheme: "http", Host: "site", Path: in.Path, RawQuery: in.Query}

	req, err := http.NewRequestWithContext(ctx, in.Method, target.String(), bytes.NewReader(in.Body))
	if err != nil {
		return relayAnswer{}, &relayFailure{status: http.StatusBadRequest, text: "Bad request"}
	}

	req.Header = forSiteRequest(in.Header, in.Prefix, in.ClientIP, in.Preview)
	req.Host = "site"

	resp, err := runtime.Client(in.Socket, requestTimeout).Do(req)
	if err != nil {
		return relayAnswer{}, &relayFailure{status: http.StatusBadGateway, text: "The app is not answering."}
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(body) > maxResponse {
		return relayAnswer{}, &relayFailure{status: http.StatusBadGateway, text: "The app's answer was too large."}
	}

	kind, encoding := resp.Header.Get("Content-Type"), resp.Header.Get("Content-Encoding")
	body = domain.Localize(kind, encoding, in.Base, in.Prefix, body)

	return relayAnswer{Status: resp.StatusCode, Header: forSiteResponse(resp.Header, in.Base, in.Prefix, in.Preview), Body: body}, nil
}
