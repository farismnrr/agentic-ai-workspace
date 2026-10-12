# System Design

## Goal

Keep deployment simple while preserving strong boundaries: one executable, one process, one public port, and business logic
that does not depend on transport or infrastructure frameworks.

## Runtime architecture

```text
Browser
  |
  | HTTP :PORT
  v
Go binary
  |
  +-- Fiber API routes
  |     |
  |     +-- feature adapters -> application -> domain
  |
  +-- embedded Svelte SPA
```

The Svelte build is embedded with `go:embed`. Node.js is not part of runtime.

## Build pipeline

```text
Svelte + TypeScript source
  -> strict type check + ESLint
  -> Vite production build
  -> internal/platform/webassets/dist
  -> Go tests + strict golangci-lint
  -> go:embed
  -> Go build
  -> bin/agentic-ai-workspace
```

`make build` is the canonical path and runs the complete verification gate first.

## Backend structure

```text
cmd/workspace/
  main.go                         process entrypoint only

internal/bootstrap/
  app.go                          composition root

internal/modules/<feature>/
  domain/                         entities, value objects, domain rules
  application/                    use cases and consumer-owned ports
  adapter/                        HTTP/persistence/external implementations

internal/platform/
  config/                         environment/runtime configuration
  httpserver/                     Fiber composition and SPA delivery
  webassets/                      embedded compiled frontend
```

Do not create empty architecture layers merely to satisfy the diagram. A feature creates the layers it actually needs while
preserving the dependency direction.

## Dependency rule

The central rule is that source dependencies point inward toward higher-level policy.

```text
adapter --------> application --------> domain
   ^                    ^
   |                    |
framework/detail        ports owned by consumer

bootstrap -> may wire concrete dependencies across outer layers
platform  -> technical details only
```

### Domain

Domain code is framework independent. It must not know Fiber, HTTP DTOs, environment variables, database drivers, platform
packages, or bootstrap code. Prefer standard-library value types and domain-owned types.

### Application

Application code coordinates use cases. It depends on domain policy and defines the small ports it needs. It must not know
which database, HTTP framework, external SDK, or concrete adapter satisfies those ports.

### Adapter

Adapters translate outer-world concerns to application/domain concepts. Fiber handlers, persistence implementations, and
external-service clients belong here when they are feature-specific.

### Platform

Platform packages are process-wide technical details such as runtime configuration, the Fiber host, and embedded assets.
They must not become a home for business rules.

### Bootstrap

Bootstrap is the composition root. It chooses concrete implementations and wires them together. `cmd/` calls bootstrap and
contains no business behavior.

## SOLID interpretation for Go

### Single Responsibility

A package, file, function, component, or service owns one cohesive responsibility and one main reason to change. File-size and
complexity gates reinforce this rule but do not replace design judgment.

### Open/Closed

Prefer adding a new adapter or strategy behind an existing application-owned port over repeatedly editing a central switch or
god service. Do not introduce an interface until there is a real boundary or substitution need.

### Liskov Substitution

Concrete implementations must preserve the behavioral contract expected by the consumer. Tests should target the contract
when multiple implementations exist.

### Interface Segregation

Interfaces are small and consumer-specific. A use case should not receive a broad repository or service interface when it only
needs one or two operations.

### Dependency Inversion

Inner policy owns the abstraction; outer details implement it. Interfaces belong near the code that consumes them, not beside
the concrete implementation merely for symmetry.

## Frontend structure

```text
web/src/
  main.ts                         browser bootstrap
  app/                            app-level composition
  features/<feature>/
    domain/                       pure TypeScript feature policy when needed
    application/                  feature orchestration/state when needed
    ui/                           Svelte feature components
  shared/
    ui/                           reusable Svelte components
    config/                       reusable typed configuration/constants
```

Frontend source is TypeScript. JavaScript source under `web/src` is rejected. Any Svelte script block must use
`<script lang="ts">`.

UI reuse order is existing shared component, daisyUI composition, Tailwind utilities, then custom CSS only when necessary.

## File and complexity budgets

Every human-authored source/config/script/markup/documentation file is capped at 200 lines. Generated build output and lockfiles
are exempt. Go functions are capped at 60 lines and 40 statements; frontend functions are capped at 60 lines. Complexity is
also bounded by lint.

When a file approaches its limit, split it by responsibility or architectural boundary. Splitting purely to evade the metric is
a guardrail violation.

## Enforcement

`make verify` runs:

1. repository guardrail checks, including file size and TypeScript-only frontend rules;
2. `svelte-check` with warnings treated as failures;
3. strict type-aware ESLint;
4. production Vite build;
5. Go tests;
6. strict golangci-lint, including formatting, security, complexity, and `depguard` architecture checks.

A change is not complete until the canonical verification and build paths are green.
