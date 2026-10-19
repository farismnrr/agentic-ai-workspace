package sqlite

import "strings"

// isSourceTaken spots the unique index that allows one live project per connected app.
func isSourceTaken(err error) bool {
	return err != nil && strings.Contains(err.Error(), "projects.user_id, projects.source")
}

// isSlugTaken spots the unique index on project IDs, which catches two people choosing the same one at once.
func isSlugTaken(err error) bool {
	return err != nil && strings.Contains(err.Error(), "projects.slug")
}
