package application

// holds counts how often the server held something back during one run, so that every hold is bounded and a
// check that cannot be satisfied never stops the run from ending.
type holds struct {
	// planWarned is set once the orchestrator was told its plan was out of date, so it is told only once.
	planWarned bool
	// siteAsked counts finishes held back because a page was built and no app was running.
	siteAsked int
	// localAsked counts finishes sent back for naming 127.0.0.1 or localhost.
	localAsked int
	// planAsked counts jobs held back because the person's latest request was not in the plan yet.
	planAsked int
	// securityAsked counts finishes held back for the Security check of the final code.
	securityAsked int
	// architectAsked counts finishes held back for the Architect's check of the final code.
	architectAsked int
	// reviewerAsked counts finishes held back for the Reviewer's look at the screens.
	reviewerAsked int
	// lookAsked counts reports sent back because a teammate had not looked at the screens it made.
	lookAsked int
}
