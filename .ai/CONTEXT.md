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
