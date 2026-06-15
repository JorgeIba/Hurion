package workflow

import (
	"testing"

	appbot "github.com/JorgeIba/Hurion/app/bot"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestWorkflow_Whitelist_FiltersMessagesInTestMode(t *testing.T) {
	t.Parallel()

	whitelistedUser := "user_allowed"
	blockedUser := "user_blocked"
	whitelistedRole := "role_allowed"

	// 1. Setup environment with TestMode = true
	env := setupWorkflowEnv(t, appbot.BotConfig{
		TestMode:         true,
		WhitelistUserIDs: []string{whitelistedUser},
		WhitelistRoleIDs: []string{whitelistedRole},
	})

	t.Run("Whitelisted User - Allowed", func(t *testing.T) {
		env.sendMessage(t, whitelistedUser, "Allowed User", "!help")
		// !help should trigger a response (Embed)
		msgs := env.Dispatcher.Messages()
		assert.NotEmpty(t, msgs, "Expected a response for whitelisted user")
	})

	t.Run("Blocked User without Role - Ignored", func(t *testing.T) {
		// Clear previous messages
		env.Dispatcher.mu.Lock()
		env.Dispatcher.messages = nil
		env.Dispatcher.mu.Unlock()

		env.sendMessage(t, blockedUser, "Blocked User", "!help")
		msgs := env.Dispatcher.Messages()
		assert.Empty(t, msgs, "Expected NO response for blocked user")
	})

	t.Run("Blocked User with Whitelisted Role - Allowed", func(t *testing.T) {
		// Clear previous messages
		env.Dispatcher.mu.Lock()
		env.Dispatcher.messages = nil
		env.Dispatcher.mu.Unlock()

		// Manually trigger HandleMessageEvent with custom member to simulate role
		msg := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "msg_custom",
				Content:   "!help",
				ChannelID: testChannelID,
				GuildID:   testGuildID,
				Author:    &discordgo.User{ID: blockedUser, Username: "Role User"},
				Member: &discordgo.Member{
					User:  &discordgo.User{ID: blockedUser, Username: "Role User"},
					Nick:  "Role User",
					Roles: []string{whitelistedRole},
				},
			},
		}
		env.Bot.HandleMessageEvent(env.Session, msg)

		msgs := env.Dispatcher.Messages()
		assert.NotEmpty(t, msgs, "Expected response for user with whitelisted role")
	})

	t.Run("Blocked User with Whitelisted Role (Empty in Event, Present in Cache) - Allowed", func(t *testing.T) {
		// Clear previous messages
		env.Dispatcher.mu.Lock()
		env.Dispatcher.messages = nil
		env.Dispatcher.mu.Unlock()

		// 1. Set roles in cache
		env.setRoles(blockedUser, []string{whitelistedRole})

		// 2. Trigger message with Member present but Roles empty
		msg := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "msg_empty_roles",
				Content:   "!help",
				ChannelID: testChannelID,
				GuildID:   testGuildID,
				Author:    &discordgo.User{ID: blockedUser, Username: "Cached Role User"},
				Member: &discordgo.Member{
					User:  &discordgo.User{ID: blockedUser, Username: "Cached Role User"},
					Nick:  "Cached Role User",
					Roles: []string{}, // Empty!
				},
			},
		}
		env.Bot.HandleMessageEvent(env.Session, msg)

		msgs := env.Dispatcher.Messages()
		assert.NotEmpty(t, msgs, "Expected response for user with roles in cache despite empty roles in event")
	})
}

func TestWorkflow_Whitelist_FiltersReactionsInTestMode(t *testing.T) {
	t.Parallel()

	whitelistedUser := "user_allowed"
	blockedUser := "user_blocked"

	env := setupWorkflowEnv(t, appbot.BotConfig{
		TestMode:         true,
		WhitelistUserIDs: []string{whitelistedUser},
	})

	t.Run("Whitelisted User Reaction - Processed", func(t *testing.T) {
		// Seed a confirmation message and its referenced message into the mock discord provider
		msgID := "msg_123"
		origMsgID := "orig_123"
		env.Discord.storeMessage(&discordgo.Message{
			ID:        origMsgID,
			ChannelID: testChannelID,
			Author:    &discordgo.User{ID: whitelistedUser},
			Content:   "!ac problem_1",
		})
		env.Discord.storeMessage(newConfirmationMessage(msgID, origMsgID, []string{"!ac", "confirm"}, whitelistedUser))

		reaction := &discordgo.MessageReactionAdd{
			MessageReaction: &discordgo.MessageReaction{
				UserID:    whitelistedUser,
				MessageID: msgID,
				ChannelID: testChannelID,
				GuildID:   testGuildID,
				Emoji:     discordgo.Emoji{Name: "👍"},
			},
			Member: &discordgo.Member{
				User:  &discordgo.User{ID: whitelistedUser, Username: "Whitelisted User"},
				Nick:  "Whitelisted User",
			},
		}
		
		env.Bot.HandleReactionEvent(env.Session, reaction)
		// If processed, it would try to remove reactions. We verify it didn't return early.
		// Since we didn't fully seed everything for !ac confirm, we at least check it attempted to call Discord
		assert.NotEmpty(t, env.Discord.removedReactionEmojis(), "Expected reaction processing to reach Discord calls")
	})

	t.Run("Blocked User Reaction - Ignored", func(t *testing.T) {
		// Reset removed reactions
		env.Discord.removedReactions = nil

		reaction := &discordgo.MessageReactionAdd{
			MessageReaction: &discordgo.MessageReaction{
				UserID:    blockedUser,
				MessageID: "msg_456",
				ChannelID: testChannelID,
				GuildID:   testGuildID,
				Emoji:     discordgo.Emoji{Name: "👍"},
			},
			Member: &discordgo.Member{
				User:  &discordgo.User{ID: blockedUser, Username: "Blocked User"},
				Nick:  "Blocked User",
			},
		}

		env.Bot.HandleReactionEvent(env.Session, reaction)
		assert.Empty(t, env.Discord.removedReactionEmojis(), "Expected reaction from blocked user to be ignored")
	})
}
