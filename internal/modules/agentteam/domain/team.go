package domain

import "strconv"

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

// Normalize validates the team: 1 to MaxRoles roles, exactly one lead, unique keys.
func (t Team) Normalize() (Team, error) {
	if len(t.Roles) == 0 || len(t.Roles) > MaxRoles {
		return Team{}, ErrInvalid
	}

	roles, taken, err := cleanRoles(t.Roles)
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
