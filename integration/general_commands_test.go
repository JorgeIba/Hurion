package integration

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/general"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_RegisterUser(t *testing.T) {
	t.Parallel()

	// 1. SETUP Environment
	env := setupIntegrationEnv(t)
	env.SeedClass(t, "basicos_1", "Basicos 2026")

	// Define test user
	testUser := service_model.User{
		UserID:      "123456789",
		DisplayName: "Jorge el Pro",
	}

	// Mocking Discord context
	env.Ctx.Message = &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ChannelID: "channel_123",
			Author: &discordgo.User{
				ID:       testUser.UserID,
				Username: testUser.DisplayName,
			},
		},
	}
	env.Ctx.Args = []string{"!register", "basicos_1"}
	env.Ctx.Executor = testUser

	// 2. ACT
	general.HandleRegisterUserInClass(env.Ctx)

	// 3. ASSERT Dispatcher
	assert.Len(t, env.Dispatcher.SentMessages, 1)
	msg := env.Dispatcher.SentMessages[0]
	assert.Equal(t, dispatcher.SimpleMessage, msg.Type)
	assert.Contains(t, msg.Content, "User `Jorge el Pro` has registered in the class `basicos_1`")

	// 4. ASSERT Database using DSL
	env.AssertUserExists(t, testUser.UserID, testUser.DisplayName)
	env.AssertEnrollmentExists(t, testUser.UserID, "basicos_1")
}

func TestIntegration_ListProblems(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	env.SeedProblem(t, "Two Sum", "https://leetcode.com/problems/two-sum/", classID, "Easy", "Arrays")
	env.SeedProblem(t, "Add Two Numbers", "https://leetcode.com/problems/add-two-numbers/", classID, "Medium", "Linked Lists")

	env.Ctx.Message = &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ChannelID: "channel_123",
		},
	}
	env.Ctx.Args = []string{"!problems", "basicos_1"}

	// 2. ACT
	general.HandleGetListOfProblemsByClass(env.Ctx)

	// 3. ASSERT Dispatcher
	assert.Len(t, env.Dispatcher.SentMessages, 1)
	msg := env.Dispatcher.SentMessages[0]
	assert.Equal(t, dispatcher.PaginationEmbed, msg.Type)
	assert.Len(t, msg.Embeds, 1)
	assert.Contains(t, msg.Embeds[0].Title, "basicos_1")
	
	// Instead of hardcoding Fields[2], we find the field with Name == "Name"
	var nameColumnValue string
	for _, f := range msg.Embeds[0].Fields {
		if f.Name == "Name" {
			nameColumnValue = f.Value
			break
		}
	}
	assert.NotEmpty(t, nameColumnValue, "Expected an embed field with name 'Name'")
	assert.Contains(t, nameColumnValue, "Two Sum")
	assert.Contains(t, nameColumnValue, "Add Two Numbers")
}

