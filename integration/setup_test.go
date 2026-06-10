package integration

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "failed to initialize in-memory DB")
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
	require.NoError(t, err, "failed to seed class %q", classCode)
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
	require.NoError(t, err, "failed to seed problem %q", name)
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
	require.NoError(t, err, "failed to query user %q", userID)
	assert.Equal(t, expectedName, dbUser.DisplayName)
}

// AssertUserNotExists verifies that the given user ID does not exist in the database
func (env *testEnv) AssertUserNotExists(t *testing.T, userID string) {
	t.Helper()
	_, err := env.DB.GetUserByUserID(userID)
	if !assert.Error(t, err, "expected user %q to not exist, but it does", userID) {
		return
	}

	var notFoundErr *dataerr.NotFoundError
	require.ErrorAs(t, err, &notFoundErr, "unexpected DB error while checking user existence")
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

// SeedClassroom links a Discord room to a class code.
func (env *testEnv) SeedClassroom(t *testing.T, roomID, classCode string) {
	t.Helper()
	err := env.DB.AddClassroom(data.Classroom{RoomID: roomID, ClassCode: classCode})
	if err != nil {
		t.Fatalf("Failed to seed classroom mapping: %v", err)
	}
}

// SeedClassAccessRole links a Discord role to class access.
func (env *testEnv) SeedClassAccessRole(t *testing.T, classCode, roleID string) {
	t.Helper()
	err := env.DB.AddClassAccessRole(data.ClassAccessRole{ClassCode: classCode, RoleID: roleID})
	if err != nil {
		t.Fatalf("Failed to seed class access role: %v", err)
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
	require.NoError(t, err, "failed to query AC status")
	assert.True(t, isSolved, "expected problem %d to be solved by user %q", problemID, userID)
}

// AssertProblemNotSolved verifies that a user has NOT solved a specific problem
func (env *testEnv) AssertProblemNotSolved(t *testing.T, userID string, problemID int64) {
	t.Helper()
	isSolved, err := env.DB.HasUserSolvedProblem(userID, problemID)
	require.NoError(t, err, "failed to query AC status")
	assert.False(t, isSolved, "expected problem %d NOT to be solved by user %q", problemID, userID)
}

// AssertClassNotExists verifies that the given class code does not exist in the database
func (env *testEnv) AssertClassNotExists(t *testing.T, classCode string) {
	t.Helper()
	_, err := env.DB.GetClassByClassCode(classCode)
	if !assert.Error(t, err, "expected class %q to not exist, but it does", classCode) {
		return
	}

	var notFoundErr *dataerr.NotFoundError
	require.ErrorAs(t, err, &notFoundErr, "unexpected DB error while checking class existence")
}

// AssertProblemNotExists verifies that the given problem ID does not exist in the database
func (env *testEnv) AssertProblemNotExists(t *testing.T, problemID int64) {
	t.Helper()
	_, err := env.DB.GetProblemByID(problemID)
	if !assert.Error(t, err, "expected problem %d to not exist, but it does", problemID) {
		return
	}

	var notFoundErr *dataerr.NotFoundError
	require.ErrorAs(t, err, &notFoundErr, "unexpected DB error while checking problem existence")
}

// SetRoles configures the stub fetcher to return the given roles for a specific user.
func (env *testEnv) SetRoles(userID string, roles []string) {
	if env.Fetcher.UserRoles == nil {
		env.Fetcher.UserRoles = make(map[string][]string)
	}
	env.Fetcher.UserRoles[userID] = roles
}

// SeedRoleToUser adds a specific role to a user in the stub fetcher.
func (env *testEnv) SeedRoleToUser(userID string, roleID string) {
	if env.Fetcher.UserRoles == nil {
		env.Fetcher.UserRoles = make(map[string][]string)
	}
	roles := env.Fetcher.UserRoles[userID]
	for _, r := range roles {
		if r == roleID {
			return // Already has the role
		}
	}
	env.Fetcher.UserRoles[userID] = append(roles, roleID)
}

// UnseedRoleToUser removes a specific role from a user in the stub fetcher.
func (env *testEnv) UnseedRoleToUser(userID string, roleID string) {
	if env.Fetcher.UserRoles == nil {
		return
	}
	roles := env.Fetcher.UserRoles[userID]
	newRoles := []string{}
	for _, r := range roles {
		if r != roleID {
			newRoles = append(newRoles, r)
		}
	}
	env.Fetcher.UserRoles[userID] = newRoles
}
