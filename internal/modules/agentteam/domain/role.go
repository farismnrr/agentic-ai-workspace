package domain

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// Limits keep the instructions that reach the model small and sane.
const (
	MaxRoles   = 20
	MaxName    = 40
	MaxPurpose = 200
	MaxPrompt  = 2000
	MaxShort   = 120
)

// ErrInvalid means a role or the team is too long, has too many entries or misses its lead.
var ErrInvalid = errors.New("the agent team is not valid")

// Access groups: what a role may do. They match the tool groups of the coding space.
const (
	AccessRead  = "read"
	AccessWrite = "write"
	AccessExec  = "exec"
)

var (
	keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)
	slugRuns   = regexp.MustCompile(`[^a-z0-9]+`)
)

// Role is one member of the team: who it is, which model it uses and what it may touch.
type Role struct {
	// Key names the role to the orchestrator; it is a lowercase slug, unique within a team.
	Key     string `json:"key"`
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	// Prompt is the role's personality and working style, added to its instructions.
	Prompt string `json:"prompt"`
	// ProviderID, Model and Effort pick the role's model; empty means the person's default.
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
	Effort     string `json:"effort"`
	// Access lists the groups of tools the role may use; read is always included.
	Access  []string `json:"access"`
	Lead    bool     `json:"lead"`
	Enabled bool     `json:"enabled"`
}

// Preset reports whether the role is one of the built-in SDLC roles.
func (r Role) Preset() bool {
	_, ok := presetByKey(r.Key)

	return ok
}

// Can reports whether the role may use an access group.
func (r Role) Can(group string) bool {
	return r.Lead || slices.Contains(r.Access, group)
}

func (r Role) normalize() (Role, error) {
	for _, field := range []*string{&r.Key, &r.Name, &r.Purpose, &r.Prompt, &r.ProviderID, &r.Model, &r.Effort} {
		*field = strings.TrimSpace(*field)
	}

	if !r.fits() {
		return Role{}, ErrInvalid
	}

	if r.ProviderID == "" || r.Model == "" {
		r.ProviderID, r.Model = "", ""
	}

	access, err := cleanAccess(r.Access)
	if err != nil {
		return Role{}, err
	}

	r.Access = access

	built, isBuiltIn := presetByKey(r.Key)
	// The orchestrator is the only lead, and a built-in role keeps the access that fits it.
	r.Lead = isBuiltIn && built.lead

	if isBuiltIn {
		r.Access = slices.Clone(built.access)
	}

	r.Enabled = r.Enabled || r.Lead

	return r, nil
}

// fits reports whether every field is within its limit and the key, when there is one, is a slug.
func (r Role) fits() bool {
	if r.Name == "" || tooLong(MaxName, r.Name) || tooLong(MaxPurpose, r.Purpose) || tooLong(MaxPrompt, r.Prompt) {
		return false
	}

	if tooLong(MaxShort, r.ProviderID, r.Model, r.Effort) {
		return false
	}

	return r.Key == "" || keyPattern.MatchString(r.Key)
}

func cleanAccess(groups []string) ([]string, error) {
	clean := []string{AccessRead}

	for _, group := range groups {
		switch group {
		case AccessRead:
		case AccessWrite, AccessExec:
			if !slices.Contains(clean, group) {
				clean = append(clean, group)
			}
		default:
			return nil, ErrInvalid
		}
	}

	return clean, nil
}

func tooLong(limit int, values ...string) bool {
	for _, value := range values {
		if utf8.RuneCountInString(value) > limit {
			return true
		}
	}

	return false
}

// slug turns a name into a key.
func slug(name string) string {
	key := strings.Trim(slugRuns.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if key == "" {
		key = "role"
	}

	if len(key) > MaxName {
		key = strings.Trim(key[:MaxName], "-")
	}

	return key
}
