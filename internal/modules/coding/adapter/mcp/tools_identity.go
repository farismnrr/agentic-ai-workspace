package mcp

import "context"

// whoami needs the token's user, which the guard has already verified; the
// handler reads it again from the context the guard stored it in.
type identityKey struct{}

func (t *tools) whoami(ctx context.Context, _ struct{}) (whoamiOutput, report, error) {
	id, _ := ctx.Value(identityKey{}).(string)
	if id == "" {
		return whoamiOutput{}, report{}, errNoIdentity
	}

	return whoamiOutput{ID: id}, report{}, nil
}
