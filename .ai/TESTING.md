# Testing Guidelines

When writing or updating unit tests for this project, you MUST strictly adhere to these practices.

## 1. Core Testing Architecture
- **Table-Driven Testing**: ALWAYS use anonymous structs for test cases.
- **Assertions**: Use `github.com/stretchr/testify/assert`.
- **Mocking**: Use `go.uber.org/mock/gomock` for interfaces.
- **Parallelism**: ALWAYS call `t.Parallel()` at the start of the top-level test AND inside the `t.Run` loop.
- **Go Version**: Using Go 1.24+, so loop variable rebinding (`tt := tt`) is NOT required.

## 2. Test Case Structure (The `assertFunc` Pattern)
Every table-driven test should define the following properties. The most important is `assertFunc` to guarantee we test business logic, not just error absence.

```go
tests := []struct {
	name        string
	// Input parameters
	input       string
	
	// Mocks and Expected Outputs
	setupMock   func(*mocks.MockInterfaces)
	wantErr     bool
	wantErrType error
	
	// CRITICAL: Used on successful executions to validate business logic
	assertFunc  func(t *testing.T, res ExpectedReturnType) 
}{...}
```

## 3. Mocking Best Practices (`setupMock`)
- **Naming**: Refer to the interface as `DataStore` (CamelCase) and the struct field as `datastore` (lowercase) to maintain clarity.
- **Isolation**: Do NOT use global mocks. Re-instantiate `ctrl := gomock.NewController(t)` and `mockStore := mocks.NewMockDataStore(ctrl)` inside the `for _, tt := range tests` loop.
- Only mock what is absolutely necessary for the specific path.
- If a mocked function is called iteratively inside a loop, use `.AnyTimes()` safely.

## 4. Execution & Assertion Loop Template
Always use this standardized execution loop:

```go
func TestYourMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		// ... fields ...
		setupMock   func(*mocks.MockDataStore)
		wantErr     bool
		wantErrType error
		assertFunc  func(t *testing.T, res ExpectedType)
	}{...}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &YourService{Datastore: mockStore}
			
			// ACT
			res, err := svc.YourMethod(tt.inputs)

			// ASSERT ERRORS
			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
				
				// ASSERT BUSINESS LOGIC
				if tt.assertFunc != nil {
					tt.assertFunc(t, res)
				}
			}
		})
	}
}
```

## 5. Error Layer Validations
Our application uses custom layered errors (`dataerr`, `serviceerr`, `controllerserr`).
- **Mocking Failures**: Return native errors via mocks: `.Return(nil, &dataerr.NotFoundError{})`.
- **Validation**: Assert that the service successfully transformed the `dataerr` into the corresponding `serviceerr` type using `assert.IsType(t, tt.wantErrType, err)`.

## 6. Pointer Helpers
To avoid "cannot use string constant as *string" errors, use helpers at the top of the test file:
```go
func ptrStr(s string) *string { return &s }
// Usage: Difficulty: ptrStr("Easy")
```

## 7. Data Layer (Integration) Testing
- For files in `app/data`, do NOT use mocks.
- Use a real SQLite in-memory database (`:memory:`) to verify actual SQL execution and constraints.
- Always initialize a fresh database schema for each test case to ensure isolation.

## 8. Mock Generation
- If you modify any interface, you MUST run `make generate` (or `go generate ./...`) to update the mocks before writing tests.
- **IMPORTANT**: Mocks are gitignored. A fresh clone will NOT compile tests until you run `make generate` or `make test`.
