package application

import (
	"fmt"
	"strings"
)

const defaultName = "Awam"

// chatSystemPrompt instructs the model for conversational mode.
func chatSystemPrompt(project string, persona Persona) string {
	var out strings.Builder

	name := persona.Name
	if name == "" {
		name = defaultName
	}

	fmt.Fprintf(&out, "You are %s, an AI assistant conversing with a person who is not a developer. ", name)
	fmt.Fprintf(&out, "You are in the project %q.\n\n", project)
	out.WriteString("Role: You are in Chat Mode (discussion, brainstorming, planning, and Q&A). ")
	out.WriteString("Answer directly, warmly, and concisely in plain words without jargon. ")
	out.WriteString("Do not call update_plan or attempt tool execution in this mode. ")
	out.WriteString("If the user wants concrete code changes or test runs, explain the approach briefly ")
	out.WriteString("and suggest running it as a Task.\n")

	if persona.TalkStyle != "" {
		fmt.Fprintf(&out, "Style: %s. ", persona.TalkStyle)
	}

	if persona.Language != "" {
		fmt.Fprintf(&out, "Always write to the person in %s. ", persona.Language)
	}

	out.WriteString("\n")

	if persona.About != "" {
		fmt.Fprintf(&out, "\nAbout the person: %s\n", persona.About)
	}

	if len(persona.Rules) > 0 {
		out.WriteString("\nRules to follow:\n")
		for _, rule := range persona.Rules {
			fmt.Fprintf(&out, "- %s\n", rule)
		}
	}

	return out.String()
}
