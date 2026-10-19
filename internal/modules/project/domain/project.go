package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits on what a person can type.
const (
	maxName        = 80
	maxDescription = 500
)

// Errors of the project use cases.
var (
	ErrInvalidName        = errors.New("give the project a name of up to 80 characters")
	ErrInvalidDescription = errors.New("the description can have up to 500 characters")
	ErrNotFound           = errors.New("project not found")
	ErrInvalidChecks      = errors.New("those checks are not valid")
	ErrSourceTaken        = errors.New("a project for that app already exists")
	ErrNameTaken          = errors.New("another project already has that name")
)

// Project is a piece of work with its own folder of files.
type Project struct {
	CreatedAt   time.Time
	ArchivedAt  *time.Time
	ID          string
	UserID      string
	Name        string
	Description string
	// Source names the connected app that owns the project, such as ChatGPT; it is empty for
	// projects the person made themselves.
	Source string
}

// Normalize trims and validates the editable fields.
func Normalize(name, description string) (string, string, error) {
	name, description = strings.TrimSpace(name), strings.TrimSpace(description)

	if name == "" || utf8.RuneCountInString(name) > maxName {
		return "", "", ErrInvalidName
	}

	if utf8.RuneCountInString(description) > maxDescription {
		return "", "", ErrInvalidDescription
	}

	return name, description, nil
}
