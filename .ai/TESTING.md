# Testing Guidelines

This document defines the testing strategy for the repo by **confidence boundary**, not by setup mechanics. A test is classified by the behavior it validates, not by whether it seeds data, uses a real DB, or touches multiple files.

The repo currently uses three suites:
- **Unit tests** validate isolated logic and local behavior.
- **Integration tests** validate real contracts between adjacent application layers.
- **Workflow tests** validate multi-step user journeys and actor-driven state transitions.

---

## Part 1: Unit Testing
*Located alongside the code (for example `app/services/user_service_test.go`).*

### What unit tests prove
- One component behaves correctly in isolation.
- Failures should point to a narrow rule, transformation, or branch in that component.
- The test should explain *why* the logic is wrong without needing to inspect other layers.

### Typical scope in this repo
- Service methods with mocked repositories.
- Controller helpers and adapter behavior.
- Provider logic with mocked Discord session interfaces.
- Sorting, parsing, validation, and presentation formatting helpers.

### Unit test standards
- **Table-Driven Testing**: ALWAYS use anonymous structs for test cases.
- **Assertions**: Use `github.com/stretchr/testify/assert`.
- **Mocking**: Use `go.uber.org/mock/gomock` for interfaces.
- **Parallelism**: ALWAYS call `t.Parallel()` at the start of the test AND inside the `t.Run` loop (but if the test file is not already following these guidelines, please ignore this rule for that file).
- **Error Layer Validation**: Assert that lower-layer errors (for example `dataerr`) are correctly transformed into higher-layer errors (for example `serviceerr`) using `assert.IsType`.

### Standard unit test loop
Every unit test should follow this template so the test validates business logic, not just the absence of errors.

```go
func TestMethod(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockDataStore)
		wantErr     bool
		wantErrType error
		assertFunc  func(t *testing.T, res ResultType)
	}{...}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)
			// ... ACT and ASSERT ...
		})
	}
}
```

---

## Part 2: Integration Testing
*Located in the root `integration/` directory.*

### What integration tests prove
- Real adjacent layers collaborate correctly inside the application.
- The contract between layers holds when the real implementations are wired together.
- A failure should tell us that two or more real layers disagree about behavior, data shape, or domain rules.

In this repo, that usually means checks across boundaries like:
- Controller -> Service -> Repository -> SQLite
- Controller -> Presentation -> Dispatcher output
- Permission and contextual resolution flows that rely on real services plus fake external I/O

### What does **not** define an integration test
- It is **not** defined by “tests that seed data.”
- Direct seeding is only a setup technique.
- Using `:memory:` SQLite is an implementation detail, not the philosophy.

### Typical scope in this repo
- Controller-level cross-layer checks in `integration/`.
- Presentation smoke tests that validate real rendering/output contracts.
- Permission and context-aware behavior using real services and DB plus fake Discord collaborators.

### Integration test standards
- **Real Database**: Use an isolated in-memory SQLite database (`:memory:`).
- **Real Services**: Use the actual service implementations, not mocks.
- **Faked External I/O**: Use `FakeDispatcher`/test doubles to capture outgoing Discord behavior and stub role/member lookups.
- **Readable DSL**: Prefer `testEnv` helper methods over raw SQL to keep tests aligned with domain concepts instead of schema details.

### Setup guidance
- Seed **static prerequisites** directly when helpful, such as classes, problems, room mappings, and access-role mappings.
- Do not worry about whether a test seeds data; worry about whether it is validating a **layer contract**.
- Existing tests in `integration/` remain valid and should stay there unless a future change explicitly decides otherwise.

### Example fit
- Good integration tests for this repo:
  - controller-level score/list/export behavior using real DB/services
  - admin/general/mentor contract checks through controller entrypoints
  - presentation smoke tests for generated files or embeds

---

## Part 3: Workflow Testing
*Located in the root `workflow/` directory.*

### What workflow tests prove
- A real user or business journey behaves correctly across multiple steps.
- Actor transitions are correct: one user starts a flow, another user continues or approves it, and the system updates state and outputs accordingly.
- A failure should point to a broken journey, not just a broken function.

### Typical scope in this repo
- Bot-entry flows that begin from message or reaction events.
- Multi-actor transitions involving enrolled users, mentors, and admins.
- Confirmation flows and pending-state lifecycles.

### Workflow test standards
- Prefer entering through bot-level handlers or equivalent event-driven entrypoints.
- Assert **intermediate transitions** as well as final state.
- Prefer creating **dynamic workflow state** through earlier actions in the test instead of pre-seeding that exact state.
- Static prerequisites like classes, problems, room mappings, and role mappings may still be seeded directly.

### Example fit
- Good workflow tests for this repo:
  - student proposes `!ac` -> mentor reacts -> pending state becomes accepted/rejected
  - author triggers `!clear_class` -> confirmation is emitted -> author confirms/cancels -> state mutates or stays intact
  - non-author or unauthorized actor attempts to continue a workflow and the bot ignores it

---

## Part 4: General Best Practices
- **Pointer Helpers**: Use `ptrStr(s string) *string { return &s }` to handle optional fields in test structs.
- **Mock Generation**: If you modify an interface, run `make generate` before writing tests.
- **CI Readiness**: Tests must be parallel-safe and should not depend on local environment variables or files (except for `schema.sql`).
- **Workflow Additions**: Add new workflow coverage only when it provides confidence that existing unit and integration tests do not already provide.
