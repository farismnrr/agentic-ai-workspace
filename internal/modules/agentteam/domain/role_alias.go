package domain

// CanonicalRoleKey maps retired built-in roles to their replacement without
// rewriting historical task events. This is not a new permission grant.
func CanonicalRoleKey(key string) string {
	switch key {
	case "designer", "media", "documents":
		return "developer"
	case "reviewer":
		return "qa"
	case "business-analyst":
		return "planner"
	default:
		return key
	}
}

// roleSkillKeys keeps the specialized craft playbooks after consolidation.
func roleSkillKeys(key string) []string {
	switch key {
	case "developer":
		return []string{"developer", "designer", "media", "documents"}
	case "qa":
		return []string{"qa", "reviewer"}
	case "planner":
		return []string{"planner", "business-analyst"}
	default:
		return []string{key}
	}
}
