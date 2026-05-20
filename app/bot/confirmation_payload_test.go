package bot

import (
	"os"
	"strings"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/dispatcher/mock_dispatcher"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/providers/mock_providers"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

func TestGetConfirmationArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		command    string
		args       []string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(t *testing.T, got []string, ok bool)
	}{
		{
			name:    "copies explicit args for regular confirmable command",
			command: ExterminateClass.String(),
			args:    []string{"!exterminate_class", "class_a"},
			setupMock: func(m *mock_data.MockDataStore) {
			},
			assertFunc: func(t *testing.T, got []string, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, []string{"!exterminate_class", "class_a"}, got)
			},
		},
		{
			name:    "copies explicit clear class args without requiring class existence",
			command: ClearClass.String(),
			args:    []string{"!clear_class", "missing_class"},
			setupMock: func(m *mock_data.MockDataStore) {
			},
			assertFunc: func(t *testing.T, got []string, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, []string{"!clear_class", "missing_class"}, got)
			},
		},
		{
			name:    "resolves contextual clear class args",
			command: ClearClass.String(),
			args:    []string{"!clear_class"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("channel_1").Return("class_a", nil)
			},
			assertFunc: func(t *testing.T, got []string, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, []string{"!clear_class", "class_a"}, got)
			},
		},
		{
			name:    "rejects contextual clear class without room mapping",
			command: ClearClass.String(),
			args:    []string{"!clear_class"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("channel_1").Return("", &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, got []string, ok bool) {
				assert.False(t, ok)
				assert.Nil(t, got)
			},
		},
		{
			name:    "rejects unknown confirmable command",
			command: "!unknown",
			args:    []string{"!unknown"},
			setupMock: func(m *mock_data.MockDataStore) {
			},
			assertFunc: func(t *testing.T, got []string, ok bool) {
				assert.False(t, ok)
				assert.Nil(t, got)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			ctx := confirmationPayloadContext(mockStore, nil, tt.args)

			got, ok := GetConfirmationArgs(ctx, tt.command)

			tt.assertFunc(t, got, ok)
		})
	}
}

func TestRouterClearClassConfirmationPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(t *testing.T, sent dispatcher.MessageResponse)
	}{
		{
			name: "contextual clear class sends canonical confirmation args",
			args: []string{"!clear_class"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("channel_1").Return("class_a", nil)
			},
			assertFunc: func(t *testing.T, sent dispatcher.MessageResponse) {
				assert.Equal(t, dispatcher.ConfirmationMessage, sent.Type)
				assert.Equal(t, []string{"!clear_class", "class_a"}, sent.ConfirmationArgs)
				assert.Contains(t, sent.Content, "`class_a`")
			},
		},
		{
			name: "explicit clear class sends explicit confirmation args",
			args: []string{"!clear_class", "missing_class"},
			setupMock: func(m *mock_data.MockDataStore) {
			},
			assertFunc: func(t *testing.T, sent dispatcher.MessageResponse) {
				assert.Equal(t, dispatcher.ConfirmationMessage, sent.Type)
				assert.Equal(t, []string{"!clear_class", "missing_class"}, sent.ConfirmationArgs)
				assert.Contains(t, sent.Content, "`missing_class`")
			},
		},
		{
			name: "contextual clear class without mapping sends usage error",
			args: []string{"!clear_class"},
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassCodeByRoomID("channel_1").Return("", &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, sent dispatcher.MessageResponse) {
				assert.Equal(t, dispatcher.SimpleMessage, sent.Type)
				assert.Contains(t, sent.Content, "Invalid usage")
				assert.Empty(t, sent.ConfirmationArgs)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockSender := mock_dispatcher.NewMockMessageSender(ctrl)
			tt.setupMock(mockStore)

			var sent dispatcher.MessageResponse
			mockSender.EXPECT().
				Send(gomock.Any()).
				Do(func(msg dispatcher.MessageResponse) {
					sent = msg
				})

			ctx := confirmationPayloadContext(mockStore, mockSender, tt.args)

			routerClearClass(ctx)

			tt.assertFunc(t, sent)
		})
	}
}

func TestConfirmationArgsFromMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		msg        *discordgo.Message
		assertFunc func(t *testing.T, got []string)
	}{
		{
			name: "reads args from command footer",
			msg:  confirmationMessageWithFooter(dispatcher.ConfirmationCommandFooterPrefix + "!clear_class class_a"),
			assertFunc: func(t *testing.T, got []string) {
				assert.Equal(t, []string{"!clear_class", "class_a"}, got)
			},
		},
		{
			name: "returns nil when footer is missing",
			msg:  confirmationMessageWithFooter(""),
			assertFunc: func(t *testing.T, got []string) {
				assert.Nil(t, got)
			},
		},
		{
			name: "returns nil when prefix is invalid",
			msg:  confirmationMessageWithFooter("Args: !clear_class class_a"),
			assertFunc: func(t *testing.T, got []string) {
				assert.Nil(t, got)
			},
		},
		{
			name: "returns nil when command is empty",
			msg:  confirmationMessageWithFooter(dispatcher.ConfirmationCommandFooterPrefix),
			assertFunc: func(t *testing.T, got []string) {
				assert.Nil(t, got)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confirmationArgsFromMessage(tt.msg)

			tt.assertFunc(t, got)
		})
	}
}

func TestConfirmationYesUsesCapturedArgsAfterRoomRelink(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockStore := mock_data.NewMockDataStore(ctrl)
	mockSender := mock_dispatcher.NewMockMessageSender(ctrl)
	mockValidator := mock_providers.NewMockDiscordValidator(ctrl)
	mockFetcher := mock_providers.NewMockDiscordFetcher(ctrl)
	mockModifier := mock_providers.NewMockDiscordMessageModifier(ctrl)

	var capturedArgs []string
	mockStore.EXPECT().GetClassCodeByRoomID("channel_1").Return("class_a", nil)
	mockSender.EXPECT().
		Send(gomock.Any()).
		Do(func(msg dispatcher.MessageResponse) {
			capturedArgs = msg.ConfirmationArgs
		})

	commandCtx := confirmationPayloadContext(mockStore, mockSender, []string{"!clear_class"})
	routerClearClass(commandCtx)
	require.Equal(t, []string{"!clear_class", "class_a"}, capturedArgs)

	confirmationMsg, originalMsg := confirmationMessagesForArgs(capturedArgs)
	mockFetcher.EXPECT().FetchMessage("channel_1", "confirmation_1").Return(confirmationMsg, nil).Times(2)
	mockFetcher.EXPECT().FetchMessage("channel_1", "original_1").Return(originalMsg, nil)
	mockValidator.EXPECT().IsAdmin("guild_1", "admin_1").Return(true, nil)
	mockStore.EXPECT().GetClassByClassCode("class_a").Return(data.Class{ID: 1, ClassCode: "class_a"}, nil)
	mockStore.EXPECT().GetClassByClassCode("class_a").Return(data.Class{ID: 1, ClassCode: "class_a"}, nil)
	mockStore.EXPECT().ClearClassContents(int64(1)).Return(nil)
	mockSender.EXPECT().Send(gomock.Any())
	mockFetcher.EXPECT().
		GetCachedMember("guild_1", "admin_1").
		Return(&discordgo.Member{User: &discordgo.User{ID: "admin_1", GlobalName: "Admin User"}}, nil)
	mockModifier.EXPECT().
		EditEmbed("channel_1", "confirmation_1", gomock.Any()).
		Do(func(channelID, messageID string, embed *discordgo.MessageEmbed) {
			assert.Equal(t, "EXECUTED", embed.Title)
		})
	mockModifier.EXPECT().RemoveBotReaction("channel_1", "confirmation_1", common.REACTION_THUMBS_UP).Return(nil)
	mockModifier.EXPECT().RemoveBotReaction("channel_1", "confirmation_1", common.REACTION_THUMBS_DOWN).Return(nil)

	reactionCtx := confirmationReactionContext(mockStore, mockSender, mockValidator, mockFetcher, mockModifier, common.REACTION_THUMBS_UP)

	routerConfirmationReaction(reactionCtx)
}

