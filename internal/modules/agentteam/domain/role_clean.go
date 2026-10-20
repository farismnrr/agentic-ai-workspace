package domain

import "slices"

func cleanAccess(groups []string) ([]string, error) {
	clean := []string{AccessRead}

	for _, group := range groups {
		switch group {
		case AccessRead:
		case AccessWrite, AccessExec, AccessAsk, AccessWeb, AccessImages:
			if !slices.Contains(clean, group) {
				clean = append(clean, group)
			}
		default:
			return nil, ErrInvalid
		}
	}

	return clean, nil
}

// withoutWebWhenItCanChange drops web access from a role that can also change files or run commands: text from
// strangers must not reach a role that could act on it. Pictures (a separate group) stay.
func withoutWebWhenItCanChange(access []string) []string {
	changes := slices.Contains(access, AccessWrite) || slices.Contains(access, AccessExec)
	if !slices.Contains(access, AccessWeb) || !changes {
		return access
	}

	return slices.DeleteFunc(slices.Clone(access), func(group string) bool { return group == AccessWeb })
}

// cleanTier keeps a known tier for a role with no model of its own; a role with its own model has no tier.
func cleanTier(tier string, hasOwnModel bool) string {
	if hasOwnModel || (tier != "thinking" && tier != "fast") {
		return ""
	}

	return tier
}
