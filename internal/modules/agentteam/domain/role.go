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
	// AccessAsk lets a role ask the person questions, with options to pick from or free text.
	AccessAsk = "ask"
	// AccessWeb lets a role search the web and read pages. What it reads comes from strangers, so only a role that
	// cannot change anything has it, and the orchestrator never reads it raw.
	AccessWeb = "web"
	// AccessImages lets a role find pictures with a licence and save them in the project. It does not read web pages:
	// a role that can write and run commands must not read text from strangers.
	AccessImages = "images"
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
	// Tier is the model tier the role uses when it has no model of its own: thinking, default or fast. Empty is
	// the default tier. The person sets what each tier means once, in Settings, Models.
	Tier string `json:"tier"`
	// Skills are the built-in playbooks of a built-in role. They are not saved or sent out: they are read from the
	// app each time.
	Skills []Skill `json:"-"`
	// Library lists the references the role may look up while it works; their text is read when asked for.
	Library []Reference `json:"-"`
	// Access lists the groups of tools the role may use; read is always included.
	Access  []string `json:"access"`
	Lead    bool     `json:"lead"`
	Enabled bool     `json:"enabled"`
	// WritesPRD is true for the built-in role whose document is the team's PRD; no other role's document is
	// saved as one. It is not saved: it follows the built-in role.
	WritesPRD bool `json:"-"`
	// WritesResearch is the same for the role whose documents are the team's research notes.
	WritesResearch bool `json:"-"`
	// ChecksSecurity is true for the built-in role that reviews sensitive changes.
	ChecksSecurity bool `json:"-"`
	// RunsApps is true for the role that alone may start and stop the web app.
	RunsApps bool `json:"-"`
	// ChecksArchitecture is true for the role that reviews affected architecture when required.
	ChecksArchitecture bool `json:"-"`
	// ChecksLook is true for the built-in independent Reviewer.
	ChecksLook bool `json:"-"`
	// LooksAtScreens is true for the Designer, who must look at the running app before reporting screens it made.
	LooksAtScreens bool `json:"-"`
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
	for _, field := range []*string{&r.Key, &r.Name, &r.Purpose, &r.Prompt, &r.ProviderID, &r.Model, &r.Effort, &r.Tier} {
		*field = strings.TrimSpace(*field)
	}

	if !r.fits() {
		return Role{}, ErrInvalid
	}

	if r.ProviderID == "" || r.Model == "" {
		r.ProviderID, r.Model = "", ""
	}

	r.Tier = cleanTier(r.Tier, r.ProviderID != "")

	access, err := cleanAccess(r.Access)
	if err != nil {
		return Role{}, err
	}

	r.Access = withoutWebWhenItCanChange(access)

	built, isBuiltIn := presetByKey(r.Key)
	// The orchestrator is the only lead, and a built-in role keeps the access that fits it.
	r.Lead = isBuiltIn && built.lead

	r.Skills, r.Library, r.WritesPRD, r.WritesResearch = nil, nil, false, false
	r.ChecksSecurity, r.ChecksArchitecture, r.ChecksLook, r.LooksAtScreens = false, false, false, false
	r.RunsApps = false

	if isBuiltIn {
		r.Access = slices.Clone(built.access)
		r.Skills, r.Library = SkillsFor(built.key), LibraryFor(built.key)
		r.WritesPRD, r.WritesResearch = built.prd, built.research
		r.ChecksSecurity, r.ChecksArchitecture = built.security, built.architecture
		r.ChecksLook, r.LooksAtScreens = built.look, built.designs
		r.RunsApps = built.runs
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
