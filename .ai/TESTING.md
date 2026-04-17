# Testing Guidelines

This document defines the mandatory practices for maintaining code quality. We distinguish between **Unit Tests** (isolated logic) and **Integration Tests** (full workflows).

---

## Part 1: Unit Testing (Isolated Logic)
*Located alongside the code (e.g., `app/services/user_service_test.go`).*

### 1. Core Unit Test Architecture
- **Table-Driven Testing**: ALWAYS use anonymous structs for test cases.
- **Assertions**: Use `github.com/stretchr/testify/assert`.
- **Mocking**: Use `go.uber.org/mock/gomock` for interfaces.
- **Parallelism**: ALWAYS call `t.Parallel()` at the start of the test AND inside the `t.Run` loop (but if the test file is not already following these guidelines, please ignore this rule for that file).
- **Error Layer Validation**: Assert that lower-layer errors (e.g., `dataerr`) are correctly transformed into higher-layer errors (e.g., `serviceerr`) using `assert.IsType`.

### 2. Standard Unit Test Loop (The `assertFunc` Pattern)
Every unit test must follow this template to ensure business logic is validated, not just the absence of errors.

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

## Part 2: Integration Testing (Full Workflows)
*Located in the root `integration/` directory.*

### 1. The "Subcutaneous" Approach
Integration tests verify the contract between layers (Controller → Service → Repository → DB) without the overhead of real Discord network calls.
- **Real Database**: ALWAYS use an in-memory SQLite database (`:memory:`).
- **Real Services**: Use the actual service implementations, not mocks.
- **Faked External I/O**: Use `FakeDispatcher` to capture outgoing messages and `MockDiscordProvider` for role checks.

### 2. The Testing DSL (Domain Specific Language)
Do NOT use raw SQL in integration tests. Use the helper methods on `testEnv` to keep tests readable and decoupled from the schema.
- **Seeding**: Use `env.SeedClass(t, ...)` and `env.SeedProblem(t, ...)` to set up the state.
- **Assertions**: Use `env.AssertUserExists(t, ...)` and `env.AssertEnrollmentExists(t, ...)` to verify database state.

### 3. Execution Standard
Every integration test must initialize its own isolated environment to remain parallel-safe:
```go
func TestIntegration_Workflow(t *testing.T) {
    t.Parallel()
    env := setupIntegrationEnv(t) // 1. Isolated Setup
    
    // 2. ACT (Call a Controller or Service method)
    // 3. ASSERT (Check env.Dispatcher and env.AssertX methods)
}
```

---

## Part 3: General Best Practices
- **Pointer Helpers**: Use `ptrStr(s string) *string { return &s }` to handle optional fields in test structs.
- **Mock Generation**: If you modify an interface, run `make generate` before writing tests.
- **CI Readiness**: Tests must be parallel-safe and should not depend on local environment variables or files (except for `schema.sql`).
