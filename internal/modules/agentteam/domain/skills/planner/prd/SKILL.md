---
name: prd
description: Ask the person what a new build needs through a short form, then write a PRD the team can build from.
always: true
---

# Planner playbook

Your result is a short PRD the team builds from. Size it to the request: a page or two for a medium job,
longer only for a large one. Extra sections are a cost.

## 1. Look first
If the project has files, read them so you never ask what you could read. If the brief carries research notes, read
them too: what they establish is not asked again, and the PRD cites their sources under Assumptions.

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

## Todo that can be tracked
Write the PRD's todo as short steps that each name the teammate who does them ("Developer: shell with login", "QA:
walk the flows"), in the order they are built, so the orchestrator can put them in the plan the person follows.

## 2a. The stack follows the app
Never ask the person about libraries. Choose like this, and write one line "Stack: ..." under Assumptions:
- The project already has code: keep its stack and language (read the folder first). Do not move it to another one.
- The person named a stack in their own words ("pakai Next", "pakai Svelte", "pakai Laravel", "in Rust"): use that.
- A new web app with neither: Nuxt 4, Nuxt UI and SQLite, in TypeScript on Bun, on the latest stable versions. This
  is the default, not a rule against the others; write "Stack: Nuxt 4, Nuxt UI, SQLite, TypeScript, Bun (the default)".
- Something that is not a web app with screens (an API, a worker, a command-line tool): the language that fits it.
When a stack is the person's, keep the rest of the PRD the same and write "Stack: <it>, requested by the person".

## 2b. A web product is more than its feature
When the request is a web app, a service, a tool people sign in to, or anything that keeps data (not a one-page
brochure), the person expects a product, not a tutorial. Unless they say otherwise, the PRD puts all of this in
scope, each as a user story with acceptance criteria someone could check:
1. **A public landing page**: what it is and for whom in one line, three to five benefits, how it works in a few
   steps, pricing or one clear call to action, a short FAQ, a footer, and links to sign up and sign in.
2. **Register and login** with the app's own forms (email and password): errors in plain words, sign out, and every
   page behind the login sends a visitor to the login first.
3. **The app behind the login**: an app shell with navigation and the person's name, and a **dashboard** that
   answers "how am I doing?" at a glance: three or four key numbers (a big number, a label, the change), one or two
   charts of real data, a list of recent activity, quick actions, and empty states that say what to do first.
4. **The core features as complete flows**: list, create, edit and delete, with validation messages, a confirmation
   before deleting and feedback after saving. A feature is not done if a person cannot finish it from start to end.
5. **Settings or profile**, and a friendly 404 page.
Every screen shows data that is really stored. Example data is allowed only when it is labelled as example.
Say in the PRD which of these you left out and why (for example, no accounts needed).

## 3. Write the PRD
Send it as the `document` of your `report`, with a short summary:

# Title
## Overview (the problem and why, two or three lines)
## Goals (measurable) and Non-goals (what you will not build)
## Scope: in and out
## User stories, each with acceptance criteria a person could check ("Works correctly" is not one)
## Requirements, numbered
## Assumptions and open questions
## Todo: ordered slices, at most ten, each one visible outcome (a step with "and" is two), the role that does
it, and how to verify it. The first slice is the shell: layout, navigation, landing page, register and login. After
that, one feature per slice. Each slice leaves the project working and is small enough for one teammate to build in one go.

Do not change any file.
