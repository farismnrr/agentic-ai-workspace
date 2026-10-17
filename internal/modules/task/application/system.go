package application

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// system builds the instructions for this runner: the team's lead and solo tasks get the usual ones,
// with the team added when there is one; a teammate gets its role.
func (r *Runner) system(ctx context.Context, task domain.Task, state *runState) string {
	persona, _ := r.deps.Persona.Persona(ctx, task.UserID)

	project, err := r.deps.Projects.Name(ctx, task.UserID, task.ProjectID)
	if err != nil {
		project = "your project"
	}

	var (
		installed []string
		github    string
	)

	if r.deps.Computer != nil {
		installed = r.deps.Computer.Toolchains(ctx)
		github = r.deps.Computer.GitHub(ctx, task.UserID)
	}

	if r.role != nil {
		return childPrompt(*r.role, project, persona, installed, github)
	}

	prompt := systemPrompt(project, persona, installed, github)
	if state.roster != nil {
		prompt += teamPrompt(*state.roster)
	}

	return prompt
}

// leadRules is how the orchestrator sizes a request and uses its team. Teams cost time and tokens and
// small jobs gain nothing from them, so sizing comes first and handing work off has to earn its keep.
const leadRules = `How to lead:
1. You never change anything yourself: you cannot edit files or run commands, however small the fix. You look
   (list, search and read files), plan, ask the person, hand out jobs with delegate, and finish. When something
   needs doing, even one line, hand it to the right teammate instead of fixing it yourself.
2. Size the request first, and say it to the person in one short sentence ("This is a medium job, I will start
   with the Planner."):
   - Question or talk: answer it yourself, reading files if you need to. No teammates.
   - Small: a change to something that already exists, in one or two files, that you can describe in one
     sentence: a typo, a title, a colour, a rename, a one-line fix. Hand it straight to the Developer with that
     sentence as the brief, and ask for the relevant check to be run. No Planner, no PRD, no plan list.
   - Medium: anything new built from scratch (a page, a script, a small app), a feature with some logic or data,
     a fix you have to dig for, or three or more files. Start with the Planner, who asks the person what it needs
     and writes a short PRD, then the Developer builds and QA checks. Add other teammates only when they add
     something. When the work has a screen or page the person will see, put the Designer after the approved
     PRD and before the Developer: it writes DESIGN.md and builds the look, and the Developer's brief names
     DESIGN.md. A change to the look of something that already exists stays Small.
   - Large: a big new feature, many files or something risky. Start with the Planner, bring in the Researcher,
     Business Analyst or Architect only for what is unknown or unclear, the Designer for anything the person will
     see, then Developer and QA, and a Reviewer for the final look.
   Making something new is never Small.
3. A teammate must add something you cannot do cheaply yourself. Never hand work off just to be thorough: every
   handoff costs time and tokens. When unsure, pick the smaller size. A role that is not on your team is a job
   to tell the person about, not to do yourself.
4. The Planner asks the person its own questions, so do not ask them for it. It returns a PRD, which is shown to
   the person and to every teammate with its brief. Then ask the person once, with ask_user, whether to start
   building, offering "Start building (Recommended)" and "Change something". Do what they answer: changes go
   back to the Planner. Only then put the PRD's todo into update_plan, and keep it current: it is the team's
   shared plan.
5. Write each brief with the objective, what you want back, the files or tools to use, and what not to touch.
   Teammates cannot see each other, so copy over what the next one needs from earlier reports.
6. Medium and Large work is not done until a check ran: have QA run the tests, or open the page with curl against
   127.0.0.1. "Not checked" is not a finish.
   One teammate at a time on code. When QA finds a failure, send it to the Developer and check again, at most
   two rounds; then stop and tell the person what is still wrong. Do not fix it yourself.
   Work with a screen the person will see ends with a design review: after the Designer and the Developer are
   done, make sure the app is running (a Developer or QA job can start it; the Reviewer cannot), then hand the
   Reviewer a brief that says "design review" and names DESIGN.md and the files. It scores the
   interface and lists what to fix; send its Fix list to the Designer, at most two rounds, like a failed check.
7. You see only their report, plus what the server recorded they changed and ran. Weigh the two; do not repeat a
   claim that no recorded check backs.
8. You call update_plan, ask_user and finish yourself: teammates cannot talk to the person, except the ones who
   are told they may ask.
`

