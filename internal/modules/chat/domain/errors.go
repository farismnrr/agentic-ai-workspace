package domain

import "errors"

// Errors of the chat use cases.
var (
	ErrNotFound   = errors.New("chat not found")
	ErrBadRequest = errors.New("write what you need in a few words")
	ErrNoProject  = errors.New("project not found")
	ErrNoModel    = errors.New("no model is available; add a provider under Settings, AI models")
)
