package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_Mentor_RegisterUser(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	targetUserID := "111222333"
	targetDisplayName := "Student One"
	// Stub fetcher for display name resolution via GetCachedMember
	env.Fetcher.GetCachedMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
		if userID == targetUserID {
			return &discordgo.Member{
				User: &discordgo.User{ID: targetUserID, Username: targetDisplayName},
				Nick: targetDisplayName,
			}, nil
		}
		return nil, fmt.Errorf("user not found")
	}

	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!m_register", fmt.Sprintf("<@%s>", targetUserID), classCode})

	// 2. ACT
	mentor.HandleRegisterUser(ctx)

	// 3. ASSERT
	env.AssertUserExists(t, targetUserID, targetDisplayName)
	env.AssertEnrollmentExists(t, targetUserID, classCode)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "has registered in the class")
}

func TestIntegration_Mentor_DeregisterUser(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, classCode)

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!m_deregister", fmt.Sprintf("<@%s>", student.UserID), classCode})

	// 2. ACT
	err := mentor.HandleDeregisterUser(ctx)
	assert.NoError(t, err)

	// 3. ASSERT
	env.AssertEnrollmentNotExists(t, student.UserID, classCode)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "has been deregistered")
}

func TestIntegration_Mentor_AddAC(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, "basicos_1")

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
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "has been AC-ed")
}

func TestIntegration_Mentor_UndoAC(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, "basicos_1")
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

	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	// !update_problem <id> <class_code> <url> <difficulty> <topic> <name...>
	ctx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", []string{"!update_problem", fmt.Sprintf("%d", problemID), "class_2", "new_url", "Hard", "Graphs", "New Name"})

	// 2. ACT
	mentor.HandleUpdateProblem(ctx)

	// 3. ASSERT
	updatedProblem, err := env.DB.GetProblemByID(problemID)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updatedProblem.Name)
	assert.Equal(t, class2ID, updatedProblem.ClassID)
	assert.Equal(t, "Hard", *updatedProblem.Difficulty)
	assert.Equal(t, "Graphs", *updatedProblem.Topic)
	assert.Equal(t, "new_url", updatedProblem.URL)

	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "updated successfully")
}

func TestIntegration_Mentor_UnauthorizedAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler func(*context.ContextMessage)
		args    []string
	}{
		{
			name: "RegisterUser unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleRegisterUser(c)
			},
			args: []string{"!m_register", "<@123>", "class_1"},
		},
		{
			name: "DeregisterUser unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleDeregisterUser(c)
			},
			args: []string{"!m_deregister", "<@123>", "class_1"},
		},
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
			name: "AddProblem unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleAddProblem(c)
			},
			args: []string{"!add_problem", "class_1", "url", "Easy", "Topic", "Name"},
		},
		{
			name: "RemoveProblem unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleRemoveProblem(c)
			},
			args: []string{"!remove_problem", "1"},
		},
		{
			name: "UpdateProblem unauthorized",
			handler: func(c *context.ContextMessage) {
				mentor.HandleUpdateProblem(c)
			},
			args: []string{"!update_problem", "1", "class_1", "url", "Easy", "Topic", "Name"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			student := service_model.User{UserID: "student_1", DisplayName: "Student"}

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
			assert.Contains(t, sentMsgs[len(sentMsgs)-1].Content, "is not mentor")
		})
	}
}