func TestConfirmationYesErrorsWhenCanonicalArgsAreMissingOrInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		footerText string
		assertFunc func(t *testing.T, embed *discordgo.MessageEmbed)
	}{
		{
			name:       "missing footer",
			footerText: "",
			assertFunc: func(t *testing.T, embed *discordgo.MessageEmbed) {
				assert.Equal(t, "ERROR", embed.Title)
			},
		},
		{
			name:       "invalid footer",
			footerText: "Args: !clear_class class_a",
			assertFunc: func(t *testing.T, embed *discordgo.MessageEmbed) {
				assert.Equal(t, "ERROR", embed.Title)
			},
		},
		{
			name:       "empty command footer",
			footerText: dispatcher.ConfirmationCommandFooterPrefix,
			assertFunc: func(t *testing.T, embed *discordgo.MessageEmbed) {
				assert.Equal(t, "ERROR", embed.Title)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockStore := mock_data.NewMockDataStore(ctrl)
			mockSender := mock_dispatcher.NewMockMessageSender(ctrl)
			mockValidator := mock_providers.NewMockDiscordValidator(ctrl)
			mockFetcher := mock_providers.NewMockDiscordFetcher(ctrl)
			mockModifier := mock_providers.NewMockDiscordMessageModifier(ctrl)

			confirmationMsg, _ := confirmationMessagesForArgs([]string{"!clear_class", "class_a"})
			confirmationMsg.Embeds[0].Footer = footerForText(tt.footerText)
			mockFetcher.EXPECT().FetchMessage("channel_1", "confirmation_1").Return(confirmationMsg, nil).Times(2)
			mockModifier.EXPECT().
				EditEmbed("channel_1", "confirmation_1", gomock.Any()).
				Do(func(channelID, messageID string, embed *discordgo.MessageEmbed) {
					tt.assertFunc(t, embed)
				})
			mockModifier.EXPECT().RemoveBotReaction("channel_1", "confirmation_1", common.REACTION_THUMBS_UP).Return(nil)
			mockModifier.EXPECT().RemoveBotReaction("channel_1", "confirmation_1", common.REACTION_THUMBS_DOWN).Return(nil)

			reactionCtx := confirmationReactionContext(mockStore, mockSender, mockValidator, mockFetcher, mockModifier, common.REACTION_THUMBS_UP)

			routerConfirmationReaction(reactionCtx)
		})
	}
}

func confirmationPayloadContext(store data.DataStore, sender dispatcher.MessageSender, args []string) *context.ContextMessage {
	return &context.ContextMessage{
		Message: &discordgo.MessageCreate{Message: &discordgo.Message{
			ID:        "original_1",
			ChannelID: "channel_1",
			GuildID:   "guild_1",
			Author:    &discordgo.User{ID: "admin_1"},
		}},
		Args:            args,
		Dispatcher:      sender,
		Services:        services.NewServices(store),
		Executor:        service_model.User{UserID: "admin_1", DisplayName: "Admin User"},
		ExecutorGuildID: "guild_1",
	}
}

func confirmationReactionContext(
	store data.DataStore,
	sender dispatcher.MessageSender,
	validator *mock_providers.MockDiscordValidator,
	fetcher *mock_providers.MockDiscordFetcher,
	modifier *mock_providers.MockDiscordMessageModifier,
	emoji string,
) *context.ContextMessage {
	return &context.ContextMessage{
		MessageReaction: &discordgo.MessageReactionAdd{MessageReaction: &discordgo.MessageReaction{
			UserID:    "admin_1",
			MessageID: "confirmation_1",
			ChannelID: "channel_1",
			GuildID:   "guild_1",
			Emoji:     discordgo.Emoji{Name: emoji},
		}},
		DiscordValidator:       validator,
		DiscordFetcher:         fetcher,
		DiscordMessageModifier: modifier,
		Dispatcher:             sender,
		Services:               services.NewServices(store),
		Executor:               service_model.User{UserID: "admin_1", DisplayName: "Admin User"},
		ExecutorGuildID:        "guild_1",
	}
}

func confirmationMessagesForArgs(args []string) (*discordgo.Message, *discordgo.Message) {
	original := &discordgo.Message{
		ID:        "original_1",
		ChannelID: "channel_1",
		GuildID:   "guild_1",
		Content:   "!clear_class",
		Author:    &discordgo.User{ID: "admin_1"},
	}
	confirmation := confirmationMessageWithFooter(dispatcher.ConfirmationCommandFooterPrefix + strings.Join(args, " "))
	confirmation.ID = "confirmation_1"
	confirmation.ChannelID = "channel_1"
	confirmation.ReferencedMessage = &discordgo.Message{
		ID:        original.ID,
		ChannelID: original.ChannelID,
		Author:    original.Author,
	}

	return confirmation, original
}

func confirmationMessageWithFooter(footerText string) *discordgo.Message {
	return &discordgo.Message{
		Embeds: []*discordgo.MessageEmbed{{
			Title:  "CONFIRMATION",
			Footer: footerForText(footerText),
		}},
	}
}

func footerForText(text string) *discordgo.MessageEmbedFooter {
	if text == "" {
		return nil
	}

	return &discordgo.MessageEmbedFooter{Text: text}
}
