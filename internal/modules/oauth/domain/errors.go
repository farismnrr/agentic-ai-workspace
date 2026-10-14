package domain

import "errors"

// OAuth 2.x error codes (RFC 6749 §5.2, RFC 7591 §3.2.2, RFC 8707 §2).
const (
	CodeInvalidRequest        = "invalid_request"
	CodeInvalidClient         = "invalid_client"
	CodeInvalidGrant          = "invalid_grant"
	CodeInvalidScope          = "invalid_scope"
	CodeInvalidTarget         = "invalid_target"
	CodeInvalidClientMetadata = "invalid_client_metadata"
	CodeInvalidRedirectURI    = "invalid_redirect_uri"
	CodeUnsupportedGrantType  = "unsupported_grant_type"
	CodeUnsupportedResponse   = "unsupported_response_type"
	CodeAccessDenied          = "access_denied"
	CodeServerError           = "server_error"
)

// Not-found and reuse signals the stores report to the use cases.
var (
	ErrClientNotFound     = errors.New("client not found")
	ErrRequestNotFound    = errors.New("authorization request not found")
	ErrGrantNotFound      = errors.New("grant not found")
	ErrCredentialNotFound = errors.New("credential not found or expired")
	ErrCredentialReused   = errors.New("credential already used")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// Error is a protocol-level OAuth error. When RedirectURI is set it is safe to
// return the error to the client through that redirect (RFC 6749 §4.1.2.1).
type Error struct {
	Code        string
	Description string
	RedirectURI string
	State       string
}

// NewError builds an OAuth error.
func NewError(code, description string) *Error {
	return &Error{Code: code, Description: description}
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Code + ": " + e.Description
}

// Redirected returns a copy that may be delivered via the client's redirect URI.
func (e *Error) Redirected(redirectURI, state string) *Error {
	copied := *e
	copied.RedirectURI = redirectURI
	copied.State = state

	return &copied
}

// AsError extracts an OAuth error from err.
func AsError(err error) (*Error, bool) {
	var oauthErr *Error

	ok := errors.As(err, &oauthErr)

	return oauthErr, ok
}
