package integration

import (
	"strconv"
	"testing"

	"github.com/JorgeIba/Hurion/app/common"
	"github.com/JorgeIba/Hurion/app/controllers/enrolled"
	"github.com/JorgeIba/Hurion/app/controllers/mentor"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_AC_Propose(t *testing.T) {
	t.Parallel()

	// 1. SETUP
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")
	env.SeedClassAccessRole(t, "basicos_1", "role_enrolled")

	student := service_model.User{UserID: "student_1", DisplayName: "Student One"}
	env.SeedUser(t, student.UserID, student.DisplayName)
	env.SetRoles(student.UserID, []string{"role_enrolled"})

	ctx := env.NewContext(student.UserID, student.DisplayName, "guild_1", []string{"!ac", strconv.FormatInt(problemID, 10)})
	ctx.Message.ID = "msg_propose_123"

	// 2. ACT
	enrolled.HandleProposeAC(ctx)

	// 3. ASSERT: Reaction request sent
	sentMsgs := env.Dispatcher.GetSentMessages()
	assert.NotEmpty(t, sentMsgs, "expected at least one sent message")
	if len(sentMsgs) > 0 {
		lastMsg := sentMsgs[len(sentMsgs)-1]
		assert.Equal(t, dispatcher.Reaction, lastMsg.Type)
		assert.Equal(t, "msg_propose_123", lastMsg.ReplyTo, "Must reply to the specific proposal message")
		assert.Contains(t, lastMsg.Reactions, common.REACTION_YES)
		assert.Contains(t, lastMsg.Reactions, common.REACTION_NO)
	}

	// 4. ASSERT: DB has pending record
	exists, err := env.Services.Solution.ExistsPendingConfirmation("msg_propose_123")
	assert.NoError(t, err)
	assert.True(t, exists, "Should have created a pending confirmation in DB")
}

func TestIntegration_AC_Accept(t *testing.T) {
	t.Parallel()

	// 1. SETUP: Manually seed the "pending" state
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")
	student := service_model.User{UserID: "student_1", DisplayName: "Student One"}
	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}

	env.SeedUser(t, student.UserID, student.DisplayName)

	// Pre-seed the pending confirmation
	messageID := "msg_existing_456"
	env.SeedPendingConfirmation(t, messageID, student.UserID, problemID)

	// Mock mentor permission
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	reactionCtx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", nil)
	reactionCtx.MessageReaction = &discordgo.MessageReactionAdd{
		MessageReaction: &discordgo.MessageReaction{
			UserID:    mentorUser.UserID,
			MessageID: messageID,
			ChannelID: "channel_1",
			GuildID:   "guild_1",
			Emoji:     discordgo.Emoji{Name: common.REACTION_YES},
		},
	}

	// 2. ACT
	mentor.HandlePendingConfirmation(reactionCtx)

	// 3. ASSERT
	env.AssertProblemSolved(t, student.UserID, problemID)

	sentMsgs := env.Dispatcher.GetSentMessages()
	found := false
	expected := common.REACTION_ACCEPTED
	for _, m := range sentMsgs {
		if m.Type == dispatcher.Reaction && len(m.Reactions) > 0 && m.Reactions[0] == expected {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected success reaction containing %q", expected)
}

func TestIntegration_AC_Reject(t *testing.T) {
	t.Parallel()

	// 1. SETUP: Pre-seed a pending state
	env := setupIntegrationEnv(t)
	classID := env.SeedClass(t, "basicos_1", "Basicos 2026")
	problemID := env.SeedProblem(t, "Two Sum", "url", classID, "Easy", "Arrays")
	student := service_model.User{UserID: "student_1", DisplayName: "Student One"}
	mentorUser := service_model.User{UserID: "mentor_1", DisplayName: "Mentor One"}

	env.SeedUser(t, student.UserID, student.DisplayName)

	messageID := "msg_to_be_rejected_789"
	env.SeedPendingConfirmation(t, messageID, student.UserID, problemID)

	// Mock mentor permission
	env.Validator.IsMentorFunc = func(guildID, userID string) (bool, error) {
		return userID == mentorUser.UserID, nil
	}

	reactionCtx := env.NewContext(mentorUser.UserID, mentorUser.DisplayName, "guild_1", nil)
	reactionCtx.MessageReaction = &discordgo.MessageReactionAdd{
		MessageReaction: &discordgo.MessageReaction{
			UserID:    mentorUser.UserID,
			MessageID: messageID,
			ChannelID: "channel_1",
			GuildID:   "guild_1",
			Emoji:     discordgo.Emoji{Name: common.REACTION_NO},
		},
	}

	// 2. ACT
	mentor.HandlePendingConfirmation(reactionCtx)

	// 3. ASSERT
	env.AssertProblemNotSolved(t, student.UserID, problemID)

	// Verify DB: Record should be gone from pending_confirmations
	exists, err := env.Services.Solution.ExistsPendingConfirmation(messageID)
	assert.NoError(t, err)
	assert.False(t, exists, "Pending confirmation should be removed after rejection")

	sentMsgs := env.Dispatcher.GetSentMessages()
	found := false
	expected := common.REACTION_REJECTED
	for _, m := range sentMsgs {
		if m.Type == dispatcher.Reaction && len(m.Reactions) > 0 && m.Reactions[0] == expected {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected rejection reaction containing %q", expected)
}
