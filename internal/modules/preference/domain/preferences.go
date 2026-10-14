package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Limits keep the instructions that reach the model small and sane.
const (
	MaxRules      = 20
	MaxRuleLength = 200
	MaxShort      = 120
	MaxAbout      = 1000
)

// ErrInvalid means a preference is too long or has too many entries.
var ErrInvalid = errors.New("a preference is too long or there are too many rules")

// Rule is an instruction the person wants the agent to follow.
type Rule struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	On   bool   `json:"on"`
}

// Preferences are a person's defaults and the instructions given to the agent.
type Preferences struct {
	DefaultProviderID string
	DefaultModel      string
	PersonaName       string
	TalkStyle         string
	Language          string
	About             string
	Rules             []Rule
}

// Normalize trims fields and validates their sizes.
func (p Preferences) Normalize() (Preferences, error) {
	short := []*string{&p.DefaultProviderID, &p.DefaultModel, &p.PersonaName, &p.TalkStyle, &p.Language}
	for _, field := range short {
		*field = strings.TrimSpace(*field)
	}

	p.About = strings.TrimSpace(p.About)

	if tooLong(MaxShort, p.DefaultProviderID, p.DefaultModel, p.PersonaName, p.TalkStyle, p.Language) ||
		tooLong(MaxAbout, p.About) || len(p.Rules) > MaxRules {
		return Preferences{}, ErrInvalid
	}

	rules, err := cleanRules(p.Rules)
	if err != nil {
		return Preferences{}, err
	}

	p.Rules = rules

	return p, nil
}

func tooLong(limit int, values ...string) bool {
	for _, value := range values {
		if utf8.RuneCountInString(value) > limit {
			return true
		}
	}

	return false
}

func cleanRules(rules []Rule) ([]Rule, error) {
	clean := make([]Rule, 0, len(rules))

	for _, rule := range rules {
		rule.Text = strings.TrimSpace(rule.Text)

		if tooLong(MaxRuleLength, rule.Text) {
			return nil, ErrInvalid
		}

		if rule.Text != "" {
			clean = append(clean, rule)
		}
	}

	return clean, nil
}
