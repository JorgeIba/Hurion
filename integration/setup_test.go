package integration

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers/mock_providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"go.uber.org/mock/gomock"
)

// testEnv holds the environment for integration tests
type testEnv struct {
	Ctx          *context.ContextMessage
	DB           *data.DB
	Services     *services.Services
	Validator    *mock_providers.MockDiscordValidator
	Fetcher      *mock_providers.MockDiscordFetcher
	Modifier     *mock_providers.MockDiscordMessageModifier
	Dispatcher   *FakeDispatcher
	Ctrl         *gomock.Controller
}

// FakeDispatcher captures messages instead of sending them to Discord
type FakeDispatcher struct {
	SentMessages []dispatcher.MessageResponse
}

func (f *FakeDispatcher) Send(msg dispatcher.MessageResponse) {
	f.SentMessages = append(f.SentMessages, msg)
}

// setupIntegrationEnv initializes a fresh in-memory database and wires all services
func setupIntegrationEnv(t *testing.T) *testEnv {
	t.Helper()

	logger.Init()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

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

	// Initialize mocks
	validator := mock_providers.NewMockDiscordValidator(ctrl)
	fetcher := mock_providers.NewMockDiscordFetcher(ctrl)
	modifier := mock_providers.NewMockDiscordMessageModifier(ctrl)
	fakeDispatcher := &FakeDispatcher{}

	// Construct ContextMessage
	ctx := &context.ContextMessage{
		DiscordValidator:       validator,
		DiscordFetcher:         fetcher,
		DiscordMessageModifier: modifier,
		Dispatcher:             fakeDispatcher,
		Services:               svc,
	}

	return &testEnv{
		Ctx:          ctx,
		DB:           db,
		Services:     svc,
		Validator:    validator,
		Fetcher:      fetcher,
		Modifier:     modifier,
		Dispatcher:   fakeDispatcher,
		Ctrl:         ctrl,
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
func (env *testEnv) SeedProblem(t *testing.T, name, url string, classID int64, difficulty, topic string) {
	t.Helper()
	_, err := env.DB.AddProblem(data.Problem{
		Name:       name,
		URL:        url,
		ClassID:    classID,
		Difficulty: &difficulty,
		Topic:      &topic,
	})
	if err != nil {
		t.Fatalf("Failed to seed problem: %v", err)
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
