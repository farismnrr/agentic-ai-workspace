# System Design

## Goal

Build an AI workspace that stays simple to deploy: one executable, one process, one public port. The browser receives a Svelte SPA, while Go Fiber owns HTTP routing, APIs, and static delivery.

## High-level architecture

```text
Browser
   |
   | HTTP :PORT
   v
+-------------------------------+
| Go executable                 |
|                               |
|  Fiber                        |
|  +-------------------------+  |
|  | /api/*                  |  |
|  | backend modules         |  |
|  +-------------------------+  |
|  | /*                      |  |
|  | embedded Svelte SPA     |  |
|  +-------------------------+  |
+-------------------------------+
             ^
             |
      go:embed at build
             |
+-------------------------------+
| Svelte + Vite + Tailwind      |
| + daisyUI                     |
| build-time only               |
+-------------------------------+
```

## Build pipeline

```text
Svelte source
   -> Vite production build
   -> internal/web/dist
   -> go:embed
   -> Go build
   -> bin/agentic-ai-workspace
```

The order is intentional. Go must compile only after the latest frontend artifacts exist, otherwise the binary can embed stale UI.

## Runtime model

The final executable owns all runtime responsibilities needed for the web application:

- listens on one configurable port;
- serves backend API routes under `/api`;
- serves fingerprinted SPA assets from the embedded filesystem;
- falls back to `index.html` for browser-side SPA routes.

There is no Node.js process in production and no separate frontend port.

## Backend boundaries

- `cmd/workspace`: process entrypoint only.
- `internal/config`: runtime configuration and validation.
- `internal/server`: Fiber composition, middleware order, top-level route wiring, SPA fallback.
- future domain modules: `internal/<domain>` with their own handlers/services/repositories as complexity appears.
- shared backend code: extract only after multiple real consumers exist and give the package a concrete responsibility.

Avoid a monolithic `server.go` as features grow. The top-level server package should compose modules rather than own domain logic.

## Frontend boundaries

- `web/src/App.svelte`: top-level composition only.
- `web/src/lib/components`: reusable visual components.
- `web/src/lib`: reusable state, API clients, constants, stores, and focused shared logic.
- daisyUI provides the default semantic component layer.
- Tailwind utilities handle layout and targeted adjustments.
- custom CSS is a last resort for behavior or visuals that cannot be expressed cleanly through the existing design system.

## UI reuse rule

Before creating UI markup, check in this order:

1. existing local reusable Svelte component;
2. daisyUI component/composition;
3. Tailwind utility composition;
4. custom CSS or a new abstraction.

A component should be extracted when it represents a stable reusable concept, not merely because two fragments look vaguely similar.

## API and SPA routing

Backend/API routes are registered first. Static SPA delivery is registered afterward, and the final GET fallback returns embedded `index.html`. This prevents the SPA fallback from swallowing API endpoints.

## Configuration

Runtime configuration is environment-driven and read through `internal/config`. The initial runtime setting is:

- `PORT`, default `8080`.

As configuration grows, validation belongs in the config package rather than at individual call sites.

## Deployment contract

A deployment needs only the produced executable plus its environment variables. The canonical flow is:

```bash
make build
./bin/agentic-ai-workspace
```

No frontend source tree or Node.js installation is required on the runtime host once the binary has been built.

## Future growth

The architecture should grow by adding domain modules behind `/api`, reusable Svelte components and logic under `web/src/lib`, and explicit infrastructure packages only when needed. Preserve the single-binary boundary unless requirements prove it is no longer appropriate.
