package dispatcher

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingDiscordInteractionClient struct {
	err error
}

func (client failingDiscordInteractionClient) InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse, ...discordgo.RequestOption) error {
	return client.err
}

func (client failingDiscordInteractionClient) InteractionResponseDelete(*discordgo.Interaction, ...discordgo.RequestOption) error {
	return client.err
}

func (client failingDiscordInteractionClient) InteractionResponseEdit(*discordgo.Interaction, *discordgo.WebhookEdit, ...discordgo.RequestOption) (*discordgo.Message, error) {
	return nil, client.err
}

func (client failingDiscordInteractionClient) FollowupMessageCreate(*discordgo.Interaction, bool, *discordgo.WebhookParams, ...discordgo.RequestOption) (*discordgo.Message, error) {
	return nil, client.err
}

func TestDiscordInteractionResponderReturnsAndLogsEachDeliveryFailureOnce(t *testing.T) {
	logger.Init()
	previousHooks := logger.Log.ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() { logger.Log.ReplaceHooks(previousHooks) })
	hook := logrustest.NewLocal(logger.Log)
	failure := errors.New("Discord delivery failed")
	responder := NewDiscordInteractionResponder(failingDiscordInteractionClient{err: failure})
	interaction := &discordgo.Interaction{ID: "interaction_1", Token: "token_1"}

	for _, test := range []struct {
		name string
		send func() error
	}{
		{
			name: "initial response",
			send: func() error {
				return responder.Respond(interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal})
			},
		},
		{
			name: "private response",
			send: func() error { return responder.RespondPrivately(interaction, "Could not load the form.") },
		},
		{
			name: "private followup",
			send: func() error { return responder.FollowupPrivately(interaction, "Could not verify permissions.") },
		},
		{
			name: "delete original response",
			send: func() error { return responder.DeleteResponse(interaction) },
		},
		{
			name: "channel publication",
			send: func() error {
				_, err := responder.Publish(interaction, &discordgo.MessageSend{})
				return err
			},
		},
		{
			name: "edit original response",
			send: func() error {
				_, err := responder.EditResponse(interaction, &discordgo.WebhookEdit{})
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			hook.Reset()

			err := test.send()

			require.ErrorIs(t, err, failure)
			entries := hook.AllEntries()
			require.Len(t, entries, 1)
			assert.Equal(t, logrus.ErrorLevel, entries[0].Level)
			assert.Contains(t, entries[0].Message, failure.Error())
			assert.Contains(t, entries[0].Message, interaction.ID)
		})
	}
}

type recordingFollowupClient struct {
	failingDiscordInteractionClient
	interaction *discordgo.Interaction
	wait        bool
	response    *discordgo.WebhookParams
}

func (client *recordingFollowupClient) FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, response *discordgo.WebhookParams, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	client.interaction, client.wait, client.response = interaction, wait, response
	return &discordgo.Message{Content: response.Content}, nil
}

func TestPrivateFollowupUsesEphemeralMessageWithoutMentions(t *testing.T) {
	t.Parallel()
	client := &recordingFollowupClient{}
	responder := NewDiscordInteractionResponder(client)
	interaction := &discordgo.Interaction{ID: "acknowledged-component"}
	require.NoError(t, responder.FollowupPrivately(interaction, "Could not verify permissions."))
	assert.Same(t, interaction, client.interaction)
	assert.True(t, client.wait)
	require.NotNil(t, client.response)
	assert.Equal(t, "Could not verify permissions.", client.response.Content)
	assert.Equal(t, discordgo.MessageFlagsEphemeral, client.response.Flags)
	require.NotNil(t, client.response.AllowedMentions)
	assert.Empty(t, client.response.AllowedMentions.Parse)
}

func (client failingDiscordInteractionClient) ChannelMessageSendComplex(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
	return nil, client.err
}
