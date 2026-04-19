package integration

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
)

// testEnv holds the environment for integration tests
type testEnv struct {
	Ctx        *context.ContextMessage
	DB         *data.DB
	Services   *services.Services
	Validator  *StubDiscordProvider
	Fetcher    *StubDiscordProvider
	Modifier   *StubDiscordProvider
	Dispatcher *FakeDispatcher
}

// setupIntegrationEnv initializes a fresh in-memory database and wires all services
func setupIntegrationEnv(t *testing.T) *testEnv {
	t.Helper()

	logger.Init()

	// Initialize in-memory SQLite
	db, err := data.NewDB(&data.DBConfig{
		DBPath: ":memory:",
		DBType: data.Test,
	})
	if err != nil {
		t.Fatalf("Failed to initialize in-memory DB: %v", err)
	}
	t.Cleanup(db.CloseDB)

	// Initialize services
	svc := services.NewServices(db)

	// Initialize Stub Providers (Quiet by default)
	stub := &StubDiscordProvider{}
	fakeDispatcher := &FakeDispatcher{}

	// Construct ContextMessage
	ctx := &context.ContextMessage{
		DiscordValidator:       stub,
		DiscordFetcher:         stub,
		DiscordMessageModifier: stub,
		Dispatcher:             fakeDispatcher,
		Services:               svc,
	}
	return &testEnv{
		Ctx:        ctx,
		DB:         db,
		Services:   svc,
		Validator:  stub,
		Fetcher:    stub,
		Modifier:   stub,
		Dispatcher: fakeDispatcher,
	}
}

// ============================================================================
// Testing DSL Helpers
// ============================================================================

// SeedClass inserts a class using the DataStore repository
func (env *testEnv) SeedClass(t *testing.T, classCode, name string) int64 {
	t.Helper()
	id, err := env.DB.CreateClass(data.Class{
		ClassCode: classCode,
		Name:      name,
	})
	if err != nil {
		t.Fatalf("Failed to seed class: %v", err)
	}
	return id
}

// SeedProblem inserts a problem related to a class ID using the DataStore repository
func (env *testEnv) SeedProblem(t *testing.T, name, url string, classID int64, difficulty, topic string) int64 {
	t.Helper()
	id, err := env.DB.AddProblem(data.Problem{
		Name:       name,
		URL:        url,
		ClassID:    classID,
		Difficulty: &difficulty,
		Topic:      &topic,
	})
	if err != nil {
		t.Fatalf("Failed to seed problem: %v", err)
	}
	return id
}

// NewContext creates a pre-wired ContextMessage for testing a command
func (env *testEnv) NewContext(userID, displayName, guildID string, args []string) *context.ContextMessage {
	return &context.ContextMessage{
		Message: &discordgo.MessageCreate{
			Message: &discordgo.Message{
				ChannelID: "channel_1",
				Author:    &discordgo.User{ID: userID, Username: displayName},
			},
		},
		Args:                   args,
		Executor:               service_model.User{UserID: userID, DisplayName: displayName},
		ExecutorGuildID:        guildID,
		DiscordValidator:       env.Validator,
		DiscordFetcher:         env.Fetcher,
		DiscordMessageModifier: env.Modifier,
		Dispatcher:             env.Dispatcher,
		Services:               env.Services,
	}
}

// AssertUserExists verifies that the given user exists and matches the expected display name
func (env *testEnv) AssertUserExists(t *testing.T, userID, expectedName string) {
	t.Helper()
	dbUser, err := env.DB.GetUserByUserID(userID)
	if err != nil {
		t.Fatalf("Failed to query user %q: %v", userID, err)
	}
	if dbUser.DisplayName != expectedName {
		t.Errorf("Expected user display name %q, got %q", expectedName, dbUser.DisplayName)
	}
}

// AssertEnrollmentExists verifies that the user is enrolled in the given class code
func (env *testEnv) AssertEnrollmentExists(t *testing.T, userID, classCode string) {
	t.Helper()
	class, err := env.DB.GetClassByClassCode(classCode)
	if err != nil {
		t.Fatalf("Failed to find class %q for enrollment check: %v", classCode, err)
	}
	isEnrolled, err := env.DB.IsUserRegisteredInClass(userID, class.ID)
	if err != nil {
		t.Fatalf("Failed to query enrollment: %v", err)
	}
	if !isEnrolled {
		t.Errorf("Expected user %q to be enrolled in class %q", userID, classCode)
	}
}

// SeedUser ensures a user exists in the database
func (env *testEnv) SeedUser(t *testing.T, userID, displayName string) {
	t.Helper()
	err := env.DB.EnsureUser(data.User{
		UserID:      userID,
		DisplayName: displayName,
	})
	if err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}
}

// SeedEnrollment registers a user into a class
func (env *testEnv) SeedEnrollment(t *testing.T, userID, classCode string) {
	t.Helper()
	class, err := env.DB.GetClassByClassCode(classCode)
	if err != nil {
		t.Fatalf("Failed to find class %q for seeding enrollment: %v", classCode, err)
	}
	err = env.DB.RegisterUserIntoClass(userID, class.ID)
	if err != nil {
		t.Fatalf("Failed to seed enrollment: %v", err)
	}
}

// SeedAC adds a solved problem for a user
func (env *testEnv) SeedAC(t *testing.T, userID string, problemID int64) {
	t.Helper()
	err := env.DB.AddAC(userID, problemID)
	if err != nil {
		t.Fatalf("Failed to seed AC: %v", err)
	}
}

// SeedPendingConfirmation inserts a pending confirmation into the database
func (env *testEnv) SeedPendingConfirmation(t *testing.T, messageID, userID string, problemID int64) {
	t.Helper()
	err := env.DB.AddPendingConfirmation(data.PendingConfirmation{
		MessageID: messageID,
		UserID:    userID,
		ProblemID: problemID,
	})
	if err != nil {
		t.Fatalf("Failed to seed pending confirmation: %v", err)
	}
}

// AssertProblemSolved verifies that a user has solved a specific problem
func (env *testEnv) AssertProblemSolved(t *testing.T, userID string, problemID int64) {
	t.Helper()
	isSolved, err := env.DB.HasUserSolvedProblem(userID, problemID)
	if err != nil {
		t.Fatalf("Failed to query AC status: %v", err)
	}
	if !isSolved {
		t.Errorf("Expected problem %d to be solved by user %q", problemID, userID)
	}
}

// AssertProblemNotSolved verifies that a user has NOT solved a specific problem
func (env *testEnv) AssertProblemNotSolved(t *testing.T, userID string, problemID int64) {
	t.Helper()
	isSolved, err := env.DB.HasUserSolvedProblem(userID, problemID)
	if err != nil {
		t.Fatalf("Failed to query AC status: %v", err)
	}
	if isSolved {
		t.Errorf("Expected problem %d NOT to be solved by user %q", problemID, userID)
	}
}

// AssertClassNotExists verifies that the given class code does not exist in the database
func (env *testEnv) AssertClassNotExists(t *testing.T, classCode string) {
	t.Helper()
	_, err := env.DB.GetClassByClassCode(classCode)
	if err == nil {
		t.Errorf("Expected class %q to be deleted, but it still exists", classCode)
		return
	}

	var notFoundErr *dataerr.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Unexpected error checking class existence: %v", err)
	}
}
