package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_Mentor_AddAC(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedClassAccessRole(t, "basicos_1", "role_enrolled")
	env.SetRoles(student.UserID, []string{"role_enrolled"})

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!m_add_ac", fmt.Sprintf("<@%s>", student.UserID), fmt.Sprintf("%d", problemID)})

	// 2. ACT
	mentor.HandleAddAC(ctx)

	// 3. ASSERT
	env.AssertProblemSolved(t, student.UserID, problemID)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Equal(t, dispatcher.Reaction, sentMsgs[len(sentMsgs)-1].Type)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Reactions, common.REACTION_ACCEPTED)
}

func TestIntegration_Mentor_UndoAC(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedAC(t, student.UserID, problemID)

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!m_undo_ac", fmt.Sprintf("<@%s>", student.UserID), fmt.Sprintf("%d", problemID)})

	// 2. ACT
	mentor.HandleUndoAC(ctx)

	// 3. ASSERT
	env.AssertProblemNotSolved(t, student.UserID, problemID)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "has been UnAC-ed")
}

func TestIntegration_Mentor_RemoveProblem(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!remove_problem", fmt.Sprintf("%d", problemID)})

	// 2. ACT
	err := mentor.HandleRemoveProblem(ctx)
	assert.NoError(t, err)

	// 3. ASSERT
	env.AssertProblemNotExists(t, problemID)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "has been removed")
}

func TestIntegration_Mentor_UpdateProblem(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "class_1", "Class One")
	class2ID := env.SeedClass(t, "class_2", "Class Two")
	problemID := env.SeedProblem(t, "Old Name", "old_url", classID, "Easy", "Arrays")
	originalProblem, err := env.DB.GetProblemByID(problemID)
	if !assert.NoError(t, err) {
		return
	}

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	// !update_problem <id> <class_code> <url> <difficulty> <topic> <priority> <name...>
	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!update_problem", fmt.Sprintf("%d", problemID), "class_2", "new_url", "Hard", "Graphs", "recommended", "New Name"})

	// 2. ACT
	mentor.HandleUpdateProblem(ctx)

	// 3. ASSERT: Specific fields updated, and deterministic values checked
	updatedProblem, err := env.DB.GetProblemByID(problemID)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updatedProblem.Name)
	assert.Equal(t, class2ID, updatedProblem.ClassID)
	assert.Equal(t, "hard", *updatedProblem.Difficulty)
	assert.Equal(t, "Graphs", *updatedProblem.Topic)
	assert.NotNil(t, updatedProblem.Priority)
	assert.Equal(t, "recommended", *updatedProblem.Priority)
	assert.Equal(t, originalProblem.CreatedAt, updatedProblem.CreatedAt, "CreatedAt must remain unchanged")
	assert.Equal(t, "new_url", updatedProblem.URL)
	assert.Equal(t, problemID, updatedProblem.ID, "ID must remain unchanged")
}

func TestIntegration_Mentor_UnauthorizedAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		handler           func(*context.ContextMessage)
		args              []string
		permissionMessage string
	}{
		{
			name: "AddAC unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleAddAC(c)
			},
			args: []string{"!m_add_ac", "<@123>", "1"},
		},
		{
			name: "UndoAC unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleUndoAC(c)
			},
			args: []string{"!m_undo_ac", "<@123>", "1"},
		},
		{
			name:              "AddProblem unauthorized",
			permissionMessage: "mentor or admin",
			handler: func(c *context.ContextMessage) {
				mentor.HandleAddProblem(c)
			},
			args: []string{"!add_problem", "class_1", "url", "Easy", "Topic", "-", "Name"},
		},
		{
			name: "RemoveProblem unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleRemoveProblem(c)
			},
			args: []string{"!remove_problem", "1"},
		},
		{
			name:              "UpdateProblem unauthorized",
			permissionMessage: "mentor or admin",
			handler: func(c *context.ContextMessage) {
				mentor.HandleUpdateProblem(c)
			},
			args: []string{"!update_problem", "1", "class_1", "url", "Easy", "Topic", "-", "Name"},
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
			classID := env.SeedClass(t, classCode, "Protected")
			problemID := env.SeedProblem(t, "Stayin Alive", "url", classID, "Hard", "Safety")

			// Mock NOT mentor
			env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
				return false, nil
			}

			ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", tt.args)

			// ACT
			tt.handler(ctx)

			// ASSERT
			sentMsgs := env.Dispatcher.GetSentMessages()
			assert.NotEmpty(t, sentMsgs)
			permissionMessage := tt.permissionMessage
			if permissionMessage == "" {
				permissionMessage = "is not mentor"
			}
			assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, permissionMessage)

			// Verify DB side-effects: problem should still exist
			_, err := env.DB.GetProblemByID(problemID)
			assert.NoError(t, err, "DB state should not change after unauthorized access")
		})
	}
}
