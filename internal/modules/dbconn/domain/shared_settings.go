package domain

import "errors"

var ErrInvalidSharedSettings = errors.New("invalid shared database settings")

var SSLModes = [...]string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

// SharedSettings contains the admin connection used to provision project schemas.
type SharedSettings struct {
	Host, Database, User, Password, SSLMode string
	Port                                    int
}

func (s SharedSettings) Validate() error {
	if s.Host == "" || s.Database == "" || s.User == "" || s.Port < 1 || s.Port > 65535 {
		return ErrInvalidSharedSettings
	}
	for _, mode := range SSLModes {
		if s.SSLMode == mode {
			return nil
		}
	}
	return ErrInvalidSharedSettings
}
