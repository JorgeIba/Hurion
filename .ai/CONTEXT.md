# Project Context & AI Hub

This directory is the Source of Truth for all AI agents (Gemini CLI, GitHub Copilot, Cursor, etc.). All tool-specific entrypoints should point here.

## Documentation Map

Refer to these documents for specific project knowledge:

- **[Core Architecture](../docs/ARCHITECTURE.md)**: Layered structure, Mermaid diagrams, and Confirmation Flow logic.
- **[Commands Reference](../docs/COMMANDS.md)**: Full list of Discord commands and their parameter constraints.
- **[Testing Guidelines](./TESTING.md)**: Mandatory Table-Driven patterns, `assertFunc`, and parallelism.
- **[Project README](../README.md)**: Setup, environment variables, and Docker instructions.

---

## Technical Stack
- **Language**: Go 1.24+
- **Framework**: `discordgo` (Discord API integration).
- **Database**: SQLite (stateless architecture via `--seed`).
- **Communication**: Async via `Dispatcher` queue.

---

## Domain Language & Constraints
- **`userID`**: Internal/Discord user identifier (always use `userID` in data/service layers).
- **AC (Accepted)**: A successfully solved problem.
- **Stateless Enrollment**: Roles and channels are the source of truth, NOT a database table.
- **Permissions**:
  - Admin: Full system control + class linkage.
  - Mentor: Class management + AC approvals.
  - Enrolled: Students with access roles.
  - General: Basic non-enrolled access.

---

## Agentic Guidelines (How to Contribute)
1. **Surgical Changes**: Only modify what is strictly necessary. Avoid refactoring unrelated code.
2. **Layer Integrity**: Business logic MUST stay in Services; formatting MUST stay in Presentation/Controllers.
3. **Test-First**: Reproduce bugs with a unit or integration test before fixing.
4. **Validation**: Run `make test` or `go fmt ./...` after every change.
5. **Mocks**: If you modify interfaces, run `make generate` to update mocks.
6. **Async Response**: Always use the `Dispatcher` for Discord messages.
7. **Confirmation Pattern**: Follow the 👍/👎 pattern for dangerous commands as described in Architecture docs.

---

## Critical Constraints
- **No Discord Object Leaks**: Ensure complex Discord objects (Messages, Members, Roles) are handled only in the Controller/Provider layers. The data/service layers must only work with raw string IDs (like `userID`, `roleID`).
- **No implicit state**: Do not assume a user is enrolled unless verified via real-time role checks.
- **Agnostic Logic**: Keep documentation and logic tool-agnostic. Use entrypoints for tool-specific instructions.
