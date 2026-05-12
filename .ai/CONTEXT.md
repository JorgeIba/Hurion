# Project Context: Huron Problem Checker Bot

## Overview
The Huron Problem Checker Bot is a Discord bot designed to manage algorithm courses. It handles student enrollment, problem assignment tracking, and solution verification.

## Tech Stack
- **Language**: Go 1.24+
- **Framework**: `discordgo` (Discord API)
- **Database**: SQLite

## Domain Language & Constraints
- **Users**: Identified only by `userID` (internal/Discord).
- **userID vs discordID**: Always use `userID` in the data and service layers. Discord-specific logic is isolated in Controllers.
- **AC (Accepted)**: Refers to a successfully solved problem.
- **Classes**: Groups of users and problems identified by a `classCode`.
- **Permissions**: Four permission groups matching the controller structure:
  - `Admin`: Full system control.
  - `Mentor`: Class and student management.
  - `Enrolled`: Students registered in at least one class.
  - `General`: Non-enrolled users with basic access.

## Agentic Guidelines: How to Contribute
- **State Management**: Before making changes, use `ls -R` or `tree` to understand the current file structure if you are unsure.
- **Verification**: After every code change, run `make test` or at least `go fmt ./...`.
- **Mocks**: If you change interfaces, run `make generate` to update mocks.
- **Documentation**: If you add new commands, update `docs/COMMANDS.md`. If you change architecture, update `docs/ARCHITECTURE.md`.
- **Surgical Changes**: Only modify what is strictly necessary. Avoid refactoring unrelated code.
- **Layer Integrity**: Ensure logic stays in the correct layer (e.g., business logic in Services, formatting in Presentation).
- **Confirmation Flow**: When implementing "dangerous" commands, always follow the two-step confirmation pattern described in `docs/ARCHITECTURE.md`.
- **Async Responses**: Use the `Dispatcher` for all Discord interactions to ensure responses are handled correctly and asynchronously.
- **Test-First**: Before fixing a bug, try to reproduce it with a unit or integration test.

