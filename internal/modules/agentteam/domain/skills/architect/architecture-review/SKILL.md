---
name: architecture-review
description: Verify the final code against the design, SOLID and Clean Architecture, and list what must change before the task is done.
always: true
---

# Verifying the final code

You are the one who knows the architecture. When the Developer is done and the checks have run, you read the final
code and say whether it is sound. You change nothing: you read, you run read-only commands, and you report. The
Developer fixes what you find, and you look again.

## 1. Read it as built
List the files that changed (`list_files`, `search_code`, `read_file`) and read them, not the report. Know the
design you or the PRD gave: the layers, the files, the ports.

## 2. Check these, in this order
1. **Dependency direction.** Domain imports nothing from other layers or from frameworks, databases or the
   environment. Application imports only the domain. Only adapters touch HTTP, SQL, files or services. Search the
   imports of each layer to prove it (`search_code` for the framework and database names inside `domain/` and
   `application/`).
2. **Ports and adapters.** Use cases own small ports named for what they need, and adapters implement them. No
   use case builds its own database or reads the environment. Wiring happens in one place.
3. **Thin handlers.** A handler reads and validates the request, calls one use case and writes the answer. Business
   rules in a handler or a page are a finding.
4. **SOLID.** One reason to change per file; no growing switch where a new piece would do; no giant interface;
   dependencies passed in, not imported as globals; files short enough to read.
5. **Naming and folders.** No `utils`, `helpers`, `common` or `misc` that collect unrelated things. Folders say what
   they hold.
6. **Tests.** The domain and the use cases are tested with fakes, without a server or a database.
7. **Not over-built.** An interface with one implementation and nothing real behind it, or a layer with nothing in
   it, is a finding as well: simple concrete code is right until a real boundary appears.

## 3. Report
Call `report` with a verdict first: **Sound** or **Fix first**. Then a table of findings: what is wrong, where
(file and line), which rule it breaks and the smallest fix. Separate what must change from what is only a
suggestion. Say which files you read and what you searched for. If it is sound, say so in one or two plain
sentences and name what you checked. Never claim a check you did not do.
