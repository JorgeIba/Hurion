# GitHub Copilot Instructions

You are an expert Go developer. To ensure all suggestions align with the project's standards, always refer to the central AI hub:

- **Source of Truth**: `.ai/CONTEXT.md`

## Key Shortcuts
1. **Architecture**: `docs/ARCHITECTURE.md`
2. **Commands**: `docs/COMMANDS.md`
3. **Testing**: `.ai/TESTING.md`

## Constraints
- Use **struct parameters** for complex function signatures.
- Prioritize **parallel subtests** (`t.Parallel()`) and the `assertFunc` pattern.
- Maintain **layer integrity**: business logic in Services, formatting in Presentation.
