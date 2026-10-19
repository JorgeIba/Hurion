package interactiontransport_test

import (
	"encoding/json"
	"testing"

	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
)

func TestParseACProposalOptionsFromDiscordPayload(t *testing.T) {
	for _, withScreenshot := range []bool{false, true} {
		name := "without screenshot"
		if withScreenshot {
			name = "with screenshot"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var command discordgo.ApplicationCommandInteractionData
			require.NoError(t, json.Unmarshal([]byte(`{
				"name":"ac",
				"options":[{"name":"submit","type":1,"options":[
					{"name":"id","type":3,"value":"5"}
				]}]
			}`), &command))
			var screenshot *discordgo.MessageAttachment
			if withScreenshot {
				screenshot = &discordgo.MessageAttachment{ID: "file", Filename: "proof.png"}
				command.Options[0].Options = append(command.Options[0].Options, &discordgo.ApplicationCommandInteractionDataOption{
					Name:  "screenshot",
					Type:  discordgo.ApplicationCommandOptionAttachment,
					Value: "file",
				})
				command.Resolved = &discordgo.ApplicationCommandInteractionDataResolved{
					Attachments: map[string]*discordgo.MessageAttachment{"file": screenshot},
				}
			}
			id, attachment, err := interactiontransport.ParseACProposalOptions(command)
			require.NoError(t, err)
			require.Equal(t, int64(5), id)
			if withScreenshot {
				require.Same(t, screenshot, attachment)
			} else {
				require.Nil(t, attachment)
			}
		})
	}
}

func TestParseACProposalOptionsRejectsInvalidInputWithoutPanic(t *testing.T) {
	for _, test := range []struct {
		name    string
		options string
	}{
		{name: "missing subcommand", options: `[]`},
		{name: "null subcommand", options: `[null]`},
		{name: "wrong subcommand", options: `[{"name":"add_batch","type":1}]`},
		{name: "missing ID", options: `[{"name":"submit","type":1,"options":[]}]`},
		{name: "wrong option type", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":4,"value":5}]}]`},
		{name: "unexpected value type", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":5}]}]`},
		{name: "missing value", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3}]}]`},
		{name: "zero ID", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":"0"}]}]`},
		{name: "negative ID", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":"-1"}]}]`},
		{name: "name without selecting a suggestion", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":"Two Sum"}]}]`},
		{name: "decimal ID", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":"1.5"}]}]`},
		{name: "empty ID", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":""}]}]`},
		{name: "ID outside int64 range", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":"9223372036854775808"}]}]`},
		{name: "unresolved screenshot", options: `[{"name":"submit","type":1,"options":[{"name":"id","type":3,"value":"5"},{"name":"screenshot","type":11,"value":"missing"}]}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var command discordgo.ApplicationCommandInteractionData
			require.NoError(t, json.Unmarshal([]byte(`{"name":"ac","options":`+test.options+`}`), &command))
			require.NotPanics(t, func() {
				id, attachment, err := interactiontransport.ParseACProposalOptions(command)
				require.Error(t, err)
				require.Zero(t, id)
				require.Nil(t, attachment)
			})
		})
	}
}
