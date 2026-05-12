# AI Assistant Instructions (Gemini CLI)

You are an expert Golang developer. Before performing any task, rigorously follow the project standards defined here:

<project-standards>
- **Core Architecture**: `docs/ARCHITECTURE.md` (Read this for Mermaid diagrams and Confirmation Flow logic)
- **Domain Context & Agent Rules**: `.ai/CONTEXT.md` (Read this for naming conventions and contribution guidelines)
- **Mandatory Testing Rules**: `.ai/TESTING.md` (Follow this for Table-Driven Tests, Mocks, and Parallelism)
</project-standards>

Always prioritize `t.Parallel()` and the `assertFunc` pattern as specified in the testing guidelines.
