package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services/helpers"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestAddProblemPrefixUsesServiceValidationBeforeSaving(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		wantSave bool
	}{
		{name: "unsupported URL is explained and not saved", url: "ftp://example.com/problem"},
		{name: "valid values are normalized and saved", url: " https://example.com/problem ", wantSave: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			classID := env.SeedClass(t, "basicos_1", "Basicos")
			env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
				return true, nil
			}
			ctx := env.NewContext("mentor_1", "Mentor", "guild_1", []string{
				"!add_problem", " basicos_1 ", tt.url, "-", "-", "-", " Two Sum ",
			})

			mentor.HandleAddProblem(ctx)

			problems, err := env.DB.GetProblemsByClass(classID)
			if !assert.NoError(t, err) {
				return
			}
			messages := env.Dispatcher.GetSentMessages()
			if !assert.NotEmpty(t, messages) {
				return
			}
			if !tt.wantSave {
				assert.Empty(t, problems)
				assert.Contains(t, messages[len(messages)-1].Content, "Please correct the values: URL must be an absolute HTTP or HTTPS URL")
				return
			}
			if !assert.Len(t, problems, 1) {
				return
			}
			assert.Equal(t, "Two Sum", problems[0].Name)
			assert.Equal(t, "https://example.com/problem", problems[0].URL)
			assert.Equal(t, classID, problems[0].ClassID)
			assert.Nil(t, problems[0].Difficulty)
			assert.Nil(t, problems[0].Topic)
			assert.Nil(t, problems[0].Priority)
			assert.Contains(t, messages[len(messages)-1].Content, "was added correctly")
		})
	}
}

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

	for _, tt := range []struct {
		name       string
		values     []string
		wantReason string
		clear      bool
	}{
		{name: "normalizes and moves while preserving identity and solved records", values: []string{" https://example.com/updated ", " HARD ", " Graphs ", " RECOMMENDED ", " New Name "}},
		{name: "clears optional values while moving", values: []string{"https://example.com/updated", "-", "-", "-", "New Name"}, clear: true},
		{name: "rejects an unsupported URL without moving or changing data", values: []string{"ftp://example.com/updated", "Hard", "Graphs", "recommended", "New Name"}, wantReason: "URL must be an absolute HTTP or HTTPS URL"},
		{name: "rejects a blank name without changing data", values: []string{"https://example.com/updated", "Hard", "Graphs", "recommended", " "}, wantReason: "problem name is required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := setupIntegrationEnv(t)
			classID := env.SeedClass(t, "class_1", "Class One")
			class2ID := env.SeedClass(t, "class_2", "Class Two")
			difficulty, topic, priority := "easy", "Arrays", "essential"
			problemID, err := env.DB.AddProblem(data.Problem{
				Name: "Old Name", URL: "https://example.com/original", ClassID: classID,
				Difficulty: &difficulty, Topic: &topic, Priority: &priority,
			})
			if !assert.NoError(t, err) {
				return
			}
			env.SeedUser(t, "student_1", "Student")
			env.SeedAC(t, "student_1", problemID)
			originalProblem, err := env.DB.GetProblemByID(problemID)
			if !assert.NoError(t, err) {
				return
			}
			originalSolved, err := env.DB.GetAllSolvedProblemsByClass(classID)
			if !assert.NoError(t, err) {
				return
			}
			env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) { return true, nil }
			args := append([]string{"!update_problem", fmt.Sprintf("%d", problemID), "class_2"}, tt.values...)

			mentor.HandleUpdateProblem(env.NewContext("mentor_1", "Mentor", "guild_1", args))

			updated, err := env.DB.GetProblemByID(problemID)
			if !assert.NoError(t, err) {
				return
			}
			messages := env.Dispatcher.GetSentMessages()
			if !assert.NotEmpty(t, messages) {
				return
			}
			if tt.wantReason != "" {
				assert.Equal(t, originalProblem, updated)
				assert.Contains(t, messages[len(messages)-1].Content, tt.wantReason)
			} else {
				assert.Equal(t, "New Name", updated.Name)
				assert.Equal(t, "https://example.com/updated", updated.URL)
				assert.Equal(t, class2ID, updated.ClassID)
				assert.Equal(t, problemID, updated.ID)
				assert.Equal(t, originalProblem.CreatedAt, updated.CreatedAt)
				if tt.clear {
					assert.Nil(t, updated.Difficulty)
					assert.Nil(t, updated.Topic)
					assert.Nil(t, updated.Priority)
				} else {
					assert.Equal(t, helpers.PtrStr("hard"), updated.Difficulty)
					assert.Equal(t, helpers.PtrStr("Graphs"), updated.Topic)
					assert.Equal(t, helpers.PtrStr("recommended"), updated.Priority)
				}
				assert.Contains(t, messages[len(messages)-1].Content, "updated successfully")
			}
			solved, err := env.DB.GetAllSolvedProblemsByClass(updated.ClassID)
			assert.NoError(t, err)
			assert.Equal(t, originalSolved, solved, "solved records and their timestamps must remain unchanged")
			if updated.ClassID == class2ID {
				oldSolved, err := env.DB.GetAllSolvedProblemsByClass(classID)
				assert.NoError(t, err)
				assert.Empty(t, oldSolved, "solved records follow the problem to its new class")
			}
		})
	}
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
			name:              "AddAC unauthorized",
			permissionMessage: "mentor or admin",
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
