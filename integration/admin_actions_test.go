package integration

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_ClearClass_Logic(t *testing.T) {
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

	// 2. ACT: Call Controller directly for the final action
	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!clear_class", classCode})
	ctx.Message.ID = "msg_original_123"

	admin.HandleClearClass(ctx)

	// 3. ASSERT
	env.AssertClassNotExists(t, classCode)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Contains(t, lastMsg.Content, "has been removed successfully")
	}
}

func TestIntegration_ExterminateUser_Logic(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	env.SeedClass(t, "class_1", "Class One")
	env.SeedClass(t, "class_2", "Class Two")

	testUser := service_model.User{UserID: "999888777", DisplayName: "Dead Man Walking"}
	env.SeedUser(t, testUser.UserID, testUser.DisplayName)
	env.SeedEnrollment(t, testUser.UserID, "class_1")
	env.SeedEnrollment(t, testUser.UserID, "class_2")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return userID == adminUser.UserID, nil
	}

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!exterminate", "<@999888777>"})

	// 2. ACT
	admin.HandleExterminate(ctx)

	// 3. ASSERT: User should be gone from all tables
	env.AssertUserNotExists(t, testUser.UserID)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Contains(t, lastMsg.Content, "completely removed from the system")
	}
}
