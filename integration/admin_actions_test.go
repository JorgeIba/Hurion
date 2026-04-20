package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_ClearClass_Logic(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

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

	// 3. ASSERT: Class and its problems must be gone
	env.AssertClassNotExists(t, classCode)
	env.AssertProblemNotExists(t, problemID)

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
	class1ID := env.SeedClass(t, "class_1", "Class One")
	class2ID := env.SeedClass(t, "class_2", "Class Two")
	p1 := env.SeedProblem(t, "P1", "url", class1ID, "Easy", "Arrays")
	p2 := env.SeedProblem(t, "P2", "url", class2ID, "Easy", "Arrays")

	testUserID := "999888777"
	env.SeedUser(t, testUserID, "Dead Man Walking")
	env.SeedEnrollment(t, testUserID, "class_1")
	env.SeedEnrollment(t, testUserID, "class_2")
	env.SeedAC(t, testUserID, p1)
	env.SeedAC(t, testUserID, p2)

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return userID == adminUser.UserID, nil
	}

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!exterminate", fmt.Sprintf("<@%s>", testUserID)})

	// 2. ACT
	admin.HandleExterminate(ctx)

	// 3. ASSERT: Deep wipe verification
	env.AssertUserNotExists(t, testUserID)
	env.AssertEnrollmentNotExists(t, testUserID, "class_1")
	env.AssertEnrollmentNotExists(t, testUserID, "class_2")
	env.AssertProblemNotSolved(t, testUserID, p1)
	env.AssertProblemNotSolved(t, testUserID, p2)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "completely removed from the system")
}

func TestIntegration_CreateClass_AlreadyExists(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return userID == adminUser.UserID, nil
	}

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!create_class", classCode, "Duplicate Class"})

	// 2. ACT
	admin.HandleCreateClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	// The error mapper should handle the "already exists" error
	assert.Contains(t, lastMsg.Content, "already exists")
}

func TestIntegration_ExportClass_Logic(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "export_class"
	classID := env.SeedClass(t, classCode, "Export Class")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "student_1", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, classCode)
	env.SeedAC(t, student.UserID, problemID)

	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return userID == adminUser.UserID, nil
	}

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!export", classCode})

	// 2. ACT
	err := admin.HandleExportClass(ctx)
	assert.NoError(t, err)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
	assert.NotEmpty(t, lastMsg.Files)
	assert.Contains(t, lastMsg.Files[0].Name, classCode)
}

func TestIntegration_Admin_UnauthorizedAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler func(*context.ContextMessage)
		args    []string
	}{
		{
			name: "CreateClass unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleCreateClass(c)
			},
			args: []string{"!create_class", "new_class", "New Class"},
		},
		{
			name: "ClearClass unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleClearClass(c)
			},
			args: []string{"!clear_class", "some_class"},
		},
		{
			name: "Exterminate unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleExterminate(c)
			},
			args: []string{"!exterminate", "<@123>"},
		},
		{
			name: "ExportClass unauthorized",
			handler: func(c *context.ContextMessage) {
				admin.HandleExportClass(c)
			},
			args: []string{"!export", "some_class"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			student := service_model.User{UserID: "111", DisplayName: "Student"}

			// Seed something to verify it is NOT deleted/modified
			classCode := "protected_class"
			env.SeedClass(t, classCode, "Protected")

			// Mock NOT admin
			env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
				return false, nil
			}

			ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", tt.args)

			// ACT
			tt.handler(ctx)

			// ASSERT
			sentMsgs := env.Dispatcher.GetSentMessages()
			assert.NotEmpty(t, sentMsgs)
			assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "is not admin")

			// Verify DB side-effects: class should still exist
			_, err := env.DB.GetClassByClassCode(classCode)
			assert.NoError(t, err, "DB state should not change after unauthorized access")
		})
	}
}
