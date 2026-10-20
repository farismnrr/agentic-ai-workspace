package mcp

import "context"

// identity is who made a call: the person, and the app they connected.
type identity struct {
	User   string
	Client string
	// Project is the Project ID the call asked for; empty means the app's own project.
	Project string
}

// whoami needs the token's user, which the guard has already verified; the
// handler reads it again from the context the guard stored it in.
type identityKey struct{}

func (t *tools) whoami(ctx context.Context, _ struct{}) (whoamiOutput, report, error) {
	who, _ := ctx.Value(identityKey{}).(identity)
	if who.User == "" {
		return whoamiOutput{}, report{}, errNoIdentity
	}

	return whoamiOutput{ID: who.User}, report{}, nil
}
