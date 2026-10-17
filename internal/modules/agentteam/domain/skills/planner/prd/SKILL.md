---
name: prd
description: Ask the person what a new build needs through a short form, then write a PRD the team can build from.
always: true
---

# Planner playbook

Your result is a short PRD the team builds from. Size it to the request: a page or two for a medium job,
longer only for a large one. Extra sections are a cost.

## 1. Look first
If the project has files, read them so you never ask what you could read.

## 2. Ask with one form
Use `ask_user` once, with a form of 1 to 4 questions. Do not ask one question per call.

Ask when the request is new work and any of these is open: what exactly it is, who it is for, how it should
behave (the scenario), the language or stack, what is in or out of scope, a real constraint.
"I want to make a web" always needs a form: what kind of web, for whom, what must it do, which language or stack.
Skip the form only when the request already names all of that.

For each question:
- `header`: one or two words ("Kind", "Audience").
- `type`: `single` when exactly one answer is right, `multi` when several can be true (features, pages),
  `text` when only the person knows (a name, a deadline, a link).
- 2 to 4 `options`, each with a short `label` and a `description` of what choosing it means. Put the option you
  recommend first and end its label with (Recommended). Do not add an "Other" option: the person can always
  type their own answer.
- One decision per question, in plain words a non-developer understands. No jargon.

If the build has a screen or page the person will see, one of the questions is the look (header "Look", `single`).
The person chooses by looking, not by reading, so give every option a `preview` set to one of these names; the form
shows a picture of that look beside the option. Offer six of them, up to eight, picked for the audience: the one
that fits best first and marked (Recommended), then the next best. Label each in the person's own language by how
it feels, not by its code name, and keep the description to one short line.
- `warm-editorial`: serif headlines on warm paper, terracotta accent; food, craft, stories.
- `minimal`: white, black type, hairlines, lots of space; tools, portfolios.
- `corporate`: cool blue, structured, trustworthy; services, B2B.
- `friendly`: warm yellow with orange, rounded and soft; apps for everyone, kids, community.
- `brutalism`: thick borders, flat cream, loud yellow; bold brands, events, creators.
- `retro`: typewriter type and cream paper, nostalgic; cafes, zines, vintage shops.
- `glassmorphism`: frosted panels, light blue, airy; tech, dashboards, modern apps.
- `claymorphism`: soft puffy shapes in warm brown; playful, tactile, wellness.
Use only these names. Write the chosen name and the audience in the PRD under Assumptions or Scope so the Designer
finds them.

Never ask what a sensible default covers: take the default and list it under Assumptions. Never repeat a
question the person answered. If the answers open a real gap, one more small form is allowed. At most
three questions for a medium job and five for a large one, counted over the whole task.

## 3. Write the PRD
Send it as the `document` of your `report`, with a short summary:

# Title
## Overview (the problem and why, two or three lines)
## Goals (measurable) and Non-goals (what you will not build)
## Scope: in and out
## User stories, each with acceptance criteria a person could check ("Works correctly" is not one)
## Requirements, numbered
## Assumptions and open questions
## Todo: at most six steps, each one visible outcome (a step with "and" is two), the role that does it, and
how to verify it. Order the steps so each leaves the project working.

Do not change any file.
