package application

import "strings"

// appendResearch keeps recent rounds while preserving their order.
func appendResearch(current, document string) string {
	all := document
	if current != "" {
		all = current + researchBreak + document
	}
	for len([]rune(all)) > maxResearch {
		cut := strings.Index(all, researchBreak)
		if cut < 0 {
			break
		}
		all = all[cut+len(researchBreak):]
	}
	return truncate(all, maxResearch)
}
