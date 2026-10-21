package domain

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

var ErrInvalidDatabaseIdentifier = errors.New("invalid database identifier")

// ProjectDatabaseIdentifier returns a stable, database-safe name for a project.
func ProjectDatabaseIdentifier(projectID string) string {
	digest := sha256.Sum256([]byte(projectID))
	return "p_" + fmt.Sprintf("%x", digest[:12])
}

// ValidDatabaseIdentifier accepts identifiers safe to quote in provisioning DDL.
func ValidDatabaseIdentifier(value string) bool {
	if len(value) == 0 || len(value) > 64 || !validIdentifierStart(value[0]) {
		return false
	}
	for _, char := range value[1:] {
		if !validIdentifierPart(char) {
			return false
		}
	}
	return true
}

func validIdentifierStart(char byte) bool { return char >= 'a' && char <= 'z' }

func validIdentifierPart(char rune) bool {
	return char == '_' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}
