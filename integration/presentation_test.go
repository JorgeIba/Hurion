package integration

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/controllers/admin"
	"github.com/JorgeIba/Hurion/app/controllers/general"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
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
	_, err := env.DB.AddProblem(data.Problem{
		Name: "Two Sum", URL: "url", ClassID: classID,
		Difficulty: new("easy"), Topic: new("Arrays"), Priority: new("recommended"),
	})
	require.NoError(t, err)
	env.SeedProblem(t, "No Priority", "url-2", classID, "easy", "Arrays")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.SeedUser(t, "student_1", "Student")

	// Stub Admin role
	env.SeedRoleToUser(adminUser.UserID, "admin_role")

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
		workbook, err := excelize.OpenReader(lastMsg.Files[0].Reader)
		require.NoError(t, err)
		t.Cleanup(func() { _ = workbook.Close() })
		for cell, want := range map[string]string{
			"E1": "Priority", "E3": "Recommended", "E4": "No priority assigned",
		} {
			got, err := workbook.GetCellValue("Problems", cell)
			assert.NoError(t, err)
			assert.Equal(t, want, got)
		}
		scoreLabel, err := workbook.GetCellValue("Scores", "A4")
		assert.NoError(t, err)
		assert.Contains(t, scoreLabel, "[Priority: Recommended]")
		merges, err := workbook.GetMergeCells("Problems")
		assert.NoError(t, err)
		if assert.Len(t, merges, 1) {
			assert.Equal(t, "E2", merges[0].GetEndAxis())
		}
	}
}
