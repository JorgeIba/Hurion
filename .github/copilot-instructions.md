# Copilot Instructions for Huron Problem Checker Bot

## Project Overview
This is a Discord bot built in Go using the discordgo library for managing algorithms courses. The bot handles user registration, class management, problem assignments, and solution tracking with role-based permissions (admin, mentor, student). 

Data is stored in SQLite with a clean layered architecture.

## Architecture & Design Principles

### Layer Structure
- **Controllers**: Handle Discord interactions, organized by role/permission (admin, enrolled, general, mentor)
- **Services**: Business logic layer, currently consolidated in `user_service.go`
- **Data Layer**: Database operations, framework-agnostic with generic terminology (use `userID` not `discordID`)

### Code Style Guidelines
- Use **struct parameters** instead of individual parameters for scalable function signatures
- Example: `AddProblem(problem Problem)` instead of `AddProblem(name, url string, classID int64)`
- Keep data layer framework-agnostic (avoid Discord-specific terminology)
- Follow clean architecture principles with clear separation of concerns

## Naming Conventions

### Database/Data Layer
- Use generic terms: `userID` not `discordID`
- Parameters should be descriptive and framework-agnostic
- Function names should be clear about their operations

### Parameter Conventions (Help System)
- **Lowercase with underscore**: Should be one word (e.g., `class_code` → "basicos_1", `user_name` → "jorge_ibanez")
- **CamelUpperCase**: Can include multiple words (e.g., `ProblemName` → "Gosha Is Hunting 2", `ClassName` → "Basicos 2025")

## File Organization

### Key Files
- `app/data/queries.go`: Database operations
- `app/data/queries_test.go`: Test for each DB operation
- `app/data/datastore.go`: Data interface definitions
- `app/services/user_service.go`: Business logic
- `app/controllers/*/`: Role-based command handlers
- `app/bot/commandrouter.go`: Command routing
- `app/bot/helpCommand.go`: Help system with command metadata
- `app/data/schema.sql`: Database schema

### Test Files
- Follow pattern: `function_test.go` with descriptive test names
- Any time a function needs to be added on `queries.go`, add corresponding tests in `queries_test.go`, not every use case should be necessarily tested, but at least the main use cases should be covered.
- Use `testConfig` for in-memory SQLite testing
- Structure: Preconditions → Act → Verify, try to add the corresponding comments for each step separation.
- Use existing functions for test setup (prefer `CreateClass()`, `AddProblem()` over raw SQL)

## Common Patterns

### Error Handling
These are the common error types used across the application, organized by layer, if you need to add a new error type (none of those can be used), please follow the same pattern and add it to the corresponding layer.
```go
## Data layer
* Initialization Error
* Query Error
* Transaction Error
* Row Conflict Error
* Not Found Error
## Service Layer
* Internal Error
* Conflict Error
* Not Found Error

## Validation Layer
* Internal Error
* Not Found Error
## Controllers Layer
* Validation Error
* Internal Error
* Parsing Error
```

Errors can be seen as links betweehn layers, so if you need to add a new error type, please make sure to add it in the corresponding layer and follow the same pattern, e.g error from data layer should not
be used in the controller layer.

### Database Schema Notes
* Use foreign key relationships appropriately
* Follow SQLite best practices for schema design

### Testing Guidelines
* Write comprehensive unit tests for new data layer functions
* Test both success and error cases
* Use descriptive test names: TestFunctionName_Scenario
* Include edge cases and boundary conditions
* Verify error types match expected patterns

### When Adding New Features
New Commands:
* Add to commandInfoMap in helpCommand.go
* Add route in commandrouter.go
* Implement handler in appropriate controller
* Add business logic to service if needed
* Add data operations to queries.go and interface if needed

New Database Operations:
* Add to datastore.go interface
* Implement in queries.go using struct parameters
* Handle errors appropriately with custom error types
* Write unit tests covering success/failure cases
* Update service layer to use new operations

### Consistency Reminders
* Always use struct parameters for new functions with multiple parameters.
* Maintain framework-agnostic data layer
* Follow established error handling patterns
* Write tests that use existing helper functions rather than raw SQL
* Consider scalability when designing function signatures

### Discord Bot Specific
* Use c.Dispatcher.Send() for sending messages from controllers
* Controller is the only layer we can send messaged (there's one exception, the helpCommand.go file, which is used to send the help message).
* Role-based command access through appropriate controller files
* Message formatting uses Discord markdown syntax


## Code instructions
When a use case is requested, it should be provided with the following info:
* Who (role) is requesting the use case
* Description, what the use case is about (e.g., "Add a new problem", "Register a user")
* Any specific requirements or constraints could be read (e.g., "Use struct parameters", "Follow error handling patterns")
* Use case format (e.g. !register <user_name> <class_code>).
* If use case is already implemented, provide the file and function name where it can be found.

### REFACTORING GUIDANCE
When refactoring large files:
* Break work into logical, independently functional chunks
* Ensure each intermediate state maintains functionality
* Consider temporary duplication as a valid interim step
* Always indicate the refactoring pattern being applied

### General Code Writing & Refactoring Instructions

* When writing or refactoring code, act as a senior developer with strong experience in clean code practices, design patterns, and scalable architecture.
* Prioritize **readability, maintainability, and performance**. Code should be easy to understand and modify by others.
* Use consistent **styling and formatting** that aligns with the project’s conventions (e.g., indentation, naming, spacing).
* Follow **SOLID principles**, where applicable, and favor clear abstractions over premature optimization.
* Remove **duplicated logic** or overly complex structures. Simplify where possible without sacrificing clarity.
* Leverage well-known, **idiomatic patterns** in the language or framework being used.
* Avoid unnecessary dependencies or clever hacks unless justified and well-documented.
* Write **descriptive names** for functions, variables, and classes—avoid abbreviations unless they are domain-standard.
* If comments are necessary, write clear and concise explanations—but let clean code be self-documenting when possible.
* Don't change code if it's not related to the requested use case, even if it seems like an improvement. Focus on the specific request and ensure it works as intended.

## Layers responsabilities
- **Controllers**: Orchestrate service layer calls, handle Discord interactions, and manage user permissions.
  - Each controller should be responsible for a specific role or permission level (admin, enrolled, general, mentor).
  - Controllers should not contain business logic; they should delegate to services.
  - Use structured parameters for function signatures to maintain clarity and scalability.
- **Services**: Implement business logic, manage data transformations, and coordinate between controllers and data layer.
- **Data Layer**: Perform database operations, manage data persistence, and provide a clean interface for services.
- **Validation Layer**: Ensure data integrity and correctness before processing in services or controllers.
