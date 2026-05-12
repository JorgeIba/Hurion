# Epic: Contextual Channels & Role-Based Enrollment

## Motivation
Historically, the Huron bot relied on an internal `enrollments` database table to track which users were in which classes. This approach led to two main issues:
1. **State Drift:** If a user's Discord role changed, the database became out of sync.
2. **Poor UX:** Users were forced to type the `class_code` (e.g., `!ac 1 basicos_1`) despite already being in a channel dedicated exclusively to that class.

## The Vision
We are transitioning the bot to a **Discord-Native, Stateless Architecture**:
- **Implicit Context:** The bot will infer the target class based on the Discord `channel_id` where the command is executed.
- **RBAC via Discord:** Enrollments will no longer be stored in our database. We will rely entirely on Discord's highly-optimized Role-Based Access Control (checking the user's roles in memory at execution time).
- **Configuration as Code:** A `--seed` CLI flag will allow admins to populate the database on startup using a `config.yaml` file, separating infrastructure deployment from runtime state.

## Implementation Plan (4-Part Merge Request)

*This epic is broken down into 4 sequential MRs to ensure safe, reviewable increments without breaking the main branch.*

### [MR 1] Database Normalization (Current)
- **Goal:** Prepare the database foundation without altering existing business logic.
- **Changes:** Add `classrooms` and `class_access_roles` tables to map Discord infrastructure to our domain logic. Add corresponding data layer queries. **Stateless pivot:** Delete the old `enrollments` logic and table in favor of real-time role checks.

### [MR 2] Configuration & The `--seed` Flag
- **Goal:** Enable deterministic initialization of classes.
- **Changes:** Introduce `config.yaml` parsing. Implement the `--seed` startup flag to populate the database tables introduced in MR 1 based on the YAML configuration.

### [MR 3] Context Resolution & Security Pivot
- **Goal:** Switch the bot's internal engine to use the new architecture.
- **Changes:** Update the command router to resolve the `class_code` from the `channel_id`. Rewrite all service permission checks to validate against Discord roles instead of the `enrollments` table. Delete the old `enrollments` logic and table.

### [MR 4] UX Overhaul & Admin Tools
- **Goal:** Update the user-facing commands to take advantage of the new system.
- **Changes:** Remove `class_code` arguments from contextual commands (`!list`, `!score`, `!ac`). Add new admin commands for runtime channel and role linking. Remove obsolete `!register` and `!deregister` commands.
