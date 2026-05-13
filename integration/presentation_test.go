package integration

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_ScoreGeneration_SmokeTest(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	studentUser := service_model.User{UserID: "student_1", DisplayName: "Student One"}
	env.SeedUser(t, studentUser.UserID, studentUser.DisplayName)
	env.SeedAC(t, studentUser.UserID, problemID)

	ctx := env.NewContext(studentUser.UserID, studentUser.DisplayName, "guild_1", []string{"!score", classCode})

	// 2. ACT
	general.HandleGetScoreByClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		// The score command currently sends a SimpleMessage "Sending info" followed by the FileMessage
		// But in integration tests, we want to ensure the final result is correct
		assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
		assert.NotEmpty(t, lastMsg.Files)
	}
}

func TestIntegration_ExcelExport_SmokeTest(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedUser(t, "student_1", "Student")

	// Stub Admin role
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return userID == adminUser.UserID, nil
	}

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!export", classCode})

	// 2. ACT
	admin.HandleExportClass(ctx)

	// 3. ASSERT: Verify that dispatcher receives a file message
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
		assert.NotEmpty(t, lastMsg.Files)
		assert.Contains(t, lastMsg.Files[0].Name, classCode)
	}
}
