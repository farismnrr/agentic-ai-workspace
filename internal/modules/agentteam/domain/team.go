package domain

import (
	"slices"
	"strconv"
)

// Team is a person's agent team: whether new tasks use it, and its roles in display order.
type Team struct {
	Roles        []Role `json:"roles"`
	UseByDefault bool   `json:"useByDefault"`
}

// Lead is the orchestrator, the role that talks to the person and hands work out.
func (t Team) Lead() (Role, bool) {
	for _, role := range t.Roles {
		if role.Lead {
			return role, true
		}
	}

	return Role{}, false
}

// Members are the enabled roles the lead can hand work to.
func (t Team) Members() []Role {
	members := []Role{}

	for _, role := range t.Roles {
		if role.Enabled && !role.Lead {
			members = append(members, role)
		}
	}

	return members
}

// WithBuiltIns is the team with every built-in role on it: any the person's team lacks is added back with
// its defaults, and the ones it has get the current access and playbook. Built-in roles are the same for
// everyone and cannot be removed, only turned off or edited.
func (t Team) WithBuiltIns() Team {
	roles := slices.Clone(t.Roles)

	for _, built := range Presets().Roles {
		at := slices.IndexFunc(roles, func(role Role) bool { return role.Key == built.Key })
		if at < 0 {
			roles = append(roles, built)

			continue
		}

		// A role saved before an update keeps its prompt and model, but gets the access and playbook that ship now.
		roles[at].Access = slices.Clone(built.Access)
		roles[at].Skills, roles[at].Library = built.Skills, built.Library
	}

	return Team{Roles: roles, UseByDefault: t.UseByDefault}
}

// Normalize validates the team: it has every built-in role, at most MaxRoles roles, exactly one lead and
// unique keys.
func (t Team) Normalize() (Team, error) {
	if len(t.Roles) == 0 {
		return Team{}, ErrInvalid
	}

	full := t.WithBuiltIns()
	if len(full.Roles) > MaxRoles {
		return Team{}, ErrInvalid
	}

	roles, taken, err := cleanRoles(full.Roles)
	if err != nil {
		return Team{}, err
	}

	for i := range roles {
		if roles[i].Key == "" {
			roles[i].Key = uniqueKey(slug(roles[i].Name), taken)
		}
	}

	return Team{Roles: roles, UseByDefault: t.UseByDefault}, nil
}

// cleanRoles validates each role and checks the team has exactly one lead and no repeated key.
// It returns the keys in use, so roles without one can be given a key of their own.
func cleanRoles(roles []Role) ([]Role, map[string]bool, error) {
	clean, taken, leads := make([]Role, 0, len(roles)), map[string]bool{}, 0

	for _, role := range roles {
		one, err := role.normalize()
		if err != nil || (one.Key != "" && taken[one.Key]) {
			return nil, nil, ErrInvalid
		}

		taken[one.Key] = one.Key != ""

		if one.Lead {
			leads++
		}

		clean = append(clean, one)
	}

	if leads != 1 {
		return nil, nil, ErrInvalid
	}

	return clean, taken, nil
}

func uniqueKey(base string, taken map[string]bool) string {
	key := base

	for n := 2; taken[key]; n++ {
		key = base + "-" + strconv.Itoa(n)
	}

	taken[key] = true

	return key
}
