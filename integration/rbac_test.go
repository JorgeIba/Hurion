package integration

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_RBAC_StudentCannotAddProblem(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	studentUser := service_model.User{UserID: "student_1", DisplayName: "Student One"}

	// Stub that student is NOT a mentor
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return false, nil
	}

	ctx := env.NewContext(studentUser.UserID, studentUser.DisplayName, "guild_1", []string{"!add_problem", classCode, "url", "Easy", "Arrays", "New Problem"})

	// 2. ACT: Student tries to add a problem
	mentor.HandleAddProblem(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Equal(t, dispatcher.SimpleMessage, lastMsg.Type)
		assert.Contains(t, lastMsg.Content, "Bro has not enough permissions (is not mentor)")
	}

	// Verify database: no problem should be added
	class, err := env.DB.GetClassByClassCode(classCode)
	assert.NoError(t, err)
	problems, err := env.DB.GetProblemsByClass(class.ID)
	assert.NoError(t, err)
	assert.Empty(t, problems)
}

func TestIntegration_RBAC_MentorCanAddProblem(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}

	// Stub that user IS a mentor
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return true, nil
	}

	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!add_problem", classCode, "http://valid.url", "Easy", "Arrays", "Success Problem"})

	// 2. ACT
	mentor.HandleAddProblem(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Contains(t, lastMsg.Content, "Success Problem")
	}

	// Verify database
	class, err := env.DB.GetClassByClassCode(classCode)
	assert.NoError(t, err)
	problems, err := env.DB.GetProblemsByClass(class.ID)
	assert.NoError(t, err)
	assert.Len(t, problems, 1)
	assert.Equal(t, "Success Problem", problems[0].Name)
}

func TestIntegration_RBAC_NonAdminCannotCreateClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	studentUser := service_model.User{UserID: "student_1", DisplayName: "Student One"}

	// Stub that student is NOT an admin
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return false, nil
	}

	ctx := env.NewContext(studentUser.UserID, studentUser.DisplayName, "guild_1", []string{"!create_class", "new_class", "New Class Name"})

	// 2. ACT
	admin.HandleCreateClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Equal(t, dispatcher.SimpleMessage, lastMsg.Type)
		assert.Contains(t, lastMsg.Content, "is not admin")
	}

	// Verify database: no class should be created
	env.AssertClassNotExists(t, "new_class")
}

func TestIntegration_RBAC_AdminCanCreateClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin User"}

	// Stub that user IS an admin
	env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
		return true, nil
	}

	ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!create_class", "power_class", "The Power Class"})

	// 2. ACT
	admin.HandleCreateClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Contains(t, lastMsg.Content, "created correctly")
	}

	// Verify database
	_, err := env.DB.GetClassByClassCode("power_class")
	assert.NoError(t, err)
}
