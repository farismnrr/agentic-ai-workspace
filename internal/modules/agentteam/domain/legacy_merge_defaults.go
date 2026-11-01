package domain

// These are exact historical shipped defaults, NOT fuzzy text matching.
// User-authored instructions must survive every built-in role upgrade.
const (
	previousPlannerPrompt = "You handle business analysis and planning. Clarify requirements, users, risks and acceptance " +
		"criteria, then write a concise PRD and steps. Do not edit code, run tests or deploy."
	previousSecurityPrompt = "You are an application security reviewer. Check the changed security boundary and its relevant " +
		"critical scenarios. Broaden dependency, static and failure-path checks when impact or saved task mode " +
		"requires it. Report exact checks, findings and uncertainty in plain words. Do not run unrelated full suites " +
		"for an ordinary change. Start your report with exactly Verdict: PASS or Verdict: FIX FIRST. Do not change " +
		"the code yourself."
	previousPlannerPurpose  = "Gathers requirements and acceptance criteria, then writes a short PRD and plan."
	previousSecurityPurpose = "Checks sensitive changes for vulnerabilities, risky code and missing protections."
)

func previousBuiltInDefaults(roleKey string) (prompt, purpose string) {
	switch roleKey {
	case "planner":
		return previousPlannerPrompt, previousPlannerPurpose
	case "security":
		return previousSecurityPrompt, previousSecurityPurpose
	default:
		return "", ""
	}
}
