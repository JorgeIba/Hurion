package integration

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_General_GetScoreByClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, classCode)
	env.SeedAC(t, student.UserID, problemID)

	ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!score", classCode})

	// 2. ACT
	general.HandleGetScoreByClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
	assert.NotEmpty(t, lastMsg.Files)
	assert.Equal(t, "table.png", lastMsg.Files[0].Name)
}

func TestIntegration_General_GetScoreByUserAndClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	classID := env.SeedClass(t, classCode, "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, classCode)
	env.SeedAC(t, student.UserID, problemID)

	// Stub fetcher for target user
	env.Fetcher.GetCachedMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
		return &discordgo.Member{
			User: &discordgo.User{ID: student.UserID, Username: student.DisplayName},
			Nick: student.DisplayName,
		}, nil
	}

	ctx := env.NewContext("444555666", "Another User", "guild_1", []string{"!score_user", classCode, fmt.Sprintf("<@%s>", student.UserID)})

	// 2. ACT
	general.HandleGetScoreByUserAndClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.FileMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embeds[0].Title, student.DisplayName)
}

func TestIntegration_General_GetUsersInClass(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classCode := "basicos_1"
	env.SeedClass(t, classCode, "Basicos 2026")

	student1 := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	student2 := service_model.User{UserID: "222333444", DisplayName: "Student Two"}
	env.SeedUser(t, student1.UserID, student1.DisplayName)
	env.SeedUser(t, student2.UserID, student2.DisplayName)
	env.SeedEnrollment(t, student1.UserID, classCode)
	env.SeedEnrollment(t, student2.UserID, classCode)

	// Stub fetcher
	env.Fetcher.GetCachedMemberFunc = func(guildID, userID string) (*discordgo.Member, error) {
		if userID == student1.UserID {
			return &discordgo.Member{User: &discordgo.User{ID: student1.UserID}, Nick: student1.DisplayName}, nil
		}
		return &discordgo.Member{User: &discordgo.User{ID: student2.UserID}, Nick: student2.DisplayName}, nil
	}

	ctx := env.NewContext("555666777", "User One", "guild_1", []string{"!users", classCode})

	// 2. ACT
	general.HandleGetUsersInClass(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embed.Description, student1.DisplayName)
	assert.Contains(t, lastMsg.Embed.Description, student2.DisplayName)
}

func TestIntegration_General_GetMyClasses(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	env.SeedClass(t, "class_1", "Class One")
	env.SeedClass(t, "class_2", "Class Two")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, "class_1")
	env.SeedEnrollment(t, student.UserID, "class_2")

	ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!my_classes"})
	// HandleGetMyClasses uses c.Message.Author.ID directly
	ctx.Message.Author.ID = student.UserID

	// 2. ACT
	general.HandleGetMyClasses(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embed.Description, "class_1")
	assert.Contains(t, lastMsg.Embed.Description, "class_2")
}

func TestIntegration_General_GetClassesOfUser(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	env.SeedClass(t, "class_1", "Class One")

	student := service_model.User{UserID: "111222333", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SeedEnrollment(t, student.UserID, "class_1")

	ctx := env.NewContext("444555666", "Another User", "guild_1", []string{"!user_classes", fmt.Sprintf("<@%s>", student.UserID)})

	// 2. ACT
	general.HandleGetClassesOfUser(ctx)

	// 3. ASSERT
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs)
	lastMsg := sentMsgs[len(sentMsgs)-1]
	assert.Equal(t, dispatcher.EmbedMessage, lastMsg.Type)
	assert.Contains(t, lastMsg.Embed.Title, student.UserID)
	assert.Contains(t, lastMsg.Embed.Description, "class_1")
}
