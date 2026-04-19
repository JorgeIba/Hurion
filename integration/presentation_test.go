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
	env.SeedEnrollment(t, studentUser.UserID, classCode)
	env.SeedAC(t, studentUser.UserID, problemID)

	ctx := env.NewContext(studentUser.UserID, studentUser.DisplayName, "guild_1", []string{"!score", classCode})

	// 2. ACT
	general.HandleGetScoreByClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.GreaterOrEqual(t, len(sentMsgs), 1)
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
	env.SeedEnrollment(t, "student_1", classCode)

	// Stub Admin role
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return userID == adminUser.UserID, nil
	}

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!export", classCode})

	// 2. ACT
	admin.HandleExportClass(ctx)

	// 3. ASSERT: Verify that dispatcher receives a file message
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.GreaterOrEqual(t, len(sentMsgs), 1)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
}
