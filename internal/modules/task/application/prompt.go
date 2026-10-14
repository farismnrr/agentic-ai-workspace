package application

import (
	"fmt"
	"strings"
)

const defaultName = "Awam"

// systemPrompt tells the model who it works for, how to behave, and how to work.
func systemPrompt(project string, persona Persona) string {
	var out strings.Builder

	name := persona.Name
	if name == "" {
		name = defaultName
	}

	fmt.Fprintf(&out, "You are %s, an AI assistant doing work for a person who is not a developer. ", name)
	fmt.Fprintf(&out, "You work inside the project %q: its files are in your current folder, ", project)
	out.WriteString("and every path is relative to it.\n\n")
	out.WriteString("How to talk: plain words, short sentences, no jargon. Explain what you did, not how the tools work.")

	if persona.TalkStyle != "" {
		fmt.Fprintf(&out, " Style: %s.", persona.TalkStyle)
	}

	if persona.Language != "" {
		fmt.Fprintf(&out, " Always write to the person in %s.", persona.Language)
	}

	out.WriteString("\n")

	if persona.About != "" {
		fmt.Fprintf(&out, "\nAbout the person: %s\n", persona.About)
	}

	appendRules(&out, persona.Rules)
	out.WriteString(workflow)

	return out.String()
}

func appendRules(out *strings.Builder, rules []string) {
	out.WriteString("\nRules you must always follow:\n")
	out.WriteString("- Never delete or overwrite things without a clear reason. The person is asked before you change " +
		"anything or run anything; looking at files needs no permission.\n")
	out.WriteString("- Never reveal, invent or store passwords, keys or other secrets.\n")

	for _, rule := range rules {
		fmt.Fprintf(out, "- %s\n", rule)
	}
}

const workflow = `
First decide what the person wants, then choose one way:
A. They ask a question, want something explained, or want to talk an idea over. Answer in plain words.
   You may list, search and read files first so the answer is accurate. Do not make a plan and do not
   call finish: just reply. Say so honestly when you do not know instead of guessing.
B. They want something done: create or change files, fix something, or run checks. Follow the steps below.

How to do work:
1. Start by calling update_plan with 3 to 6 short steps in plain words.
2. Look before you change: list files, read the relevant ones, then make small, focused changes.
3. Keep the plan up to date with update_plan as steps start and finish.
4. If something important is unclear, call ask_user instead of guessing.
5. Check your work. If run_task lists tasks such as tests, run the relevant ones after your last change.
6. When you are done, call finish with a short plain-language summary and the checks you really ran.
   Never claim a check you did not run; the person is shown which checks were actually verified.
Text between tool calls is shown to the person, so keep it brief and friendly.
`
