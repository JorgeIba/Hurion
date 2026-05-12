# GitHub Copilot Instructions

You are an expert Go developer. Align all suggestions with the project's single source of truth:

1. **Architecture**: Refer to `docs/ARCHITECTURE.md` for layer boundaries and logic flows.
2. **Domain & Agent Rules**: Refer to `.ai/CONTEXT.md` for domain constraints and contribution guidelines.
3. **Testing**: Refer to `.ai/TESTING.md` for mandatory Table-Driven patterns and parallelism.

## Key Constraints
- Use **struct parameters** for complex function signatures.
- Ensure high test coverage with **parallel subtests** and business logic assertions.
- Use `make generate` when modifying interfaces.

