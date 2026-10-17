---
name: clean-code
description: The rules every line of code follows: SOLID and Clean Architecture, in whatever stack the project uses.
always: true
---

# Code rules: SOLID and Clean Architecture

The Architect will check your final code against these rules, and the task is not finished until it passes. Follow
the project's own structure when it already has one; in a new project, build it like this from the first file.

## Clean Architecture: dependencies point inward
Three layers, and a file belongs to exactly one:
1. **Domain**: the rules of the app: entities, value objects, errors. Plain code. It imports nothing from a web
   framework, a database, the environment or another layer.
2. **Application**: the use cases (one action per file, such as "place an order") and the **ports** they need:
   small interfaces that the use case itself owns, named for what it needs ("OrderStore"), not for a technology.
   It imports the domain and nothing else.
3. **Adapters**: everything that touches the outside: HTTP handlers, the database, files, other services. They
   implement the ports and call the use cases. This is the only place that knows the framework or the SQL.
Wiring (creating the real database and passing it in) happens in one place at the edge, never inside a use case.
A handler only reads and checks the request, calls one use case and writes the answer. No business rule in it.

In a Nuxt app this is: `server/domain/`, `server/application/` (use cases and ports), `server/infrastructure/`
(SQLite stores, the auth setup), `server/api/` (thin handlers: validate with zod, call a use case), one file
`server/utils/container.ts` that builds the real parts, and `app/` for pages and components. A page calls the
API; it does not hold business rules. In another stack, use the same three layers in its own folders.

## SOLID, in practice
- **Single responsibility**: one reason to change per file, function and component. A function that checks,
  saves and sends a mail is three functions. Keep files short enough to read in one go.
- **Open/closed**: add behaviour by adding a new piece (a new use case, a new adapter), not by growing a switch
  in the old one.
- **Liskov**: an implementation of a port does what the port promises, including its errors. A fake in a test
  behaves like the real one.
- **Interface segregation**: small interfaces made for the one who uses them. Not one "Repository" with twenty
  methods.
- **Dependency inversion**: use cases depend on ports; adapters depend on the ports they implement. Never the
  other way round. Pass dependencies in (constructor or argument), do not import a global.

## Do not over-build
Simple concrete code first. Add a port only where something real sits behind it: a database, a mail service, the
clock. A tiny app is still layered, but with few files. No folders named utils, helpers or common that collect
unrelated things; name a folder for what it holds.

## Tests and report
Test the domain and the use cases with fakes for the ports, with no server and no database. Say in your report
which layers you made and which ports they use, so the Architect can check them quickly.