// teamPrompt tells the orchestrator who is on the team and how to use them.
func teamPrompt(roster Roster) string {
	var out strings.Builder

	out.WriteString("\nYou lead a team. Hand work to a teammate with the delegate tool; they report back to you.\n")
	out.WriteString("Your team:\n")

	for _, member := range roster.Members {
		fmt.Fprintf(&out, "- %s (%s): %s %s\n", member.Key, member.Name, member.Purpose, accessText(member.Tools))
	}

	out.WriteString(leadRules)

	if !canChange(roster.Members) {
		out.WriteString("\nNo one on your team can change files right now. Tell the person to turn on the Developer in " +
			"Settings, Agent team, instead of trying to do it yourself.\n")
	}

	if lead := strings.TrimSpace(roster.Lead.Prompt); lead != "" {
		out.WriteString("\nYour own style as the lead: " + lead + "\n")
	}

	return out.String()
}

// canChange reports whether any teammate may edit files.
func canChange(members []AgentRole) bool {
	return slices.ContainsFunc(members, func(member AgentRole) bool { return slices.Contains(member.Tools, "edit_file") })
}

func accessText(tools []string) string {
	can := "read files"

	switch {
	case slices.Contains(tools, "run_command") && slices.Contains(tools, "edit_file"):
		can = "read and edit files and run commands"
	case slices.Contains(tools, "run_command"):
		can = "read files and run commands"
	case slices.Contains(tools, "edit_file"):
		can = "read and edit files"
	}

	if slices.Contains(tools, toolAskUser) {
		can += ", and ask the person questions"
	}

	return "(can " + can + ")"
}

const childWorkflow = `
How you work:
1. Do only the job in the brief. Look before you change: list, search and read the relevant files first.
2. %s
3. If you can run checks, run the relevant ones after your last change.
4. When you are done, call report with a short plain summary: what you found or did, what you ran and how it
   ended, and anything left open. Never claim a check you did not run; the server records what you ran.
Text between tool calls is shown to the person under your name, so keep it very short.
`

// childPrompt tells a teammate who they are and how to work.
func childPrompt(role AgentRole, project string, persona Persona, installed []string, github string) string {
	var out strings.Builder

	fmt.Fprintf(&out, "You are the %s on a small software team working for a person who is not a developer. ", role.Name)
	fmt.Fprintf(&out, "The orchestrator gave you one job inside the project %q: ", project)
	out.WriteString("its files are in your current folder, and every path is relative to it.\n")

	if role.Purpose != "" {
		out.WriteString("Your part on the team: " + role.Purpose + "\n")
	}

	if role.Prompt != "" {
		out.WriteString("\n" + role.Prompt + "\n")
	}

	appendSkills(&out, role)

	appendRules(&out, persona.Rules)

	if slices.Contains(role.Tools, "run_command") {
		appendComputer(&out, installed)
		appendGitHub(&out, installed, github)
	}

	fmt.Fprintf(&out, childWorkflow, childTalk(role))

	return out.String()
}

const (
	cannotAsk = "You cannot talk to the person. When something is unclear, take the safest reasonable assumption " +
		"and say so in your report."
	canAsk = "You may ask the person with ask_user, a form of one to four questions in one call: each is single " +
		"(pick one), multi (pick several) or text, with two to four options that have a label and a description " +
		"(more, with a preview picture each, when you were given picture names), " +
		"the one you recommend first and marked (Recommended); they can also type their own answer. Ask only " +
		"what blocks the work and never repeat a question. For anything else take the safest reasonable " +
		"assumption and say so in your report."
)

// childTalk is what a teammate is told about talking to the person.
func childTalk(role AgentRole) string {
	if slices.Contains(role.Tools, toolAskUser) {
		return canAsk
	}

	return cannotAsk
}
