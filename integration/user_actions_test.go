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
	env.SeedClass(t, "basicos_1", "Basicos 2026")

	// Define test user
	testUser := service_model.User{
		UserID:      "123456789",
		DisplayName: "Jorge el Pro",
	}

	ctx := env.NewContext(testUser.UserID, testUser.DisplayName, "guild_1", []string{"!register", "basicos_1"})

	// 2. ACT
	general.HandleRegisterUserInClass(ctx)

	// 3. ASSERT Dispatcher
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.Len(t, sentMsgs, 1)
	msg := sentMsgs[0]
	assert.Equal(t, dispatcher.SimpleMessage, msg.Type)
	assert.Contains(t, msg.Content, "has registered in the class `basicos_1`")

	// 4. ASSERT Database using DSL
	env.AssertUserExists(t, testUser.UserID, testUser.DisplayName)
	env.AssertEnrollmentExists(t, testUser.UserID, "basicos_1")
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
	class, err := env.DB.GetClassByClassCode(classCode)
	assert.NoError(t, err)
	isEnrolled, err := env.DB.IsUserRegisteredInClass(testUser.UserID, class.ID)
	assert.NoError(t, err)
	assert.False(t, isEnrolled, "User should no longer be enrolled")
}

func TestIntegration_UndoAC(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	testUser := service_model.User{UserID: "student_1", DisplayName: "Student One"}
	env.SeedUser(t, testUser.UserID, testUser.DisplayName)
	env.SeedEnrollment(t, testUser.UserID, "basicos_1")
	env.SeedAC(t, testUser.UserID, problemID)

	ctx := env.NewContext(testUser.UserID, testUser.DisplayName, "guild_1", []string{"!undo_ac", fmt.Sprintf("%d", problemID)})

	// 2. ACT
	enrolled.HandleUndoAC(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.GreaterOrEqual(t, len(sentMsgs), 1)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "UnAC-ed successfully")

	// 4. ASSERT Database: AC should be gone
	env.AssertProblemNotSolved(t, testUser.UserID, problemID)
}

func TestIntegration_ListProblems(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)

	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	env.SeedProblem(t, "Two Sum", "https://leetcode.com/problems/two-sum/", classID, "Easy", "Arrays")
	env.SeedProblem(t, "Add Two Numbers", "https://leetcode.com/problems/add-two-numbers/", classID, "Medium", "Linked Lists")

	ctx := env.NewContext("user_1", "User One", "guild_1", []string{"!list", "basicos_1"})

	// 2. ACT
	general.HandleGetListOfProblemsByClass(ctx)

	// 3. ASSERT Dispatcher
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.Len(t, sentMsgs, 1)
	msg := sentMsgs[0]
	assert.Equal(t, dispatcher.PaginationEmbed, msg.Type)
	assert.Len(t, msg.Embeds, 1)
	assert.Contains(t, msg.Embeds[0].Title, "basicos_1")
	
	var nameColumnValue string
	for _, f := range msg.Embeds[0].Fields {
		if f.Name == "Name" {
			nameColumnValue = f.Value
			break
		}
	}
	assert.NotEmpty(t, nameColumnValue, "Expected an embed field with name 'Name'")
	assert.Contains(t, nameColumnValue, "Two Sum")
	assert.Contains(t, nameColumnValue, "Add Two Numbers")
}
