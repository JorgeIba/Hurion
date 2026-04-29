package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/enrolled"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_RegisterUser(t *testing.T) {
	t.Parallel()

	// 1. SETUP Environment
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	otherClass := "other_class"
	env.SeedClass(t, classCode, "Basicos 2026")
	env.SeedClass(t, otherClass, "Other")

	// Define test user
	testUser := service_model.User{
		UserID:      "123456789",
		DisplayName: "Jorge el Pro",
	}

	ctx := env.NewContext(testUser.UserID, testUser.DisplayName, "guild_1", []string{"!register", classCode})

	// 2. ACT
	general.HandleRegisterUserInClass(ctx)

	// 3. ASSERT Dispatcher
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.Len(t, sentMsgs, 1)
	msg := sentMsgs[0]
	assert.Equal(t, dispatcher.SimpleMessage, msg.Type)
	assert.Contains(t, msg.Content, "has registered in the class `basicos_1`")

	// 4. ASSERT Database Side-effects
	env.AssertUserExists(t, testUser.UserID, testUser.DisplayName)
	env.AssertEnrollmentExists(t, testUser.UserID, classCode)
	env.AssertEnrollmentNotExists(t, testUser.UserID, otherClass)
}

func TestIntegration_DeregisterUser(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	testUser := service_model.User{UserID: "user_to_leave", DisplayName: "I am leaving"}
	env.SeedUser(t, testUser.UserID, testUser.DisplayName)
	env.SeedEnrollment(t, testUser.UserID, classCode)

	ctx := env.NewContext(testUser.UserID, testUser.DisplayName, "guild_1", []string{"!deregister", classCode})

	// 2. ACT
	enrolled.HandleDeregisterUser(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.GreaterOrEqual(t, len(sentMsgs), 1)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "deregistered from the class")

	// 4. ASSERT Database: Enrollment should be gone
	env.AssertEnrollmentNotExists(t, testUser.UserID, classCode)
}

func TestIntegration_UndoAC(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	testUser := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, testUser.UserID, testUser.DisplayName)
	env.SeedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.SetRoles(testUser.UserID, []string{"role_enrolled"})
	env.SeedAC(t, testUser.UserID, problemID)

	ctx := env.NewContext(testUser.UserID, testUser.DisplayName, "guild_1", []string{"!undo_ac", fmt.Sprintf("%d", problemID)})

	// 2. ACT
	enrolled.HandleUndoAC(ctx)

	// 3. ASSERT
	env.AssertProblemNotSolved(t, testUser.UserID, problemID)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "UnAC-ed successfully")
}

func TestIntegration_UndoAC_DoubleUndo(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	testUser := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, testUser.UserID, testUser.DisplayName)
	env.SeedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.SetRoles(testUser.UserID, []string{"role_enrolled"})
	// Note: We do NOT seed an AC here

	ctx := env.NewContext(testUser.UserID, testUser.DisplayName, "guild_1", []string{"!undo_ac", fmt.Sprintf("%d", problemID)})

	// 2. ACT
	enrolled.HandleUndoAC(ctx)

	// 3. ASSERT: Should receive a user-friendly error from MapError
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "you haven't solved problem")
}

func TestIntegration_ListProblems(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)

	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	env.SeedProblem(t, "P1", "https://p1.url", classID, "Easy", "Arrays")
	env.SeedProblem(t, "P2", "https://p2.url", classID, "Medium", "Arrays")

	ctx := env.NewContext("user_1", "User One", "guild_1", []string{"!list", "basicos_1"})

	// 2. ACT
	general.HandleGetListOfProblems(ctx)

	// 3. ASSERT Dispatcher
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.Len(t, sentMsgs, 1)
	msg := sentMsgs[0]
	assert.Equal(t, dispatcher.PaginationEmbed, msg.Type)
	assert.Len(t, msg.Embeds, 1)

	var nameColumnValue string
	for _, f := range msg.Embeds[0].Fields {
		if f.Name == "Name" {
			nameColumnValue = f.Value
			break
		}
	}
	assert.Contains(t, nameColumnValue, "P1")
	assert.Contains(t, nameColumnValue, "P2")
}

func TestIntegration_ListProblems_ContextualAccessAllowed(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)

	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	env.SeedProblem(t, "P1", "https://p1.url", classID, "Easy", "Arrays")
	env.SeedClassroom(t, "channel_1", "basicos_1")

	ctx := env.NewContext("user_1", "User One", "guild_1", []string{"!list"})

	general.HandleGetListOfProblems(ctx)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.Len(t, sentMsgs, 1)
	assert.Equal(t, dispatcher.PaginationEmbed, sentMsgs[0].Type)
	assert.NotEmpty(t, sentMsgs[0].Embeds)
}

func TestIntegration_ListProblems_ContextualIgnoresRoles(t *testing.T) {
	t.Parallel()

	env := setupIntegrationEnv(t)

	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	env.SeedProblem(t, "P1", "https://p1.url", classID, "Easy", "Arrays")
	env.SeedClassroom(t, "channel_1", "basicos_1")
	env.SeedClassAccessRole(t, "basicos_1", "role_mentor")

	ctx := env.NewContext("user_1", "User One", "guild_1", []string{"!list"})

	general.HandleGetListOfProblems(ctx)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.Len(t, sentMsgs, 1)
	assert.Equal(t, dispatcher.PaginationEmbed, sentMsgs[0].Type)
	assert.NotEmpty(t, sentMsgs[0].Embeds)
}
