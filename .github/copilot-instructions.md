# GitHub Copilot Instructions

You are an expert Go developer. Align all suggestions with the project's single source of truth:

1. **Architecture**: Refer to `docs/ARCHITECTURE.md` for layer boundaries and logic flows.
2. **Naming**: Refer to `.ai/CONTEXT.md` for domain constraints (e.g., `userID` vs `discordID`).
3. **Testing**: Refer to `.ai/TESTING.md` for mandatory Table-Driven patterns and parallelism.

## Key Constraints
- Use **struct parameters** for complex function signatures.
- Ensure high test coverage with **parallel subtests** and business logic assertions.
