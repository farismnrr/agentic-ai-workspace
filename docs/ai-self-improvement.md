# AI Self-Improvement

## Purpose

Self-improvement here means making the AI-assisted engineering process more reliable over time. It is not permission for an agent to change its own authority, bypass user intent, weaken safety constraints, or silently rewrite repository guardrails.

The improvement target is the quality of decisions, code, tests, documentation, and reusable patterns.

## Improvement loop

Every meaningful change should follow this loop:

1. **Observe** — inspect the current repository, behavior, logs, tests, and relevant documentation before editing.
2. **Diagnose** — state the concrete gap or failure and separate evidence from assumptions.
3. **Plan** — choose the smallest coherent change that preserves architecture and reuse rules.
4. **Implement** — make the change in the correct module/component boundary.
5. **Verify** — use the canonical build/test path and check the actual behavior affected by the change.
6. **Reflect** — identify whether the issue exposed a repeatable engineering lesson.
7. **Codify** — only when useful, encode the lesson in a reusable component, shared module, test, documentation, or guardrail.

## Evidence hierarchy

Prefer evidence in this order:

1. current repository code and configuration;
2. reproducible tests/build output/runtime logs;
3. official framework or dependency documentation;
4. primary upstream source/release notes;
5. assumptions only when unavoidable, clearly labeled.

Do not guess when the answer can be verified cheaply.

## Reuse learning

An improvement is considered reusable when it reduces future duplication or prevents a known class of failure. Examples:

- extracting a repeated Svelte UI pattern into a focused component;
- centralizing repeated API request behavior into a focused client module;
- moving repeated backend behavior into a domain service shared by real callers;
- adding a regression test after a bug;
- documenting a non-obvious architectural invariant.

Do not create generic abstractions merely to appear reusable. Reuse must be driven by concrete repetition or a stable domain concept.

## Reflection record

For substantial future work, the agent should be able to report:

- what evidence it inspected;
- what changed and why;
- how it verified the result;
- what reusable lesson, if any, was added;
- any unresolved risk or assumption.

A separate permanent document is not required for every task. Permanent documentation should be added only when the lesson will matter again.

## Guardrail protection

The AI must not autonomously:

- remove or weaken `AGENTS.md` constraints;
- introduce a second runtime server/port against the system design;
- add hot reload or dev-server flows against the build contract;
- replace daisyUI/reuse-first UI conventions without explicit direction;
- make dependency, security, deployment, or architecture changes simply because they are fashionable;
- claim verification that was not actually performed.

If a requested feature conflicts with an existing guardrail, surface the conflict and follow explicit user intent rather than silently choosing a new architecture.

## Success criteria

The self-improvement loop is working when later changes become easier to reason about, reuse more existing code, introduce fewer regressions, and require less ad-hoc recovery while keeping the repository simple enough to understand.
