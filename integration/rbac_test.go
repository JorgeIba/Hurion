package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/admin"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/enrolled"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

// TestIntegration_RBAC_Identity covers Identity-based security (IsMentor, IsAdmin checks)
func TestIntegration_RBAC_Identity(t *testing.T) {
	t.Parallel()

	t.Run("Mentor - Student cannot add problem", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		classCode := "basicos_1"
		env.SeedClass(t, classCode, "Basicos 2026")

		studentUser := service_model.User{UserID: "student_1", DisplayName: "Student One"}
		env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
			return false, nil
		}

		ctx := env.NewContext(studentUser.UserID, studentUser.DisplayName, "guild_1", []string{"!add_problem", classCode, "url", "Easy", "Arrays", "New Problem"})
		mentor.HandleAddProblem(ctx)

		sentMsgs := env.Dispatcher.GetSentMessages()
		assert.NotEmpty(t, sentMsgs)
		assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "is not mentor")

		env.AssertProblemNotExists(t, 1)
	})

	t.Run("Mentor - Mentor can add problem", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		classCode := "basicos_1"
		env.SeedClass(t, classCode, "Basicos 2026")

		mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
		env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
			return true, nil
		}

		ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!add_problem", classCode, "http://valid.url", "Easy", "Arrays", "Success Problem"})
		mentor.HandleAddProblem(ctx)

		sentMsgs := env.Dispatcher.GetSentMessages()
		assert.NotEmpty(t, sentMsgs)
		assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "Success Problem")

		class, _ := env.DB.GetClassByClassCode(classCode)
		problems, _ := env.DB.GetProblemsByClass(class.ID)
		assert.Len(t, problems, 1)
	})

	t.Run("Admin - Non-admin cannot create class", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		studentUser := service_model.User{UserID: "student_1", DisplayName: "Student One"}
		env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
			return false, nil
		}

		ctx := env.NewContext(studentUser.UserID, studentUser.DisplayName, "guild_1", []string{"!create_class", "new_class", "New Class"})
		admin.HandleCreateClass(ctx)

		sentMsgs := env.Dispatcher.GetSentMessages()
		assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "is not admin")
		env.AssertClassNotExists(t, "new_class")
	})

	t.Run("Admin - Admin can create class", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		adminUser := service_model.User{UserID: "admin_1", DisplayName: "Admin"}
		env.Validator.IsAdminFunc = func(guildID, userID string) (bool, error) {
			return true, nil
		}

		ctx := env.NewContext(adminUser.UserID, adminUser.DisplayName, "guild_1", []string{"!create_class", "power_class", "Power Class"})
		admin.HandleCreateClass(ctx)

		_, err := env.DB.GetClassByClassCode("power_class")
		assert.NoError(t, err)
	})
}

// TestIntegration_RBAC_Enrollment covers Enrollment-based security (Discord Roles checks)
func TestIntegration_RBAC_Enrollment(t *testing.T) {
	t.Parallel()

	t.Run("Student - !ac Denied due to missing role", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		classID := env.SeedClass(t, "secret", "Secret")
		problemID := env.SeedProblem(t, "P1", "url", classID, "Hard", "Topic")
		env.SeedClassAccessRole(t, "secret", "role_member")

		student := service_model.User{UserID: "student_1", DisplayName: "Student"}
		env.SeedUser(t, student.UserID, student.DisplayName)

		ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!ac", fmt.Sprintf("%d", problemID)})
		enrolled.HandleProposeAC(ctx)

		sentMsgs := env.Dispatcher.GetSentMessages()
		assert.Contains(t, sentMsgs[0].Content, "is not enrolled in the class")
	})

	t.Run("Student - !undoac Allowed even without role", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		classID := env.SeedClass(t, "restricted", "Restricted")
		problemID := env.SeedProblem(t, "P1", "url", classID, "Easy", "Logic")
		env.SeedClassAccessRole(t, "restricted", "role_student")
		
		student := service_model.User{UserID: "student_1", DisplayName: "Student"}
		env.SeedUser(t, student.UserID, student.DisplayName)
		env.SeedAC(t, student.UserID, problemID)
		
		ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!undo_ac", fmt.Sprintf("%d", problemID)})
		enrolled.HandleUndoAC(ctx)

		sentMsgs := env.Dispatcher.GetSentMessages()
		assert.Contains(t, sentMsgs[0].Content, "has been UnAC-ed successfully")
	})

	t.Run("Student - Cross-Class Denied (has roles for A, tries to AC B)", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		env.SeedClass(t, "class_a", "Class A")
		env.SeedClassAccessRole(t, "class_a", "role_a")
		
		classID_B := env.SeedClass(t, "class_b", "Class B")
		env.SeedClassAccessRole(t, "class_b", "role_b")
		problemID_B := env.SeedProblem(t, "Problem B", "url", classID_B, "Easy", "Topic")

		student := service_model.User{UserID: "student_1", DisplayName: "Student"}
		env.SeedUser(t, student.UserID, student.DisplayName)
		env.SetRoles(student.UserID, []string{"role_a"})

		ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!ac", fmt.Sprintf("%d", problemID_B)})
		enrolled.HandleProposeAC(ctx)

		sentMsgs := env.Dispatcher.GetSentMessages()
		assert.Contains(t, sentMsgs[0].Content, "is not enrolled in the class")
	})

	t.Run("List - Public access even without roles", func(t *testing.T) {
		env := setupIntegrationEnv(t)
		classID := env.SeedClass(t, "private", "Private")
		env.SeedProblem(t, "P1", "url", classID, "Easy", "Topic")
		env.SeedClassroom(t, "channel_123", "private")
		env.SeedClassAccessRole(t, "private", "role_vip")

		student := service_model.User{UserID: "student_1", DisplayName: "Student"}
		env.SeedUser(t, student.UserID, student.DisplayName)
		
		ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!list"})
		ctx.Message.ChannelID = "channel_123"

		general.HandleGetListOfProblems(ctx)

		sentMsgs := env.Dispatcher.GetSentMessages()
		assert.Equal(t, dispatcher.PaginationEmbed, sentMsgs[0].Type)
	})
}
